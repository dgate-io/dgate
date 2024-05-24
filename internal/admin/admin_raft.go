package admin

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path"
	"time"

	"github.com/dgate-io/chi-router"
	"github.com/dgate-io/dgate/internal/admin/changestate"
	"github.com/dgate-io/dgate/internal/config"
	"github.com/dgate-io/dgate/pkg/raftadmin"
	"github.com/dgate-io/dgate/pkg/rafthttp"
	"github.com/dgate-io/dgate/pkg/storage"
	"github.com/dgate-io/dgate/pkg/util/logger"
	raftbadgerdb "github.com/dgate-io/raft-badger"
	"github.com/hashicorp/raft"
)

func setupRaft(conf *config.DGateConfig, server *chi.Mux, cs changestate.ChangeState) {
	adminConfig := conf.AdminConfig
	var sstore raft.StableStore
	var lstore raft.LogStore
	snapstore := raft.NewInmemSnapshotStore()
	switch conf.Storage.StorageType {
	case config.StorageTypeMemory:
		sstore = raft.NewInmemStore()
		lstore = raft.NewInmemStore()
	case config.StorageTypeFile:
		fileConfig, err := config.StoreConfig[storage.FileStoreConfig](conf.Storage.Config)
		if err != nil {
			panic(fmt.Errorf("invalid config: %s", err))
		}
		badgerStore, err := raftbadgerdb.NewBadgerStore(
			path.Join(fileConfig.Directory, "raft"),
		)
		if err != nil {
			panic(err)
		}
		sstore = badgerStore
		lstore = badgerStore
	default:
		panic(fmt.Errorf("invalid storage type: %s", conf.Storage.StorageType))
	}
	raftId := adminConfig.Replication.RaftID

	raftConfig := adminConfig.Replication.LoadRaftConfig(
		&raft.Config{
			ProtocolVersion:    raft.ProtocolVersionMax,
			LocalID:            raft.ServerID(raftId),
			HeartbeatTimeout:   time.Second * 4,
			ElectionTimeout:    time.Second * 5,
			CommitTimeout:      time.Second * 4,
			BatchApplyCh:       true,
			MaxAppendEntries:   16,
			LeaderLeaseTimeout: time.Second * 4,
			// TODO: Support snapshots
			SnapshotInterval:  time.Hour * 24,
			SnapshotThreshold: ^uint64(0),
			Logger: logger.NewSLogHCAdapter(
				cs.Logger().WithGroup("admin-raft"),
			),
		},
	)
	advertAddr := adminConfig.Replication.AdvertAddr
	if advertAddr == "" {
		advertAddr = fmt.Sprintf("%s:%d", adminConfig.Host, adminConfig.Port)
	}
	address := raft.ServerAddress(advertAddr)

	raftHttpLogger := cs.Logger().WithGroup("raft-http")
	if adminConfig.Replication.AdvertScheme != "http" && adminConfig.Replication.AdvertScheme != "https" {
		panic(fmt.Errorf("invalid scheme: %s", adminConfig.Replication.AdvertScheme))
	}

	transport := rafthttp.NewHTTPTransport(
		address, http.DefaultClient, raftHttpLogger,
		adminConfig.Replication.AdvertScheme+"://(address)/raft",
	)
	raftNode, err := raft.NewRaft(
		raftConfig, newDGateAdminFSM(cs),
		lstore, sstore, snapstore, transport,
	)
	if err != nil {
		panic(err)
	}

	cs.SetupRaft(raftNode, raftConfig)
	// Setup raft handler
	server.Handle("/raft/*", transport)

	raftAdminLogger := cs.Logger().WithGroup("raft-admin")
	raftAdmin := raftadmin.NewRaftAdminHTTPServer(
		raftNode, raftAdminLogger, []raft.ServerAddress{address},
	)

	// Setup handler raft
	server.HandleFunc("/raftadmin/*", func(w http.ResponseWriter, r *http.Request) {
		if adminConfig.Replication.SharedKey != "" {
			sharedKey := r.Header.Get("X-DGate-Shared-Key")
			if sharedKey != adminConfig.Replication.SharedKey {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		raftAdmin.ServeHTTP(w, r)
	})

	configFuture := raftNode.GetConfiguration()

	if err = configFuture.Error(); err != nil {
		panic(err)
	}
	serverConfig := configFuture.Configuration()

	cs.Logger().With("config", adminConfig).
		Debug("Replication config")

	if adminConfig.Replication.BootstrapCluster {
		raftNode.BootstrapCluster(raft.Configuration{
			Servers: []raft.Server{
				{
					Suffrage: raft.Voter,
					ID:       raft.ServerID(raftId),
					Address:  raft.ServerAddress(adminConfig.Replication.AdvertAddr),
				},
			},
		})
	} else if len(serverConfig.Servers) == 0 {
		go func() {
			addresses := make([]string, 0)
			if adminConfig.Replication.DiscoveryDomain != "" {
				cs.Logger().Debug("no previous configuration found, attempting to discover cluster",
					"domain", adminConfig.Replication.DiscoveryDomain,
				)
				addrs, err := net.LookupHost(adminConfig.Replication.DiscoveryDomain)
				if err != nil {
					panic(err)
				}
				if len(addrs) == 0 {
					panic(fmt.Errorf("no addrs found for %s", adminConfig.Replication.DiscoveryDomain))
				}
				cs.Logger().Info("discovered addresses", "addresses", addrs)
				for _, addr := range addrs {
					if addr[len(addr)-1] == '.' {
						addr = addr[:len(addr)-1]
					}
					addresses = append(addresses, fmt.Sprintf("%s:%d", addr, adminConfig.Port))
				}
			}

			if adminConfig.Replication.ClusterAddrs != nil && len(adminConfig.Replication.ClusterAddrs) > 0 {
				addresses = append(addresses, adminConfig.Replication.ClusterAddrs...)
			}

			if len(addresses) > 0 {
				addresses = append(addresses, adminConfig.Replication.ClusterAddrs...)
				retries := 0
				doer := func(req *http.Request) (*http.Response, error) {
					req.Header.Set("User-Agent", "dgate")
					if adminConfig.Replication.SharedKey != "" {
						req.Header.Set("X-DGate-Shared-Key", adminConfig.Replication.SharedKey)
					}
					return http.DefaultClient.Do(req)
				}
				adminClient := raftadmin.NewHTTPAdminClient(doer,
					adminConfig.Replication.AdvertScheme+"://(address)/raftadmin",
					cs.Logger().WithGroup("raft-admin-client"),
				)
			RETRY:
				for _, url := range addresses {
					err = adminClient.VerifyLeader(context.Background(), raft.ServerAddress(url))
					if err != nil {
						if err == raftadmin.ErrNotLeader {
							continue
						}
						if retries > 15 {
							cs.Logger().Error("Skipping verifying leader at %s: %s", url, err)
							continue
						}
						retries += 1
						cs.Logger().Debug("Retrying verifying leader at %s: %s", url, err)
						<-time.After(3 * time.Second)
						goto RETRY
					}
					// If this node is watch only, add it as a non-voter node, otherwise add it as a voter node
					if adminConfig.WatchOnly {
						cs.Logger().Info("Adding non-voter", "url", url)
						resp, err := adminClient.AddNonvoter(
							context.Background(), raft.ServerAddress(url),
							&raftadmin.AddNonvoterRequest{
								ID:      raftId,
								Address: adminConfig.Replication.AdvertAddr,
							},
						)
						if err != nil {
							panic(err)
						}
						if resp.Error != "" {
							panic(resp.Error)
						}
					} else {
						cs.Logger().Info("Adding voter: %s - leader: %s",
							adminConfig.Replication.AdvertAddr, url)
						resp, err := adminClient.AddVoter(context.Background(), raft.ServerAddress(url), &raftadmin.AddVoterRequest{
							ID:      raftId,
							Address: adminConfig.Replication.AdvertAddr,
						})
						if err != nil {
							panic(err)
						}
						if resp.Error != "" {
							panic(resp.Error)
						}
					}
					break
				}
				if err != nil {
					panic(err)
				}
			} else {
				cs.Logger().Warn("no admin urls specified, waiting to be added to cluster")
			}
		}()
	} else {
		cs.Logger().Debug("previous configuration found",
			"servers", serverConfig.Servers)
	}
}

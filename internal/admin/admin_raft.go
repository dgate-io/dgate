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
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/dgate-io/dgate/pkg/storage"
	"github.com/dgate-io/dgate/pkg/util/logadapter"
	raftbadgerdb "github.com/dgate-io/raft-badger"
	"github.com/dgraph-io/badger/v4"
	"github.com/hashicorp/raft"
	"go.uber.org/zap"
)

func setupRaft(
	server *chi.Mux,
	logger *zap.Logger,
	conf *config.DGateConfig,
	cs changestate.ChangeState,
) {
	adminConfig := conf.AdminConfig
	var sstore raft.StableStore
	var lstore raft.LogStore
	switch conf.Storage.StorageType {
	case config.StorageTypeMemory:
		sstore = raft.NewInmemStore()
		lstore = raft.NewInmemStore()
	case config.StorageTypeFile:
		fileConfig, err := config.StoreConfig[storage.FileStoreConfig](conf.Storage.Config)
		if err != nil {
			panic(fmt.Errorf("invalid config: %s", err))
		}
		badgerLogger := logadapter.NewZap2BadgerAdapter(logger.Named("badger-file"))
		raftDir := path.Join(fileConfig.Directory, "raft")
		badgerStore, err := raftbadgerdb.New(
			badger.DefaultOptions(raftDir).
				WithLogger(badgerLogger),
		)
		if err != nil {
			panic(err)
		}
		sstore = badgerStore
		lstore = badgerStore
	default:
		panic(fmt.Errorf("invalid storage type: %s", conf.Storage.StorageType))
	}
	raftConfig := adminConfig.Replication.LoadRaftConfig(
		&raft.Config{
			ProtocolVersion:    raft.ProtocolVersionMax,
			LocalID:            raft.ServerID(adminConfig.Replication.RaftID),
			HeartbeatTimeout:   time.Second * 4,
			ElectionTimeout:    time.Second * 5,
			CommitTimeout:      time.Second * 4,
			BatchApplyCh:       false,
			MaxAppendEntries:   512,
			LeaderLeaseTimeout: time.Second * 4,
			// TODO: Support snapshots
			SnapshotInterval:  time.Hour*2 ^ 32,
			SnapshotThreshold: ^uint64(0),
			Logger:            logadapter.NewZap2HCLogAdapter(logger),
		},
	)

	advertAddr := adminConfig.Replication.AdvertAddr
	if advertAddr == "" {
		advertAddr = fmt.Sprintf("%s:%d", adminConfig.Host, adminConfig.Port)
	}
	address := raft.ServerAddress(advertAddr)

	raftHttpLogger := logger.Named("http")
	if adminConfig.Replication.AdvertScheme != "http" && adminConfig.Replication.AdvertScheme != "https" {
		panic(fmt.Errorf("invalid scheme: %s", adminConfig.Replication.AdvertScheme))
	}

	transport := rafthttp.NewHTTPTransport(
		address, http.DefaultClient, raftHttpLogger,
		adminConfig.Replication.AdvertScheme+"://(address)/raft",
	)
	fsmLogger := logger.Named("fsm")
	snapstore := raft.NewInmemSnapshotStore()
	fsm := newDGateAdminFSM(fsmLogger, cs)
	raftNode, err := raft.NewRaft(
		raftConfig, fsm, lstore,
		sstore, snapstore, transport,
	)
	if err != nil {
		panic(err)
	}

	observerChan := make(chan raft.Observation, 10)
	raftNode.RegisterObserver(raft.NewObserver(observerChan, false, nil))
	cs.SetupRaft(raftNode, observerChan)

	// Setup raft handler
	server.Handle("/raft/*", transport)

	raftAdminLogger := logger.Named("admin")
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
	raftId := string(raftConfig.LocalID)
	logger.Info("replication config",
		zap.String("raft_id", raftId),
		zap.Any("config", serverConfig),
		zap.Int("max_append_entries", raftConfig.MaxAppendEntries),
		zap.Bool("batch_chan", raftConfig.BatchApplyCh),
		zap.Duration("commit_timeout", raftConfig.CommitTimeout),
		zap.Int("config_proto", int(raftConfig.ProtocolVersion)),
	)

	defer cs.ProcessChangeLog(spec.NewNoopChangeLog(), false)

	if adminConfig.Replication.BootstrapCluster && len(serverConfig.Servers) == 0 {
		logger.Info("bootstrapping cluster",
			zap.String("id", raftId),
			zap.String("advert_addr", advertAddr),
		)
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
				logger.Debug("no previous configuration found, attempting to discover cluster",
					zap.String("domain", adminConfig.Replication.DiscoveryDomain),
				)
				addrs, err := net.LookupHost(adminConfig.Replication.DiscoveryDomain)
				if err != nil {
					panic(err)
				}
				if len(addrs) == 0 {
					panic(fmt.Errorf("no addrs found for %s", adminConfig.Replication.DiscoveryDomain))
				}
				logger.Info("discovered addresses",
					zap.Strings("addresses", addrs))
				for _, addr := range addrs {
					if addr[len(addr)-1] == '.' {
						addr = addr[:len(addr)-1]
					}
					addresses = append(addresses, fmt.Sprintf("%s:%d", addr, adminConfig.Port))
				}
			}
			logger.Info("no servers found in configuration, adding myself to cluster",
				zap.String("id", raftId),
				zap.String("advert_addr", advertAddr),
				zap.Strings("cluster_addrs", addresses),
			)

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
					logger.Named("raft-admin-client"),
				)
			RETRY:
				for _, url := range addresses {
					err = adminClient.VerifyLeader(context.Background(), raft.ServerAddress(url))
					if err != nil {
						if err == raftadmin.ErrNotLeader {
							continue
						}
						if retries > 15 {
							logger.Error("Skipping verifying leader",
								zap.String("url", url), zap.Error(err),
							)
							continue
						}
						retries += 1
						logger.Debug("Retrying verifying leader",
							zap.String("url", url), zap.Error(err))
						<-time.After(3 * time.Second)
						goto RETRY
					}
					// If this node is watch only, add it as a non-voter node, otherwise add it as a voter node
					if adminConfig.WatchOnly {
						logger.Info("Adding non-voter",
							zap.String("id", raftId),
							zap.String("leader", adminConfig.Replication.AdvertAddr),
							zap.String("url", url),
						)
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
						logger.Info("Adding voter: %s - leader: %s",
							zap.String("id", raftId),
							zap.String("leader", adminConfig.Replication.AdvertAddr),
							zap.String("url", url),
						)
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
				logger.Warn("no admin urls specified, waiting to be added to cluster")
			}
		}()
	} else {
		logger.Debug("previous configuration found",
			zap.Any("servers", serverConfig.Servers),
		)
	}
}

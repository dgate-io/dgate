package store

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"errors"

	"github.com/dgate-io/dgate/internal/config"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/dgate-io/dgate/pkg/storage"
	"github.com/dgate-io/raft"
	"github.com/dgraph-io/badger/v4"
	"go.uber.org/zap"
)

type StateManager interface {
	ProcessChangeLog(cl *spec.ChangeLog, reload bool) error
}

type Store struct {
	logger       *zap.Logger
	state        StateManager
	storage      storage.Storage
	raft         *raft.Raft
	transport    raft.Transport
	applyTimeout time.Duration
	leaderId     string
	leaderInfo   *ServerInfo
	myInfo       *ServerInfo
}

type ServerInfo struct {
	ID          string `json:"id"`
	Address     string `json:"address"`
	AdminAdvert string `json:"adminAdvert"`
}

func (i *ServerInfo) JSON() []byte {
	data, _ := json.Marshal(i)
	return data
}

func (i *ServerInfo) FromJSON(data []byte) error {
	return json.Unmarshal(data, i)
}

func NewStore(
	logger *zap.Logger,
	state StateManager,
	conf *config.DGateConfig,
) (store *Store, err error) {
	var dataStore storage.Storage
	switch conf.Storage.StorageType {
	case config.StorageTypeDebug:
		dataStore = storage.NewDebugStore(&storage.DebugStoreConfig{
			Logger: logger,
		})
	case config.StorageTypeMemory:
		dataStore = storage.NewMemoryStore(&storage.MemoryStoreConfig{
			Logger: logger,
		})
	case config.StorageTypeFile:
		fileConfig, err := config.StoreConfig[storage.FileStoreConfig](conf.Storage.Config)
		if err != nil {
			panic(fmt.Errorf("invalid config: %s", err))
		}
		fileConfig.Logger = logger
		dataStore = storage.NewFileStore(&fileConfig)
	default:
		panic(fmt.Errorf("invalid storage type: %s", conf.Storage.StorageType))
	}
	store = &Store{
		logger:  logger,
		state:   state,
		storage: dataStore,
	}

	if repl := conf.Storage.Replication; repl != nil {
		address := fmt.Sprintf("%s:%d", repl.Host, repl.Port)
		dataPath := ".dgate/v2"
		if conf.Storage.StorageType == config.StorageTypeMemory {
			// TODO: Implement a memory store or remove StorageTypeMemory
			dataPath = os.TempDir()
		}
		store.transport, err = raft.NewTransport(address)
		if err != nil {
			return nil, err
		}
		fsm := &FSM{state, store, logger.Named("fsm")}
		if store.raft, err = raft.NewRaft(
			repl.RaftID, repl.AdvertAddr, fsm, dataPath,
			repl.LoadRaftOptions(store.transport)...,
		); err != nil {
			return nil, err
		}
		store.applyTimeout = repl.ApplyTimeout
		if conf.AdminConfig != nil {
			store.myInfo = &ServerInfo{
				ID:          repl.RaftID,
				Address:     repl.AdvertAddr,
				AdminAdvert: conf.AdminConfig.AdvertAddr,
			}
		}
		members := store.raft.Configuration().Members
		if repl.BootstrapCluster && len(members) == 0 {
			logger.Debug("Bootstrapping cluster",
				zap.String("id", repl.RaftID),
				zap.Any("cluster", repl.ClusterAddrs),
				zap.String("advertAddr", repl.AdvertAddr),
			)
			config, err := parseCluster(repl.ClusterAddrs, repl.DiscoveryDomain, repl.Port)
			if err != nil {
				return nil, err
			}
			if _, ok := config[repl.RaftID]; !ok {
				config[repl.RaftID] = repl.AdvertAddr
			}
			logger.Debug("Bootstrap config", zap.Any("config", config))
			err = store.raft.Bootstrap(config)
			if err != nil {
				return nil, err
			}
		}
		store.raft.SetStateCallback(func(leaderId string, state raft.State) {
			logger.Info("Raft state changed",
				zap.String("leaderId", leaderId),
				zap.Stringer("state", state),
				zap.Bool("isLeader", leaderId == repl.RaftID),
			)
			store.leaderId = leaderId
		})
		go func() {
			if store.leaderId == "" {
				logger.Info("Waiting for raft state change")
				store.raft.WaitForStableState()
			} else {
				logger.Info("Leader is " + store.leaderId)
			}
			if store.leaderId == store.myInfo.ID {
				store.leaderInfo = store.myInfo
				store.logger.Info("I am the leader", zap.String("id", repl.RaftID))

				req := &ReadRequest{
					Key:  "leaderInfo",
					Data: store.myInfo.JSON(),
				}
				bytes, err := req.Marshal()
				if err != nil {
					return
				}
				results := store.raft.SubmitOperation(
					bytes,
					raft.Broadcasted,
					store.applyTimeout,
				).Await()
				if results.Error() != nil {
					// return nil, results.Error()
					return
				}
				if results.Success().ApplicationResponse != nil {
					response := results.Success().
						ApplicationResponse.(*ApplyResponse)
					store.logger.Info("leader info response",
						zap.Any("response", response),
					)
				}
				store.logger.Info("leader info sent",
					zap.Any("leaderInfo", store.leaderInfo),
				)
			} else {
				store.logger.Info("I am a follower",
					zap.String("id", repl.RaftID),
					zap.String("leader", store.raft.LeaderID()),
					zap.Stringer("state", store.raft.Status().State),
				)
				// req := &ReadRequest{
				// 	Key: "leaderInfo",
				// }
				// results := store.raft.SubmitOperation(
				// 	req.Marshal(),
				// 	raft.LeaseBasedReadOnly,
				// 	store.applyTimeout,
				// ).Await()
				// if results.Error() != nil {
				// 	return nil, results.Error()
				// }
				// if results.Success().ApplicationResponse == nil {
				// 	return nil, errors.New("no leader info")
				// }
				// response := results.Success().ApplicationResponse.(*ReadResponse)
				// store.leaderInfo = &ServerInfo{}
				// err = store.leaderInfo.FromJSON(response.Data)
				// if err != nil {
				// 	return nil, err
				// }
			}
		}()
		if err = store.raft.Start(); err != nil {
			return nil, err
		}
	}

	return store, nil
}

func parseCluster(
	addresses map[string]string,
	discoveryDomain string,
	defaultPort int,
) (map[string]string, error) {
	if discoveryDomain != "" {
		ips, err := net.LookupIP(discoveryDomain)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			host := ip.String()
			hosts, err := net.LookupHost(ip.String())
			if err == nil {
				host = strings.TrimSuffix(shortestString(
					ip.String(), hosts...,
				), ".")
			}
			addresses[host] = fmt.Sprintf("%s:%d", host, defaultPort)
		}
	}
	return addresses, nil
}

func shortestString(def string, strs ...string) string {
	if len(strs) == 0 {
		return def
	}
	shortest := strs[0]
	for _, str := range strs {
		if len(str) < len(shortest) {
			shortest = str
		}
	}
	return shortest
}

func (s *Store) Start() error {
	if s.raft != nil {
		return s.raft.Start()
	} else {
		panic("unimplemented")
	}
}

func (store *Store) InitStore() error {
	err := store.storage.Connect()
	if err != nil {
		return err
	}
	return nil
}

func (store *Store) ReplicationEnabled() bool {
	return store.raft != nil
}

func (store *Store) IsLeader() bool {
	if store.raft != nil {
		return store.raft.Status().State == raft.Leader
	}
	return false
}

func (store *Store) LeaderAdminAddr() string {
	if store.leaderInfo == nil {
		return ""
	}
	if store.leaderInfo.ID == store.raft.LeaderID() {
		return store.leaderInfo.AdminAdvert
	}
	return ""
}

func (store *Store) FetchChangeLogs() ([]*spec.ChangeLog, error) {
	clBytes, err := store.storage.GetPrefix("changelog/", 0, -1)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, nil
		}
		return nil, errors.New("failed to fetch changelog" + err.Error())
	}
	if len(clBytes) == 0 {
		return nil, nil
	}
	store.logger.Debug("found changelog entries", zap.Int("numBytes", len(clBytes)))
	logs := make([]*spec.ChangeLog, len(clBytes))
	for i, clKv := range clBytes {
		var clObj spec.ChangeLog
		err = json.Unmarshal(clKv.Value, &clObj)
		if err != nil {
			store.logger.Debug("failed to unmarshal changelog entry", zap.Error(err))
			return nil, errors.New("failed to unmarshal changelog entry: " + err.Error())
		}
		logs[i] = &clObj
	}

	return logs, nil
}

func (store *Store) StoreChangeLog(cl *spec.ChangeLog) error {
	if store.raft != nil {
		if store.raft.Status().State != raft.Leader {
			return errors.New("this node is not the leader")
		} else {
			clBytes, err := cl.JSONBytes()
			if err != nil {
				return err
			}
			result := store.raft.SubmitOperation(clBytes, raft.Replicated, store.applyTimeout).Await()
			if err = result.Error(); err != nil {
				return err
			}
			response, ok := result.Success().ApplicationResponse.(*ApplyResponse)
			if !ok {
				return errors.New("invalid application response")
			}
			if !response.Success {
				return response.Error
			}
			return nil
		}
	}
	clBytes, err := json.Marshal(*cl)
	if err != nil {
		return err
	}
	retries, delay := 30, time.Microsecond*100
RETRY:
	err = store.storage.Set("changelog/"+cl.ID, clBytes)
	if err != nil {
		if retries > 0 {
			store.logger.Error("failed to store changelog",
				zap.Error(err), zap.Int("retries", retries),
			)
			time.Sleep(delay)
			retries--
			goto RETRY
		}
		return err
	}
	return nil
}

func (store *Store) DeleteChangeLogs(logs []*spec.ChangeLog) (int, error) {
	removed := 0
	for _, cl := range logs {
		err := store.storage.Delete("changelog/" + cl.ID)
		if err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

func createDocumentKey(docId, colName, nsName string) string {
	return "doc/" + nsName + "/" + colName + "/" + docId
}

func (store *Store) FetchDocument(docId, colName, nsName string) (*spec.Document, error) {
	docBytes, err := store.storage.Get(createDocumentKey(docId, colName, nsName))
	if err != nil {
		if err == storage.ErrStoreLocked {
			return nil, err
		}
		return nil, errors.New("failed to fetch document: " + err.Error())
	}
	doc := &spec.Document{}
	err = json.Unmarshal(docBytes, doc)
	if err != nil {
		store.logger.Debug("failed to unmarshal document entry: %s, skipping %s",
			zap.Error(err), zap.String("document_id", docId))
		return nil, errors.New("failed to unmarshal document entry" + err.Error())
	}
	return doc, nil
}

func (store *Store) FetchDocuments(
	namespaceName, collectionName string,
	limit, offset int,
) ([]*spec.Document, error) {
	docs := make([]*spec.Document, 0)
	docPrefix := createDocumentKey("", collectionName, namespaceName)
	err := store.storage.IterateValuesPrefix(docPrefix, func(key string, val []byte) error {
		if offset -= 1; offset > 0 {
			return nil
		} else if limit -= 1; limit != 0 {
			var newDoc spec.Document
			err := json.Unmarshal(val, &newDoc)
			if err != nil {
				return err
			}
			docs = append(docs, &newDoc)
		}
		return nil
	})
	if err != nil {
		return nil, errors.New("failed to fetch documents: " + err.Error())
	}
	return docs, nil
}

func (store *Store) StoreDocument(doc *spec.Document) error {
	docBytes, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	store.logger.Debug("storing document")
	err = store.storage.Set(createDocumentKey(doc.ID, doc.CollectionName, doc.NamespaceName), docBytes)
	if err != nil {
		return err
	}
	return nil
}

func (store *Store) DeleteDocument(id, colName, nsName string) error {
	err := store.storage.Delete(createDocumentKey(id, colName, nsName))
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil
		}
		return err
	}
	return nil
}

func (store *Store) DeleteDocuments(doc *spec.Document) error {
	err := store.storage.IterateTxnPrefix(createDocumentKey("", doc.CollectionName, doc.NamespaceName),
		func(txn storage.StorageTxn, key string) error {
			return txn.Delete(key)
		})
	if err != nil {
		return err
	}
	return nil
}

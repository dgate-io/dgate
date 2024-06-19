package proxy

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dgate-io/dgate/internal/config"
	"github.com/dgate-io/dgate/internal/pattern"
	"github.com/dgate-io/dgate/internal/proxy/proxy_transport"
	"github.com/dgate-io/dgate/internal/proxy/proxystore"
	"github.com/dgate-io/dgate/internal/proxy/reverse_proxy"
	"github.com/dgate-io/dgate/internal/router"
	"github.com/dgate-io/dgate/pkg/cache"
	"github.com/dgate-io/dgate/pkg/modules/extractors"
	"github.com/dgate-io/dgate/pkg/resources"
	"github.com/dgate-io/dgate/pkg/scheduler"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/dgate-io/dgate/pkg/storage"
	"github.com/dgate-io/dgate/pkg/util"
	"github.com/dgate-io/dgate/pkg/util/tree/avl"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/hashicorp/raft"
	"go.uber.org/zap"
)

type ProxyState struct {
	debugMode      bool
	changeHash     *atomic.Uint64
	startTime      time.Time
	logger         *zap.Logger
	printer        console.Printer
	config         *config.DGateConfig
	store          *proxystore.ProxyStore
	changeLogs     []*spec.ChangeLog
	metrics        *ProxyMetrics
	sharedCache    cache.TCache
	proxyLock      *sync.RWMutex
	pendingChanges bool

	rm   *resources.ResourceManager
	skdr scheduler.Scheduler

	providers   avl.Tree[string, *RequestContextProvider]
	modPrograms avl.Tree[string, *goja.Program]

	ready               *atomic.Bool
	replicationSettings *ProxyReplication
	replicationEnabled  bool
	routers             avl.Tree[string, *router.DynamicRouter]

	ReverseProxyBuilder   reverse_proxy.Builder
	ProxyTransportBuilder proxy_transport.Builder
	ProxyHandler          ProxyHandlerFunc
}

func NewProxyState(logger *zap.Logger, conf *config.DGateConfig) *ProxyState {
	var dataStore storage.Storage
	switch conf.Storage.StorageType {
	case config.StorageTypeMemory:
		memConfig, err := config.StoreConfig[storage.MemStoreConfig](conf.Storage.Config)
		if err != nil {
			panic(fmt.Errorf("invalid config: %s", err))
		} else {
			memConfig.Logger = logger
		}
		dataStore = storage.NewMemStore(&memConfig)
	case config.StorageTypeFile:
		fileConfig, err := config.StoreConfig[storage.FileStoreConfig](conf.Storage.Config)
		if err != nil {
			panic(fmt.Errorf("invalid config: %s", err))
		} else {
			fileConfig.Logger = logger
		}
		dataStore = storage.NewFileStore(&fileConfig)
	default:
		panic(fmt.Errorf("invalid storage type: %s", conf.Storage.StorageType))
	}
	var opt resources.Options
	if conf.DisableDefaultNamespace {
		logger.Debug("default namespace disabled")
	} else {
		opt = resources.WithDefaultNamespace(spec.DefaultNamespace)
	}
	var printer console.Printer = &extractors.NoopPrinter{}
	consoleLevel, err := zap.ParseAtomicLevel(conf.ProxyConfig.ConsoleLogLevel)
	if err != nil {
		panic(fmt.Errorf("invalid console log level: %s", err))
	}
	printer = NewProxyPrinter(logger, consoleLevel)
	rpLogger := logger.Named("reverse-proxy")
	storeLogger := logger.Named("store")
	schedulerLogger := logger.Named("scheduler")

	replicationEnabled := false
	if conf.AdminConfig != nil && conf.AdminConfig.Replication != nil {
		replicationEnabled = true
	}
	state := &ProxyState{
		startTime:  time.Now(),
		ready:      new(atomic.Bool),
		changeHash: new(atomic.Uint64),
		logger:     logger,
		debugMode:  conf.Debug,
		config:     conf,
		metrics:    NewProxyMetrics(),
		printer:    printer,
		routers:    avl.NewTree[string, *router.DynamicRouter](),
		rm:         resources.NewManager(opt),
		skdr: scheduler.New(scheduler.Options{
			Logger: schedulerLogger,
		}),
		providers:          avl.NewTree[string, *RequestContextProvider](),
		modPrograms:        avl.NewTree[string, *goja.Program](),
		proxyLock:          new(sync.RWMutex),
		sharedCache:        cache.New(),
		store:              proxystore.New(dataStore, storeLogger),
		replicationEnabled: replicationEnabled,
		ReverseProxyBuilder: reverse_proxy.NewBuilder().
			FlushInterval(-1).
			ErrorLogger(zap.NewStdLog(rpLogger)).
			CustomRewrite(func(in *http.Request, out *http.Request) {
				if in.URL.Scheme == "ws" {
					out.URL.Scheme = "http"
				} else if in.URL.Scheme == "wss" {
					out.URL.Scheme = "https"
				} else if in.URL.Scheme == "" {
					if in.TLS != nil {
						out.URL.Scheme = "https"
					} else {
						out.URL.Scheme = "http"
					}
				}
			}),
		ProxyTransportBuilder: proxy_transport.NewBuilder(),
		ProxyHandler:          proxyHandler,
	}

	if conf.Debug {
		if err := state.initConfigResources(conf.ProxyConfig.InitResources); err != nil {
			panic("error initializing resources: " + err.Error())
		}
	}

	return state
}

func (ps *ProxyState) Store() *proxystore.ProxyStore {
	return ps.store
}

func (ps *ProxyState) ChangeHash() uint64 {
	return ps.changeHash.Load()
}

func (ps *ProxyState) ChangeLogs() []*spec.ChangeLog {
	// return a copy of the change logs
	ps.proxyLock.RLock()
	defer ps.proxyLock.RUnlock()
	return append([]*spec.ChangeLog{}, ps.changeLogs...)
}

func (ps *ProxyState) Ready() bool {
	return ps.ready.Load()
}

func (ps *ProxyState) SetReady(ready bool) {
	ps.ready.CompareAndSwap(false, true)
}

func (ps *ProxyState) Raft() *raft.Raft {
	if ps.replicationEnabled {
		return ps.replicationSettings.raft
	}
	return nil
}

func (ps *ProxyState) SetupRaft(r *raft.Raft) {
	ps.proxyLock.Lock()
	defer ps.proxyLock.Unlock()

	oc := make(chan raft.Observation, 32)
	r.RegisterObserver(raft.NewObserver(oc, false, func(o *raft.Observation) bool {
		switch o.Data.(type) {
		case
			raft.FailedHeartbeatObservation,
			raft.LeaderObservation,
			raft.PeerObservation:
			return true
		}
		return false
	}))
	go func() {
		logger := ps.logger.Named("raft-observer")
		for obs := range oc {
			switch ro := obs.Data.(type) {
			case raft.PeerObservation:
				if ro.Removed {
					logger.Info("peer removed",
						zap.Stringer("suffrage", ro.Peer.Suffrage),
						zap.String("address", string(ro.Peer.Address)),
						zap.String("id", string(ro.Peer.ID)),
					)
				} else {
					logger.Info("peer added",
						zap.Stringer("suffrage", ro.Peer.Suffrage),
						zap.String("address", string(ro.Peer.Address)),
						zap.String("id", string(ro.Peer.ID)),
					)
				}
			case raft.LeaderObservation:
				logger.Info("leader observation",
					zap.String("leader_addr", string(ro.LeaderAddr)),
					zap.String("leader_id", string(ro.LeaderID)),
				)
			}
		}
		panic("raft observer channel closed")
	}()
	ps.replicationSettings = NewProxyReplication(r)
}

func (ps *ProxyState) WaitForChanges(log *spec.ChangeLog) error {
	if r := ps.Raft(); r != nil && log != nil {
		waitTime := time.Second * 5
		if r.State() != raft.Leader {
			return r.Barrier(waitTime).Error()
		} else {
			hasChanges := func() bool {
				ps.proxyLock.RLock()
				defer ps.proxyLock.RUnlock()
				if ps.changeLogs != nil {
					lastLog := ps.changeLogs[len(ps.changeLogs)-1]
					if lastLog.ID >= log.ID {
						return true
					}
				}
				return false
			}
			timeout := time.After(waitTime)
			backoff := time.Millisecond * 10
			multiplier := 1.8
			for {
				select {
				case <-timeout:
					if hasChanges() {
						return nil
					}
					return errors.New("timeout waiting for changes")
				case <-time.After(backoff):
					if hasChanges() {
						return nil
					}
					backoff = min(time.Duration(float64(backoff)*multiplier), time.Millisecond*500)
				}
			}
		}
	} else {
		ps.proxyLock.RLock()
		defer ps.proxyLock.RUnlock()
	}
	return nil
}

func (ps *ProxyState) ApplyChangeLog(log *spec.ChangeLog) error {
	if r := ps.Raft(); r != nil {
		if r.State() != raft.Leader {
			return raft.ErrNotLeader
		}
		if err := ps.processChangeLog(log, true, false); err != nil {
			return err
		}
		encodedCL, err := json.Marshal(log)
		if err != nil {
			return err
		}
		raftLog := raft.Log{Data: encodedCL}
		now := time.Now()
		future := r.ApplyLog(raftLog, time.Second*15)
		err = future.Error()
		ps.logger.With().
			Debug("waiting for reply from raft",
				zap.String("id", log.ID),
				zap.Stringer("command", log.Cmd),
				zap.Stringer("command", time.Since(now)),
				zap.Uint64("index", future.Index()),
				zap.Any("response", future.Response()),
				zap.Error(err),
			)
		return err
	} else {
		return ps.processChangeLog(log, true, true)
	}
}

func (ps *ProxyState) ResourceManager() *resources.ResourceManager {
	return ps.rm
}

func (ps *ProxyState) Scheduler() scheduler.Scheduler {
	return ps.skdr
}

func (ps *ProxyState) SharedCache() cache.TCache {
	return ps.sharedCache
}

// restartState - restart state clears the state and reloads the configuration
// this is useful for rollbacks when broken changes are made.
func (ps *ProxyState) restartState(fn func(error)) {
	ps.proxyLock.Lock()
	defer ps.proxyLock.Unlock()

	ps.logger.Info("Attempting to restart state...")

	ps.rm.Empty()
	ps.modPrograms.Clear()
	ps.providers.Clear()
	ps.routers.Clear()
	ps.sharedCache.Clear()
	ps.Scheduler().Stop()
	if err := ps.initConfigResources(ps.config.ProxyConfig.InitResources); err != nil {
		fn(err)
		return
	}
	if ps.replicationEnabled {
		raft := ps.Raft()
		err := raft.ReloadConfig(raft.ReloadableConfig())
		if err != nil {
			fn(err)
			return
		}
	} else {
		if err := ps.restoreFromChangeLogs(true); err != nil {
			fn(err)
			return
		}
	}
	ps.logger.Info("State successfully restarted")
	fn(nil)
}

// ReloadState - reload state checks the change logs to see if a reload is required,
// specifying check as false skips this step and automatically reloads
func (ps *ProxyState) ReloadState(check bool, logs ...*spec.ChangeLog) error {
	reload := !check
	if check {
		for _, log := range logs {
			if log.Cmd.Resource().IsRelatedTo(spec.Routes) {
				reload = true
				continue
			}
		}
	}
	if reload {
		return ps.processChangeLog(nil, true, false)
	}
	return nil
}

func (ps *ProxyState) ProcessChangeLog(log *spec.ChangeLog, reload bool) error {
	if err := ps.processChangeLog(log, reload, true); err != nil {
		ps.logger.Error("processing error", zap.Error(err))
		return err
	}
	return nil
}

func (ps *ProxyState) DynamicTLSConfig(certFile, keyFile string) *tls.Config {
	var fallbackCert *tls.Certificate
	if certFile != "" && keyFile != "" {
		cert, err := loadCertFromFile(certFile, keyFile)
		if err != nil {
			panic(fmt.Errorf("error loading cert: %s", err))
		}
		fallbackCert = cert
	}

	return &tls.Config{
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			if cert, err := ps.getDomainCertificate(info.ServerName); err != nil {
				return nil, err
			} else if cert == nil {
				if fallbackCert != nil {
					return fallbackCert, nil
				} else {
					ps.logger.Error("no cert found matching: " + info.ServerName)
					return nil, errors.New("no cert found")
				}
			} else {
				return cert, nil
			}
		},
	}
}

func loadCertFromFile(certFile, keyFile string) (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

func (ps *ProxyState) getDomainCertificate(domain string) (*tls.Certificate, error) {
	start := time.Now()
	allowedDomains := ps.config.ProxyConfig.AllowedDomains
	domainAllowed := len(allowedDomains) == 0
	if !domainAllowed {
		_, domainMatch, err := pattern.MatchAnyPattern(domain, allowedDomains)
		if err != nil {
			ps.logger.Error("Error checking domain match list",
				zap.Error(err),
			)
			return nil, err
		}
		domainAllowed = domainMatch
	}
	if domainAllowed {
		for _, d := range ps.rm.GetDomainsByPriority() {
			_, match, err := pattern.MatchAnyPattern(domain, d.Patterns)
			if err != nil {
				ps.logger.Error("Error checking domain match list",
					zap.Error(err),
				)
				return nil, err
			} else if match && d.Cert != "" && d.Key != "" {
				var err error
				defer ps.metrics.MeasureCertResolutionDuration(
					start, domain, false, err)

				certBucket := ps.sharedCache.Bucket("certs")
				key := fmt.Sprintf("cert:%s:%s:%d", d.Namespace.Name,
					d.Name, d.CreatedAt.UnixMilli())
				if cert, ok := certBucket.Get(key); ok {
					return cert.(*tls.Certificate), nil
				}
				var serverCert tls.Certificate
				serverCert, err = tls.X509KeyPair(
					[]byte(d.Cert), []byte(d.Key))
				if err != nil {
					ps.logger.Error("Error loading cert",
						zap.Error(err),
						zap.String("domain_name", d.Name),
						zap.String("namespace", d.Namespace.Name),
					)
					return nil, err
				}
				certBucket.Set(key, &serverCert)
				return &serverCert, nil
			}
		}
	}
	return nil, nil
}

func (ps *ProxyState) initConfigResources(resources *config.DGateResources) error {
	if resources != nil {
		numChanges, err := resources.Validate()
		if err != nil {
			return err
		}
		if numChanges > 0 {
			defer func() {
				if err != nil {
					err = ps.processChangeLog(nil, false, false)
				}
			}()
		}
		ps.logger.Info("Initializing resources")
		for _, ns := range resources.Namespaces {
			cl := spec.NewChangeLog(&ns, ns.Name, spec.AddNamespaceCommand)
			err := ps.processChangeLog(cl, false, false)
			if err != nil {
				return err
			}
		}
		for _, mod := range resources.Modules {
			if mod.PayloadFile != "" {
				payload, err := os.ReadFile(mod.PayloadFile)
				if err != nil {
					return err
				}
				mod.Payload = base64.StdEncoding.EncodeToString(payload)
			}
			if mod.Payload != "" {
				mod.Payload = base64.StdEncoding.EncodeToString(
					[]byte(mod.Payload),
				)
			}
			cl := spec.NewChangeLog(&mod.Module, mod.NamespaceName, spec.AddModuleCommand)
			err := ps.processChangeLog(cl, false, false)
			if err != nil {
				return err
			}
		}
		for _, svc := range resources.Services {
			cl := spec.NewChangeLog(&svc, svc.NamespaceName, spec.AddServiceCommand)
			err := ps.processChangeLog(cl, false, false)
			if err != nil {
				return err
			}
		}
		for _, rt := range resources.Routes {
			cl := spec.NewChangeLog(&rt, rt.NamespaceName, spec.AddRouteCommand)
			err := ps.processChangeLog(cl, false, false)
			if err != nil {
				return err
			}
		}
		for _, dom := range resources.Domains {
			if dom.CertFile != "" {
				cert, err := os.ReadFile(dom.CertFile)
				if err != nil {
					return err
				}
				dom.Cert = string(cert)
			}
			if dom.KeyFile != "" {
				key, err := os.ReadFile(dom.KeyFile)
				if err != nil {
					return err
				}
				dom.Key = string(key)
			}
			cl := spec.NewChangeLog(&dom.Domain, dom.NamespaceName, spec.AddDomainCommand)
			err := ps.processChangeLog(cl, false, false)
			if err != nil {
				return err
			}
		}
		for _, col := range resources.Collections {
			cl := spec.NewChangeLog(&col, col.NamespaceName, spec.AddCollectionCommand)
			err := ps.processChangeLog(cl, false, false)
			if err != nil {
				return err
			}
		}
		for _, doc := range resources.Documents {
			cl := spec.NewChangeLog(&doc, doc.NamespaceName, spec.AddDocumentCommand)
			err := ps.processChangeLog(cl, false, false)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (ps *ProxyState) FindNamespaceByRequest(r *http.Request) *spec.DGateNamespace {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}

	// if there are no domains and only one namespace, return that namespace
	if ps.rm.DomainCountEquals(0) && ps.rm.NamespaceCountEquals(1) {
		return ps.rm.GetFirstNamespace()
	}

	// search through domains for a match
	var defaultNsHasDomain bool
	if domains := ps.rm.GetDomainsByPriority(); len(domains) > 0 {
		for _, d := range domains {
			if !ps.config.DisableDefaultNamespace {
				if d.Namespace.Name == "default" {
					defaultNsHasDomain = true
				}
			}
			_, match, err := pattern.MatchAnyPattern(host, d.Patterns)
			if err != nil {
				ps.logger.Error("error matching namespace", zap.Error(err))
			} else if match {
				return d.Namespace
			}
		}
	}
	// if no domain matches, return the default namespace, if it doesn't have a domain
	if !ps.config.DisableDefaultNamespace && !defaultNsHasDomain {
		if defaultNs, ok := ps.rm.GetNamespace("default"); ok {
			return defaultNs
		}
	}
	return nil
}

func (ps *ProxyState) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	if ns := ps.FindNamespaceByRequest(r); ns != nil {
		allowedDomains := ps.config.ProxyConfig.AllowedDomains
		// if allowed domains is empty, allow all domains
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
			// ignore host/port error for metrics
			err = nil
		}
		defer ps.metrics.MeasureNamespaceResolutionDuration(
			start, host, ns.Name, err,
		)
		var ok bool
		if len(allowedDomains) > 0 {
			_, ok, err = pattern.MatchAnyPattern(host, allowedDomains)
			if err != nil {
				ps.logger.Debug("Error checking domain match list",
					zap.Error(err),
				)
				util.WriteStatusCodeError(w, http.StatusInternalServerError)
				return
			} else if !ok {
				ps.logger.Debug("Domain not allowed", zap.String("domain", host))
				// if debug mode is enabled, return a 403
				util.WriteStatusCodeError(w, http.StatusForbidden)
				if ps.debugMode {
					w.Write([]byte("domain not allowed"))
				}
				return
			}
		}
		redirectDomains := ps.config.ProxyConfig.RedirectHttpsDomains
		if r.TLS == nil && len(redirectDomains) > 0 {
			if _, ok, err = pattern.MatchAnyPattern(host, redirectDomains); err != nil {
				ps.logger.Error("Error checking domain match list",
					zap.Error(err),
				)
				util.WriteStatusCodeError(w, http.StatusInternalServerError)
				return
			} else if ok {
				url := *r.URL
				url.Scheme = "https"
				ps.logger.Info("Redirecting to https",
					zap.Stringer("url", &url),
				)
				http.Redirect(w, r, url.String(),
					// maybe change to http.StatusMovedPermanently
					http.StatusTemporaryRedirect)
				return
			}
		}
		if router, ok := ps.routers.Find(ns.Name); ok {
			router.ServeHTTP(w, r)
		} else {
			util.WriteStatusCodeError(w, http.StatusNotFound)
		}
	} else {
		if ps.config.ProxyConfig.StrictMode {
			closeConnection(w)
			return
		}
		trustedIp := util.GetTrustedIP(r, ps.config.ProxyConfig.XForwardedForDepth)
		ps.logger.Debug("No namespace found for request",
			zap.String("protocol", r.Proto),
			zap.String("host", r.Host),
			zap.String("path", r.URL.Path),
			zap.Bool("secure", r.TLS != nil),
			zap.String("remote_addr", trustedIp),
		)
		util.WriteStatusCodeError(w, http.StatusNotFound)
	}
}

func closeConnection(w http.ResponseWriter) {
	if loot, ok := w.(http.Hijacker); ok {
		if conn, _, err := loot.Hijack(); err == nil {
			defer conn.Close()
			return
		}
	}
}

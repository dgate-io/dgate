package proxy

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dgate-io/dgate/internal/config"
	"github.com/dgate-io/dgate/internal/pattern"
	"github.com/dgate-io/dgate/internal/proxy/proxy_store"
	"github.com/dgate-io/dgate/internal/proxy/proxy_transport"
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
	"github.com/rs/zerolog"
)

type ProxyState struct {
	version    string
	debugMode  bool
	startTime  time.Time
	config     *config.DGateConfig
	logger     zerolog.Logger
	printer    console.Printer
	store      *proxy_store.ProxyStore
	proxyLock  *sync.RWMutex
	changeHash uint32

	metrics     *ProxyMetrics
	sharedCache cache.TCache

	changeChan chan *spec.ChangeLog
	rm         *resources.ResourceManager
	skdr       scheduler.Scheduler

	providers   avl.Tree[string, *RequestContextProvider]
	modPrograms avl.Tree[string, *goja.Program]

	raftReady           atomic.Bool
	replicationSettings *ProxyReplication
	replicationEnabled  bool

	routers avl.Tree[string, *router.DynamicRouter]

	ReverseProxyBuilder   reverse_proxy.Builder
	ProxyTransportBuilder proxy_transport.Builder
	ProxyHandlerFunc      ProxyHandlerFunc
}

func NewProxyState(conf *config.DGateConfig) *ProxyState {
	var logger zerolog.Logger
	level, err := zerolog.ParseLevel(conf.LogLevel)
	if err != nil {
		panic(fmt.Errorf("invalid log level: %s", err))
	}
	if level == zerolog.NoLevel {
		level = zerolog.InfoLevel
	}

	if util.EnvVarCheckBool("LOG_JSON") {
		logger = zerolog.New(os.Stdout).
			Level(level).With().
			Timestamp().Logger()
	} else {
		cw := zerolog.NewConsoleWriter()
		cw.TimeFormat = "2006/01/02T03:04:05PM"
		cw.NoColor = util.EnvVarCheckBool("LOG_NO_COLOR")
		logger = zerolog.New(cw).
			Level(level).With().
			Timestamp().Logger()
	}
	logger = logger.Level(level)

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
	var opt resources.Options
	if conf.DisableDefaultNamespace {
		logger.Debug().Msg("default namespace disabled")
	} else {
		opt = resources.WithDefaultNamespace(spec.DefaultNamespace)
	}
	var printer console.Printer = &extractors.NoopPrinter{}
	if conf.ProxyConfig.EnableConsoleLogger {
		printer = NewProxyPrinter(logger)
	}
	rpLogger := Logger(&logger,
		WithComponentLogger("reverse-proxy"),
		WithDefaultLevel(zerolog.InfoLevel),
	)
	storeLogger := Logger(&logger,
		WithComponentLogger("proxy_store"),
		WithDefaultLevel(zerolog.InfoLevel),
	)
	replicationEnabled := false
	if conf.AdminConfig != nil && conf.AdminConfig.Replication != nil {
		replicationEnabled = true
	}
	state := &ProxyState{
		version:            "unknown",
		startTime:          time.Now(),
		raftReady:          atomic.Bool{},
		logger:             logger,
		debugMode:          conf.Debug,
		config:             conf,
		metrics:            NewProxyMetrics(),
		printer:            printer,
		routers:            avl.NewTree[string, *router.DynamicRouter](),
		changeChan:         make(chan *spec.ChangeLog, 1),
		rm:                 resources.NewManager(opt),
		providers:          avl.NewTree[string, *RequestContextProvider](),
		modPrograms:        avl.NewTree[string, *goja.Program](),
		proxyLock:          new(sync.RWMutex),
		sharedCache:        cache.New(),
		store:              proxy_store.New(dataStore, storeLogger),
		replicationEnabled: replicationEnabled,
		ReverseProxyBuilder: reverse_proxy.NewBuilder().
			FlushInterval(-1).
			ErrorLogger(log.New(rpLogger, "", 0)).
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
		ProxyHandlerFunc:      proxyHandler,
	}

	if conf.Debug {
		if err = state.initConfigResources(conf.ProxyConfig.InitResources); err != nil {
			panic("error initializing resources: " + err.Error())
		}
	}

	return state
}

func (ps *ProxyState) Version() string {
	return ps.version
}

func (ps *ProxyState) Store() *proxy_store.ProxyStore {
	return ps.store
}

func (ps *ProxyState) Logger(opts ...LoggerOptions) *zerolog.Logger {
	return Logger(&ps.logger, opts...)
}

type LoggerOptions func(zerolog.Context) zerolog.Context

func Logger(logger *zerolog.Logger, opts ...LoggerOptions) *zerolog.Logger {
	logCtx := logger.With()

	for _, opt := range opts {
		logCtx = opt(logCtx)
	}
	lgr := logCtx.Logger()
	return &lgr
}

func WithComponentLogger(component string) LoggerOptions {
	return func(ctx zerolog.Context) zerolog.Context {
		return ctx.Str("component", component)
	}
}

func WithDefaultLevel(level zerolog.Level) LoggerOptions {
	return func(ctx zerolog.Context) zerolog.Context {
		return ctx.Str(zerolog.LevelFieldName, zerolog.LevelFieldMarshalFunc(level))
	}
}

func (ps *ProxyState) ChangeHash() uint32 {
	return ps.changeHash
}

func (ps *ProxyState) SetReady() {
	if ps.replicationEnabled && !ps.raftReady.Load() {
		ps.logger.Info().
			Msgf("Replication status is now ready after %s", time.Since(ps.startTime))
		ps.raftReady.Store(true)
		return
	}
}

func (ps *ProxyState) Ready() bool {
	if ps.replicationEnabled {
		return ps.raftReady.Load()
	}
	return true
}

func (ps *ProxyState) Raft() *raft.Raft {
	if ps.replicationEnabled {
		return ps.replicationSettings.raft
	}
	return nil
}

func (ps *ProxyState) SetupRaft(r *raft.Raft, rc *raft.Config) {
	ps.proxyLock.Lock()
	defer ps.proxyLock.Unlock()
	ps.replicationSettings = NewProxyReplication(r, rc)
}

func (ps *ProxyState) WaitForChanges() {
	ps.proxyLock.RLock()
	defer ps.proxyLock.RUnlock()
}

func (ps *ProxyState) ApplyChangeLog(log *spec.ChangeLog) error {
	if ps.replicationEnabled {
		if log.Cmd.IsNoop() {
			ps.processChangeLog(
				log, true, false,
			)
		}
		r := ps.replicationSettings.raft
		if r.State() != raft.Leader {
			return raft.ErrNotLeader
		}
		encodedCL, err := json.Marshal(log)
		if err != nil {
			return err
		}
		raftLog := raft.Log{
			Data: encodedCL,
		}
		future := r.ApplyLog(raftLog, time.Second*15)
		ps.logger.Trace().
			Interface("changelog", log).
			Msgf("waiting for reply: %s", log.ID)
		return future.Error()
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
		<-ps.applyChange(nil)
	}
	return nil
}

func (ps *ProxyState) ProcessChangeLog(log *spec.ChangeLog, reload bool) error {
	err := ps.processChangeLog(log, reload, !ps.replicationEnabled)
	if err != nil {
		ps.logger.Error().Err(err).Msg("error processing change log")
	}
	return err
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
					ps.logger.Error().Msg("no cert found matching: " + info.ServerName)
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
			ps.logger.Error().Msgf("Error checking domain match list: %s", err.Error())
			return nil, err
		}
		domainAllowed = domainMatch
	}
	if domainAllowed {
		for _, d := range ps.rm.GetDomainsByPriority() {
			_, match, err := pattern.MatchAnyPattern(domain, d.Patterns)
			if err != nil {
				ps.logger.Error().Msgf("Error checking domain match list: %s", err.Error())
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
					ps.logger.Error().Msgf("Error loading cert: %s on domain %s/%s",
						err.Error(), d.Namespace.Name, d.Name)
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
			defer ps.processChangeLog(nil, false, false)
		}
		ps.logger.Info().Msg("Initializing resources")
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
				ps.logger.Error().Err(err).
					Msg("error matching namespace")
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
				ps.logger.Trace().Msgf("Error checking domain match list: %s", err.Error())
				util.WriteStatusCodeError(w, http.StatusInternalServerError)
				return
			} else if !ok {
				ps.logger.Trace().Msgf("Domain %s not allowed", host)
				// if debug mode is enabled, return a 403
				util.WriteStatusCodeError(w, http.StatusForbidden)
				if ps.debugMode {
					w.Write([]byte(" - Domain not allowed"))
				}
				return
			}
		}
		redirectDomains := ps.config.ProxyConfig.RedirectHttpsDomains
		if r.TLS == nil && len(redirectDomains) > 0 {
			if _, ok, err = pattern.MatchAnyPattern(host, redirectDomains); err != nil {
				ps.logger.Error().Msgf("Error checking domain match list: %s", err.Error())
				util.WriteStatusCodeError(w, http.StatusInternalServerError)
				return
			} else if ok {
				url := *r.URL
				url.Scheme = "https"
				ps.logger.Info().Msgf("Redirecting to https: %s", url.String())
				http.Redirect(w, r, url.String(),
					// maybe change to http.StatusMovedPermanently
					http.StatusTemporaryRedirect)
				return
			}
		}
		if router, ok := ps.routers.Find(ns.Name); ok {
			router.ServeHTTP(w, r)
		} else {
			ps.logger.Debug().Msgf("No router found for namespace: %s", ns.Name)
			util.WriteStatusCodeError(w, http.StatusNotFound)
		}
	} else {
		ps.logger.Debug().Msgf(
			"No namespace found for request - %s %s %s secure:%t from %s",
			r.Proto, r.Host, r.URL.String(), r.TLS != nil, r.RemoteAddr,
		)
		util.WriteStatusCodeError(w, http.StatusNotFound)
	}
}

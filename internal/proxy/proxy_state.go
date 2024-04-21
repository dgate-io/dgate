package proxy

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"sync"
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
	"github.com/rs/zerolog"
)

type ProxyState struct {
	debugMode  bool
	config     *config.DGateConfig
	status     ProxyStatus
	logger     zerolog.Logger
	printer    console.Printer
	stats      *ProxyStats
	store      *proxystore.ProxyStore
	proxyLock  *sync.RWMutex
	changeHash uint32

	sharedCache cache.TCache

	changeChan chan *spec.ChangeLog
	rm         *resources.ResourceManager
	skdr       scheduler.Scheduler

	providers   avl.Tree[string, *RequestContextProvider]
	modPrograms avl.Tree[string, *goja.Program]

	replicationSettings *ProxyReplication
	replicationEnabled  bool

	routers avl.Tree[string, *router.DynamicRouter]

	ReverseProxyBuilder   reverse_proxy.Builder
	ProxyTransportBuilder proxy_transport.Builder
	ProxyHandlerFunc      ProxyHandlerFunc
}

type (
	ProxyStatus byte
)

const (
	ProxyStatusStarting ProxyStatus = iota
	ProxyStatusModifying
	ProxyStatusRunning
	ProxyStatusStopping
	ProxyStatusClosed
)

type ProxySnapshot struct {
	ResourceManager *resources.ResourceManager `json:"resource_manager"`
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
	case config.StorageTypeMemory:
		memConfig := storage.MemoryStoreConfig{}
		memConfig.Logger = logger
		dataStore = storage.NewMemoryStore(&memConfig)
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
	// kl := keylock.NewKeyLock()
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
	state := &ProxyState{
		logger:      logger,
		debugMode:   conf.Debug,
		config:      conf,
		status:      ProxyStatusModifying,
		stats:       NewProxyStats(20),
		printer:     printer,
		routers:     avl.NewTree[string, *router.DynamicRouter](),
		changeChan:  make(chan *spec.ChangeLog, 1),
		rm:          resources.NewManager(opt),
		providers:   avl.NewTree[string, *RequestContextProvider](),
		modPrograms: avl.NewTree[string, *goja.Program](),
		proxyLock:   new(sync.RWMutex),
		sharedCache: cache.New(),

		ReverseProxyBuilder: reverse_proxy.NewBuilder().
			FlushInterval(time.Millisecond * 10).
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

	state.store = proxystore.New(dataStore, state.Logger(WithComponentLogger("proxystore")))

	err = state.initConfigResources(conf.ProxyConfig.InitResources)
	if err != nil {
		panic("error initializing resources: " + err.Error())
	}
	return state
}

func (ps *ProxyState) RestoreState(r io.Reader) error {
	// register empty change to refresh state
	defer ps.applyChange(nil)
	dec := gob.NewDecoder(r)
	var snapshot *ProxySnapshot
	err := dec.Decode(snapshot)
	if err != nil {
		return err
	}
	// TODO: ps.rm.RestoreState(snapshot)
	return nil
}

// func (ps *ProxyState) CaptureState() *ProxySnapshot {
// 	unlock := ps.keyLock.LockAll()
// 	defer unlock()
// 	return &ProxySnapshot{
// 		Namespaces:          ps.namespaces,
// 		NamespaceModuleMap:  ps.namespaceModuleMap,
// 		NamespaceRouteMap:   ps.namespaceRouteMap,
// 		NamespaceServiceMap: ps.namespaceServiceMap,
// 	}
// }

func (ps *ProxySnapshot) PersistState(w io.Writer) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	return enc.Encode(ps)
}

func (ps *ProxyState) Stats() *ProxyStats {
	return ps.stats
}

type LoggerOptions func(zerolog.Context) zerolog.Context

func (ps *ProxyState) Logger(opts ...LoggerOptions) *zerolog.Logger {
	logCtx := ps.logger.With()

	for _, opt := range opts {
		logCtx = opt(logCtx)
	}
	logger := logCtx.Logger()
	return &logger
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

func (ps *ProxyState) Raft() *raft.Raft {
	if ps.replicationEnabled {
		return ps.replicationSettings.raft
	}
	return nil
}

func (ps *ProxyState) EnableRaft(r *raft.Raft, rc *raft.Config) {
	ps.proxyLock.Lock()
	defer ps.proxyLock.Unlock()
	ps.replicationEnabled = true
	ps.replicationSettings = NewProxyReplication(r, rc)

}

func (ps *ProxyState) WaitForChanges() {
	ps.proxyLock.RLock()
	defer ps.proxyLock.RUnlock()
}

func (ps *ProxyState) ApplyChangeLog(log *spec.ChangeLog) error {
	ps.logger.Trace().
		Bool("replication_enabled", ps.replicationEnabled).
		Msgf("Applying change log: %s", log.Cmd)
	if ps.replicationEnabled {
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
	if ps == nil {
		return nil
	}
	return ps.rm
}

func (ps *ProxyState) Scheduler() scheduler.Scheduler {
	return ps.skdr
}

func (ps *ProxyState) SharedCache() cache.TCache {
	return ps.sharedCache
}

func (ps *ProxyState) ReloadState() error {
	return <-ps.applyChange(nil)
}

func (ps *ProxyState) ProcessChangeLog(log *spec.ChangeLog, register bool) error {
	err := ps.processChangeLog(log, register, !ps.replicationEnabled)
	if err != nil {
		ps.logger.Error().Err(err).Msg("error processing change log")
		return err
	}
	return nil
}

func (ps *ProxyState) DynamicTLSConfig(
	certFile, keyFile string,
) *tls.Config {
	var fallbackCert *tls.Certificate
	return &tls.Config{
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			cert, ok, err := ps.getDomainCertificate(info.ServerName)
			if err != nil {
				return nil, err
			} else if !ok {
				if certFile != "" && keyFile != "" {
					cert, err := ps.loadCertFromFile(certFile, keyFile)
					if err != nil {
						return nil, err
					}
					fallbackCert = cert
				} else if fallbackCert != nil {
					return fallbackCert, nil
				} else {
					return nil, errors.New("no cert found")
				}
			}
			return cert, nil
		},
	}
}

func (ps *ProxyState) loadCertFromFile(certFile, keyFile string) (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

func (ps *ProxyState) getDomainCertificate(domain string) (*tls.Certificate, bool, error) {
	allowedDomains := ps.config.ProxyConfig.AllowedDomains
	_, domainAllowed, err := pattern.MatchAnyPattern(domain, allowedDomains)
	if err != nil {
		ps.logger.Error().Msgf("Error checking domain match list: %s", err.Error())
		return nil, false, err
	}
	if domainAllowed || len(allowedDomains) == 0 {
		domains := ps.rm.GetDomains()
		sort.Slice(domains, func(i, j int) bool {
			if domains[i].Priority == domains[j].Priority {
				return domains[i].Name > domains[j].Name
			}
			return domains[i].Priority > domains[j].Priority
		})
		for _, d := range domains {
			if _, domainMatches, err := pattern.MatchAnyPattern(domain, d.Patterns); err != nil {
				ps.logger.Error().Msgf("Error checking domain match list: %s", err.Error())
				return nil, false, err
			} else {
				if domainMatches {
					if d.TLSCert == nil {
						return nil, false, nil
					}
					return d.TLSCert, true, nil
				}
			}
		}
	}
	return nil, false, nil
}

func (ps *ProxyState) initConfigResources(resources *config.DGateResources) error {
	if resources != nil {
		err := resources.Validate()
		if err != nil {
			return err
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

	domains := ps.rm.GetDomains()
	if len(domains) > 0 {
		sort.Slice(domains, func(i, j int) bool {
			if domains[i].Priority == domains[j].Priority {
				return domains[i].Name > domains[j].Name
			}
			return domains[i].Priority > domains[j].Priority
		})
		var priority int
		var namespace *spec.DGateNamespace
		for _, d := range domains {
			if d.Priority < priority {
				continue
			}
			_, match, err := pattern.MatchAnyPattern(host, d.Patterns)
			if err != nil {
				ps.logger.Error().Err(err).
					Msg("error matching namespace")
				continue
			}
			if match {
				priority, namespace = d.Priority, d.Namespace
			}
		}
		return namespace
	}
	return nil
}

func (ps *ProxyState) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if ns := ps.FindNamespaceByRequest(r); ns != nil {
		allowedDomains := ps.config.ProxyConfig.AllowedDomains
		// if allowed domains is empty, allow all domains
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil {
			host = r.Host
		}
		if len(allowedDomains) > 0 {
			_, ok, err := pattern.MatchAnyPattern(host, allowedDomains)
			if err != nil {
				ps.logger.Trace().Msgf("Error checking domain match list: %s", err.Error())
				util.WriteStatusCodeError(w, http.StatusInternalServerError)
				return
			}
			if !ok {
				ps.logger.Trace().Msgf("Domain %s not allowed", host)
				// if debug mode is enabled, return a 403
				util.WriteStatusCodeError(w, http.StatusForbidden)
				if ps.debugMode {
					w.Write([]byte(" - Domain not allowed"))
				}
				return
			}
		}
		if r.TLS == nil && len(ps.config.ProxyConfig.RedirectHttpsDomains) > 0 {
			_, match, err := pattern.MatchAnyPattern(host, ps.config.ProxyConfig.RedirectHttpsDomains)
			if err != nil {
				ps.logger.Error().Msgf("Error checking domain match list: %s", err.Error())
				util.WriteStatusCodeError(w, http.StatusInternalServerError)
				return
			}
			if match {
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

func (ps *ProxyState) Snapshot() *ProxySnapshot {
	return &ProxySnapshot{
		ResourceManager: ps.rm,
	}
}

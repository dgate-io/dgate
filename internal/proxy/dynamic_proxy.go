package proxy

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/dgate-io/dgate/internal/router"
	"github.com/dgate-io/dgate/pkg/modules/extractors"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/dgate-io/dgate/pkg/typescript"
	"github.com/dgate-io/dgate/pkg/util/tree/avl"
	"github.com/dop251/goja"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func (ps *ProxyState) reconfigureState(init bool, log *spec.ChangeLog) (err error) {
	defer func() {
		if err != nil {
			ps.restartState(func(err error) {
				if err != nil {
					ps.logger.Error("Error restarting state", zap.Error(err))
					go ps.Stop()
				}
			})
		}
		log.PushError(err)
	}()

	ps.proxyLock.Lock()
	defer ps.proxyLock.Unlock()

	start := time.Now()
	if err = ps.setupModules(); err != nil {
		return
	}
	if err = ps.setupRoutes(); err != nil {
		return
	}
	elapsed := time.Since(start)
	if !init {
		ps.logger.Debug("State reloaded",
			zap.Duration("elapsed", elapsed),
		)
	} else {
		ps.logger.Info("State initialized",
			zap.Duration("elapsed", elapsed),
		)
	}
	return nil
}

func (ps *ProxyState) setupModules() error {
	ps.logger.Debug("Setting up modules")
	for _, route := range ps.rm.GetRoutes() {
		if len(route.Modules) > 0 {
			mod := route.Modules[0]
			var (
				err        error
				program    *goja.Program
				modPayload string = mod.Payload
			)
			start := time.Now()
			if mod.Type == spec.ModuleTypeTypescript {
				if modPayload, err = typescript.Transpile(modPayload); err != nil {
					ps.logger.Error("Error transpiling module: " + mod.Name)
					return err
				}
			}
			if mod.Type == spec.ModuleTypeJavascript || mod.Type == spec.ModuleTypeTypescript {
				if program, err = goja.Compile(mod.Name, modPayload, true); err != nil {
					ps.logger.Error("Error compiling module: " + mod.Name)
					return err
				}
			} else {
				return errors.New("invalid module type: " + mod.Type.String())
			}

			testRtCtx := NewRuntimeContext(ps, route, mod)
			defer testRtCtx.Clean()
			err = extractors.SetupModuleEventLoop(ps.printer, testRtCtx)
			if err != nil {
				ps.logger.Error("Error applying module changes",
					zap.Error(err), zap.String("module", mod.Name),
				)
				return err
			}
			ps.modPrograms.Insert(mod.Name+"/"+mod.Namespace.Name, program)
			elapsed := time.Since(start)
			ps.logger.Debug("Module changed applied",
				zap.Duration("elapsed", elapsed),
				zap.String("name", mod.Name),
				zap.String("namespace", mod.Namespace.Name),
			)
		}
	}
	return nil
}

func (ps *ProxyState) setupRoutes() (err error) {
	ps.logger.Debug("Setting up routes")
	reqCtxProviders := avl.NewTree[string, *RequestContextProvider]()
	for namespaceName, routes := range ps.rm.GetRouteNamespaceMap() {
		mux := router.NewMux()
		for _, r := range routes {
			reqCtxProvider := NewRequestContextProvider(r, ps)
			reqCtxProviders.Insert(r.Namespace.Name+"/"+r.Name, reqCtxProvider)
			if len(r.Modules) > 0 {
				modPool, err := NewModulePool(
					256, 1024, reqCtxProvider,
					ps.createModuleExtractorFunc(r),
				)
				if err != nil {
					ps.logger.Error("Error creating module buffer", zap.Error(err))
					return err
				}
				reqCtxProvider.SetModulePool(modPool)
			}
			err = func() (err error) {
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("%v", r)
					}
				}()
				for _, path := range r.Paths {
					if len(r.Methods) > 0 && r.Methods[0] == "*" {
						if len(r.Methods) > 1 {
							return errors.New("route methods cannot have other methods with *")
						}
						mux.Handle(path, ps.HandleRoute(reqCtxProvider, path))
					} else {
						if len(r.Methods) == 0 {
							return errors.New("route must have at least one method")
						} else if err = ValidateMethods(r.Methods); err != nil {
							return err
						}
						for _, method := range r.Methods {
							mux.Method(method, path, ps.HandleRoute(reqCtxProvider, path))
						}
					}
				}
				return nil
			}()
		}

		ps.logger.Debug("Routes have changed, reloading")
		if dr, ok := ps.routers.Find(namespaceName); ok {
			dr.ReplaceMux(mux)
		} else {
			dr := router.NewRouterWithMux(mux)
			ps.routers.Insert(namespaceName, dr)
		}
	}
	return
}

func (ps *ProxyState) createModuleExtractorFunc(r *spec.DGateRoute) ModuleExtractorFunc {
	return func(reqCtx *RequestContextProvider) (_ ModuleExtractor, err error) {
		if len(r.Modules) == 0 {
			return nil, fmt.Errorf("no modules found for route: %s/%s", r.Name, r.Namespace.Name)
		}
		// TODO: Perhaps have some entrypoint flag to determine which module to use
		m := r.Modules[0]
		if program, ok := ps.modPrograms.Find(m.Name + "/" + r.Namespace.Name); !ok {
			ps.logger.Error("Error getting module program: invalid state", zap.Error(err))
			return nil, fmt.Errorf("cannot find module program: %s/%s", m.Name, r.Namespace.Name)
		} else {
			rtCtx := NewRuntimeContext(ps, r, r.Modules...)
			if err := extractors.SetupModuleEventLoop(ps.printer, rtCtx, program); err != nil {
				ps.logger.Error("Error creating runtime for route",
					zap.String("route", reqCtx.route.Name),
					zap.String("namespace", reqCtx.route.Namespace.Name),
				)
				return nil, err
			} else {
				loop := rtCtx.EventLoop()
				errorHandler, err := extractors.ExtractErrorHandlerFunction(loop)
				if err != nil {
					ps.logger.Error("Error extracting error handler function", zap.Error(err))
					return nil, err
				}
				fetchUpstream, err := extractors.ExtractFetchUpstreamFunction(loop)
				if err != nil {
					ps.logger.Error("Error extracting fetch upstream function", zap.Error(err))
					return nil, err
				}
				reqModifier, err := extractors.ExtractRequestModifierFunction(loop)
				if err != nil {
					ps.logger.Error("Error extracting request modifier function", zap.Error(err))
					return nil, err
				}
				resModifier, err := extractors.ExtractResponseModifierFunction(loop)
				if err != nil {
					ps.logger.Error("Error extracting response modifier function", zap.Error(err))
					return nil, err
				}
				reqHandler, err := extractors.ExtractRequestHandlerFunction(loop)
				if err != nil {
					ps.logger.Error("Error extracting request handler function", zap.Error(err))
					return nil, err
				}
				return NewModuleExtractor(
					rtCtx, fetchUpstream,
					reqModifier, resModifier,
					errorHandler, reqHandler,
				), nil
			}
		}
	}
}

func (ps *ProxyState) startProxyServer() {
	cfg := ps.config.ProxyConfig
	hostPort := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	ps.logger.Info("Starting proxy server on " + hostPort)
	proxyHttpLogger := ps.Logger()
	server := &http.Server{
		Addr:     hostPort,
		Handler:  ps,
		ErrorLog: zap.NewStdLog(proxyHttpLogger),
	}
	if cfg.EnableHTTP2 {
		h2Server := &http2.Server{}
		err := http2.ConfigureServer(server, h2Server)
		if err != nil {
			panic(err)
		}
		if cfg.EnableH2C {
			server.Handler = h2c.NewHandler(ps, h2Server)
		}
	}
	if err := server.ListenAndServe(); err != nil {
		ps.logger.Error("Error starting proxy server", zap.Error(err))
		os.Exit(1)
	}
}

func (ps *ProxyState) startProxyServerTLS() {
	cfg := ps.config.ProxyConfig
	if cfg.TLS == nil {
		return
	}
	hostPort := fmt.Sprintf("%s:%d", cfg.Host, cfg.TLS.Port)
	ps.logger.Info("Starting secure proxy server on " + hostPort)
	proxyHttpsLogger := ps.logger.Named("https")
	secureServer := &http.Server{
		Addr:     hostPort,
		Handler:  ps,
		ErrorLog: zap.NewStdLog(proxyHttpsLogger),
		TLSConfig: ps.DynamicTLSConfig(
			cfg.TLS.CertFile,
			cfg.TLS.KeyFile,
		),
	}
	if cfg.EnableHTTP2 {
		h2Server := &http2.Server{}
		err := http2.ConfigureServer(secureServer, h2Server)
		if err != nil {
			panic(err)
		}
		if cfg.EnableH2C {
			secureServer.Handler = h2c.NewHandler(ps, h2Server)
		}
	}
	if err := secureServer.ListenAndServeTLS("", ""); err != nil {
		ps.logger.Error("Error starting secure proxy server", zap.Error(err))
		os.Exit(1)
	}
}

func (ps *ProxyState) Start() (err error) {
	defer func() {
		if err != nil {
			ps.Stop()
		}
	}()

	ps.metrics.Setup(ps.config)
	if err = ps.store.InitStore(); err != nil {
		return err
	}

	go ps.startProxyServer()
	go ps.startProxyServerTLS()

	if !ps.replicationEnabled {
		if err = ps.restoreFromChangeLogs(false); err != nil {
			return err
		}
	}

	return nil
}

func (ps *ProxyState) Stop() {
	go func() {
		<-time.After(10 * time.Second)
		defer os.Exit(1)
		ps.logger.Error("Failed to stop proxy server")
	}()

	ps.logger.Info("Stopping proxy server")
	ps.proxyLock.Lock()
	defer ps.proxyLock.Unlock()
	defer os.Exit(0)
	defer ps.Logger().Sync()

	if raftNode := ps.Raft(); raftNode != nil {
		raftNode.Shutdown().Error()
	}
}

func (ps *ProxyState) HandleRoute(requestCtxProvider *RequestContextProvider, pattern string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ctx, cancel := context.WithCancel(requestCtxPrdovider.ctx)
		// defer cancel()
		ps.ProxyHandler(ps, requestCtxProvider.
			CreateRequestContext(requestCtxProvider.ctx, w, r, pattern))
	}
}

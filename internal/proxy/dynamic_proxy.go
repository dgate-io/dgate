package proxy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dgate-io/dgate/internal/config"
	"github.com/dgate-io/dgate/internal/router"
	"github.com/dgate-io/dgate/pkg/modules/extractors"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/dgate-io/dgate/pkg/typescript"
	"github.com/dgate-io/dgate/pkg/util/tree/avl"
	"github.com/dop251/goja"
	"github.com/rs/zerolog"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
)

func (state *ProxyState) reconfigureState(init bool, _ *spec.ChangeLog) error {
	start := time.Now()
	if err := state.setupModules(); err != nil {
		return err
	}
	if err := state.setupRoutes(); err != nil {
		return err
	}
	if !init {
		state.logger.Debug().Msgf(
			"State reloaded in %s",
			time.Since(start),
		)
	} else {
		state.logger.Info().Msgf(
			"State initialized in %s",
			time.Since(start),
		)
	}
	return nil
}

func (ps *ProxyState) setupModules() error {
	ps.logger.Debug().Msg("Setting up modules")
	eg, _ := errgroup.WithContext(context.TODO())
	for _, route := range ps.rm.GetRoutes() {
		route := route
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
					ps.logger.Err(err).Msg("Error transpiling module: " + mod.Name)
					return err
				}
			}
			if mod.Type == spec.ModuleTypeJavascript || mod.Type == spec.ModuleTypeTypescript {
				if program, err = goja.Compile(mod.Name, modPayload, true); err != nil {
					ps.logger.Err(err).Msg("Error compiling module: " + mod.Name)
					return err
				}
			} else {
				return errors.New("invalid module type: " + mod.Type.String())
			}

			testRtCtx := NewRuntimeContext(ps, route, mod)
			defer testRtCtx.Clean()
			err = extractors.SetupModuleEventLoop(ps.printer, testRtCtx)
			if err != nil {
				ps.logger.Err(err).
					Msgf("Error applying module '%s' changes", mod.Name)
				return err
			}
			ps.modPrograms.Insert(mod.Name+"/"+mod.Namespace.Name, program)
			elapsed := time.Since(start)
			ps.logger.Debug().
				Msgf("Module '%s/%s' changed applied in %s", mod.Name, mod.Namespace.Name, elapsed)
			return nil
		}
	}
	if err := eg.Wait(); err != nil {
		ps.logger.Err(err).Msg("Error setting up modules")
		return err
	}
	return nil
}

func (ps *ProxyState) setupRoutes() (err error) {
	ps.logger.Debug().Msg("Setting up routes")
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
					ps.logger.Err(err).Msg("Error creating module buffer")
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

		ps.logger.Trace().Msg("Routes have changed, reloading")
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
			ps.logger.Error().Msg("Error getting module program: invalid state")
			return nil, fmt.Errorf("cannot find module program: %s/%s", m.Name, r.Namespace.Name)
		} else {
			rtCtx := NewRuntimeContext(ps, r, r.Modules...)
			if err := extractors.SetupModuleEventLoop(ps.printer, rtCtx, program); err != nil {
				ps.logger.Err(err).Msg("Error creating runtime for route: " + reqCtx.route.Name)
				return nil, err
			} else {
				loop := rtCtx.EventLoop()
				errorHandler, err := extractors.ExtractErrorHandlerFunction(loop)
				if err != nil {
					ps.logger.Err(err).Msg("Error extracting error handler function")
					return nil, err
				}
				fetchUpstream, err := extractors.ExtractFetchUpstreamFunction(loop)
				if err != nil {
					ps.logger.Err(err).Msg("Error extracting fetch upstream function")
					return nil, err
				}
				reqModifier, err := extractors.ExtractRequestModifierFunction(loop)
				if err != nil {
					ps.logger.Err(err).Msg("Error extracting request modifier function")
					return nil, err
				}
				resModifier, err := extractors.ExtractResponseModifierFunction(loop)
				if err != nil {
					ps.logger.Err(err).Msg("Error extracting response modifier function")
					return nil, err
				}
				reqHandler, err := extractors.ExtractRequestHandlerFunction(loop)
				if err != nil {
					ps.logger.Err(err).Msg("Error extracting request handler function")
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

func (ps *ProxyState) startChangeLoop() {
	ps.proxyLock.Lock()
	if err := ps.reconfigureState(true, nil); err != nil {
		ps.logger.Err(err).Msg("Error initiating state")
		ps.Stop()
		return
	}
	ps.proxyLock.Unlock()

	for {
		log := <-ps.changeChan
		switch log.Cmd {
		case spec.ShutdownCommand:
			ps.logger.Warn().
				Msg("Shutdown command received, closing change loop")
			log.PushError(nil)
			return
		case spec.RestartCommand:
			ps.logger.Warn().
				Msg("Restart command received, not supported")
			// 	ps.logger.Warn().
			// 		Msg("Restart command received, restarting state")
			// 	go ps.RestartState(func(err error) {
			// 		ps.logger.Err(err).
			// 			Msg("Error restarting state")
			// 		os.Exit(1)
			// 	})
		}

		func() {
			ps.proxyLock.Lock()
			defer ps.proxyLock.Unlock()

			err := ps.reconfigureState(false, log)
			if log.PushError(err); err != nil {
				ps.logger.Err(err).
					Msgf("Error reconfiguring state @namespace:%s", log.Namespace)
				go ps.RestartState(func(err error) {
					ps.logger.Err(err).
						Msg("Error restarting state, exiting")
					ps.changeChan <- &spec.ChangeLog{
						Cmd: spec.ShutdownCommand,
					}
				})
			}
		}()
	}
}

func (ps *ProxyState) startProxyServer() {
	cfg := ps.config.ProxyConfig
	hostPort := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	ps.logger.Info().
		Msgf("Starting proxy server on %s", hostPort)
	proxyHttpLogger := Logger(&ps.logger,
		WithComponentLogger("proxy-http"),
	)
	server := &http.Server{
		Addr:     hostPort,
		Handler:  ps,
		ErrorLog: log.New(proxyHttpLogger, "", 0),
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
		ps.logger.Err(err).Msg("Error starting proxy server")
		os.Exit(1)
	}
}

func (ps *ProxyState) startProxyServerTLS() {
	cfg := ps.config.ProxyConfig
	if cfg.TLS == nil {
		return
	}
	hostPort := fmt.Sprintf("%s:%d", cfg.Host, cfg.TLS.Port)
	ps.logger.Info().
		Msgf("Starting secure proxy server on %s", hostPort)
	proxyHttpsLogger := Logger(&ps.logger,
		WithComponentLogger("proxy-https"),
		WithDefaultLevel(zerolog.InfoLevel),
	)
	secureServer := &http.Server{
		Addr:     hostPort,
		Handler:  ps,
		ErrorLog: log.New(proxyHttpsLogger, "", 0),
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
		ps.logger.Err(err).Msg("Error starting secure proxy server")
		os.Exit(1)
	}
}

func StartProxyGateway(version string, conf *config.DGateConfig) *ProxyState {
	ps := NewProxyState(conf)
	ps.version = version

	return ps
}

func (ps *ProxyState) Start() (err error) {
	defer func() {
		if err != nil {
			ps.Stop()
		}
	}()

	go ps.startChangeLoop()
	go ps.startProxyServer()
	go ps.startProxyServerTLS()

	ps.metrics.Setup(ps.config)
	if err = ps.store.InitStore(); err != nil {
		return err
	}

	if !ps.replicationEnabled {
		if err = ps.restoreFromChangeLogs(false); err != nil {
			return err
		}
	}

	return nil
}

func (ps *ProxyState) Stop() {
	cl := &spec.ChangeLog{
		Cmd: spec.ShutdownCommand,
	}
	done := make(chan error, 1)
	cl.SetErrorChan(done)
	// push change to change loop
	ps.changeChan <- cl
	// wait for change loop to stop
	<-done
}

func (ps *ProxyState) HandleRoute(requestCtxProvider *RequestContextProvider, pattern string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ctx, cancel := context.WithCancel(requestCtxPrdovider.ctx)
		// defer cancel()
		ps.ProxyHandlerFunc(ps, requestCtxProvider.
			CreateRequestContext(requestCtxProvider.ctx, w, r, pattern))
	}
}

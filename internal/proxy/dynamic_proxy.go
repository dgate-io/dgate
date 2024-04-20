package proxy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/dgate-io/dgate/internal/config"
	"github.com/dgate-io/dgate/internal/router"
	"github.com/dgate-io/dgate/pkg/modules/extractors"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/dgate-io/dgate/pkg/typescript"
	"github.com/dgate-io/dgate/pkg/util/sliceutil"
	"github.com/dgate-io/dgate/pkg/util/tree/avl"
	"github.com/dop251/goja"
	"github.com/rs/zerolog"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
)

func (state *ProxyState) reconfigureState(log *spec.ChangeLog) error {
	start := time.Now()
	err := state.setupModules()
	if err != nil {
		return err
	}
	err = state.setupRoutes()
	if err != nil {
		return err
	}
	if log != nil {
		state.logger.Info().
			Msgf("State reloaded in %s: %s",
				time.Since(start), log.Cmd)
	}
	return nil
}

func (ps *ProxyState) setupModules() error {
	ps.logger.Debug().Msg("Setting up modules")
	eg, _ := errgroup.WithContext(context.Background())
	newModPrograms := avl.NewTree[string, *goja.Program]()
	for _, route := range ps.rm.GetRoutes() {
		route := route
		for _, mod := range route.Modules {
			mod := mod
			eg.Go(func() error {
				var (
					err        error
					program    *goja.Program
					modPayload string = mod.Payload
				)
				start := time.Now()
				if mod.Type == spec.ModuleTypeTypescript {
					modPayload, err = typescript.Transpile(modPayload)
					if err != nil {
						ps.logger.Err(err).Msg("Error transpiling module: " + mod.Name)
						return err
					}
				}
				if mod.Type == spec.ModuleTypeJavascript || mod.Type == spec.ModuleTypeTypescript {
					program, err = goja.Compile(mod.Name+".js", modPayload, true)
					if err != nil {
						ps.logger.Err(err).Msg("Error compiling module: " + mod.Name)
						return err
					}
				} else {
					return errors.New("invalid module type: " + mod.Type.String())
				}

				rtCtx := NewRuntimeContext(ps, route, mod)
				loop, err := extractors.NewModuleEventLoop(ps.printer, rtCtx)
				if err != nil {
					ps.logger.Err(err).
						Msgf("Error applying module '%s' changes", mod.Name)
					return err
				}
				defer loop.Stop()
				newModPrograms.Insert(mod.Name+"/"+mod.Namespace.Name, program)
				elapsed := time.Since(start)
				ps.logger.Debug().
					Msgf("Module '%s/%s' changed applied in %s", mod.Name, mod.Namespace.Name, elapsed)
				return nil
			})
		}
	}
	if err := eg.Wait(); err != nil {
		ps.logger.Err(err).Msg("Error setting up modules")
		return err
	} else {
		ps.modPrograms = newModPrograms

	}
	return nil
}

func (ps *ProxyState) setupRoutes() (err error) {
	ps.logger.Debug().Msg("Setting up routes")
	reqCtxProviders := avl.NewTree[string, *RequestContextProvider]()
	for namespaceName, routes := range ps.rm.GetRouteNamespaceMap() {
		mux := router.NewMux()
		for _, r := range routes {
			reqCtxProvider := NewRequestContextProvider(r)
			reqCtxProviders.Insert(r.Namespace.Name+"/"+r.Name, reqCtxProvider)
			if len(r.Modules) > 0 {
				modBuf, err := NewModuleBuffer(
					256, 1024, reqCtxProvider,
					createModuleExtractorFunc(ps, r),
				)
				if err != nil {
					ps.logger.Err(err).Msg("Error creating module buffer")
					return err
				}
				reqCtxProvider.SetModuleBuffer(modBuf)
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
						}
						// TODO: validate methods
						for _, method := range r.Methods {
							mux.Method(method, path, ps.HandleRoute(reqCtxProvider, path))
						}
					}
				}
				return nil
			}()
		}

		ps.logger.Info().Msg("Routes have changed, reloading")
		if dr, ok := ps.routers.Find(namespaceName); ok {
			dr.ReplaceMux(mux)
		} else {
			dr := router.NewRouterWithMux(mux)
			ps.routers.Insert(namespaceName, dr)
		}
	}
	return
}

func createModuleExtractorFunc(ps *ProxyState, r *spec.DGateRoute) ModuleExtractorFunc {
	return func(reqCtx *RequestContextProvider) ModuleExtractor {
		programs := sliceutil.SliceMapper(r.Modules, func(m *spec.DGateModule) *goja.Program {
			program, ok := ps.modPrograms.Find(m.Name + "/" + r.Namespace.Name)
			if !ok {
				ps.logger.Error().Msg("Error getting module program: invalid state")
				panic("Error getting module program: invalid state")
			}
			return program
		})
		rtCtx := NewRuntimeContext(ps, r, r.Modules...)
		if loop, err := extractors.NewModuleEventLoop(ps.printer, rtCtx, programs...); err != nil {
			ps.logger.Err(err).Msg("Error creating runtime for route: " + reqCtx.route.Name)
			return nil
		} else {
			errorHandler, err := extractors.ExtractErrorHandlerFunction(loop)
			if err != nil {
				ps.logger.Err(err).Msg("Error extracting error handler function")
				return nil
			}
			fetchUpstream, err := extractors.ExtractFetchUpstreamFunction(loop)
			if err != nil {
				ps.logger.Err(err).Msg("Error extracting fetch upstream function")
				return nil
			}
			reqModifier, err := extractors.ExtractRequestModifierFunction(loop)
			if err != nil {
				ps.logger.Err(err).Msg("Error extracting request modifier function")
				return nil
			}
			resModifier, err := extractors.ExtractResponseModifierFunction(loop)
			if err != nil {
				ps.logger.Err(err).Msg("Error extracting response modifier function")
				return nil
			}
			reqHandler, err := extractors.ExtractRequestHandlerFunction(loop)
			if err != nil {
				ps.logger.Err(err).Msg("Error extracting request handler function")
				return nil
			}
			return NewModuleExtractor(
				rtCtx, fetchUpstream,
				reqModifier, resModifier,
				errorHandler, reqHandler,
			)
		}
	}
}

func StartProxyGateway(conf *config.DGateConfig) (*ProxyState, error) {
	ps := NewProxyState(conf)

	go func() {
		ps.proxyLock.Lock()
		err := ps.reconfigureState(nil)
		if err != nil {
			ps.logger.Err(err).Msg("Error initiating state")
			os.Exit(1)
		}
		ps.proxyLock.Unlock()

		for {
			ps.status = ProxyStatusRunning
			var log = <-ps.changeChan
			if ps.status == ProxyStatusStopping || ps.status == ProxyStatusClosed {
				ps.logger.Info().Msg("Stopping proxy gracefully :D")
				os.Exit(0)
				return
			}
			func() {
				ps.proxyLock.Lock()
				defer ps.proxyLock.Unlock()
				ps.status = ProxyStatusModifying
				var err error
				defer func() { log.PushError(err) }()
				err = ps.reconfigureState(log)
				if err != nil {
					ps.logger.Err(err).Msg("Error reconfiguring state")
					ps.rollbackChange(log)
				}
			}()
		}
	}()

	err := ps.store.InitStore()
	if err != nil {
		return nil, err
	}
	if !ps.replicationEnabled {
		err = ps.restoreFromChangeLogs()
		if err != nil {
			return nil, err
		}
	}

	go func() {
		config := conf.ProxyConfig
		hostPort := fmt.Sprintf("%s:%d", config.Host, config.Port)
		ps.logger.Info().
			Msgf("Starting proxy server on %s", hostPort)
		proxyHttpLogger := ps.Logger(
			WithComponentLogger("proxy-http"),
			WithDefaultLevel(zerolog.InfoLevel),
		)
		server := &http.Server{
			Addr:     hostPort,
			Handler:  ps,
			ErrorLog: log.New(proxyHttpLogger, "", 0),
		}
		if config.EnableHTTP2 {
			h2Server := &http2.Server{}
			err := http2.ConfigureServer(server, h2Server)
			if err != nil {
				panic(err)
			}
			if config.EnableH2C {
				server.Handler = h2c.NewHandler(ps, h2Server)
			}
		}
		if err := server.ListenAndServe(); err != nil {
			ps.logger.Err(err).Msg("Error starting proxy server")
			os.Exit(1)
		}
	}()

	if conf.ProxyConfig.TLS != nil {
		go func() {
			config := conf.ProxyConfig
			hostPort := fmt.Sprintf("%s:%d", config.Host, config.TLS.Port)
			ps.logger.Info().
				Msgf("Starting secure proxy server on %s", hostPort)
			proxyHttpsLogger := ps.Logger(
				WithComponentLogger("proxy-https"),
				WithDefaultLevel(zerolog.InfoLevel),
			)
			secureServer := &http.Server{
				Addr:     hostPort,
				Handler:  ps,
				ErrorLog: log.New(proxyHttpsLogger, "", 0),
				TLSConfig: ps.DynamicTLSConfig(
					conf.ProxyConfig.TLS.CertFile,
					conf.ProxyConfig.TLS.KeyFile,
				),
			}
			if config.EnableHTTP2 {
				h2Server := &http2.Server{}
				err := http2.ConfigureServer(secureServer, h2Server)
				if err != nil {
					panic(err)
				}
				if config.EnableH2C {
					secureServer.Handler = h2c.NewHandler(ps, h2Server)
				}
			}
			if err := secureServer.ListenAndServeTLS("", ""); err != nil {
				ps.logger.Err(err).Msg("Error starting secure proxy server")
				os.Exit(1)
			}
		}()
	}

	return ps, nil
}

func (ps *ProxyState) HandleRoute(requestCtxProvider *RequestContextProvider, pattern string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(requestCtxProvider.ctx)
		defer cancel()
		reqCtx := requestCtxProvider.
			CreateRequestContext(ctx, w, r, pattern)
		ps.ProxyHandlerFunc(ps, reqCtx)
	}
}

func TimeoutDialer(connTimeout time.Duration) func(net, addr string) (c net.Conn, err error) {
	return func(_net, _addr string) (net.Conn, error) {
		conn, err := net.DialTimeout(_net, _addr, connTimeout)
		if err != nil {
			return nil, err
		}
		return conn, nil
	}
}

func RouteHash(routes ...*spec.DGateRoute) (hash uint32, err error) {
	hash, err = HashAny[*spec.Route](0, spec.TransformDGateRoutes(routes...))
	return
}

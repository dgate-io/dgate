package admin

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dgate-io/chi-router"
	"github.com/dgate-io/chi-router/middleware"
	"github.com/dgate-io/dgate/internal/config"
	"github.com/dgate-io/dgate/internal/proxy"
	"github.com/dgate-io/dgate/pkg/util"
	"github.com/rs/zerolog"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func StartAdminAPI(conf *config.DGateConfig, proxyState *proxy.ProxyState) {
	if conf.AdminConfig == nil {
		proxyState.Logger().Warn().
			Msg("Admin API is disabled")
		return
	}

	// Start HTTP Server
	mux := chi.NewRouter()
	configureRoutes(mux, proxyState, conf)

	// Start HTTPS Server
	go func() {
		if conf.AdminConfig.TLS != nil {
			adminHttpsLogger := proxyState.Logger(
				proxy.WithComponentLogger("admin-https"),
				proxy.WithDefaultLevel(zerolog.InfoLevel),
			)
			secureHostPort := fmt.Sprintf("%s:%d",
				conf.AdminConfig.Host, conf.AdminConfig.TLS.Port)
			secureServer := &http.Server{
				Addr:     secureHostPort,
				Handler:  mux,
				ErrorLog: log.New(adminHttpsLogger, "", 0),
			}
			proxyState.Logger().Info().
				Msgf("Starting secure admin api on %s", secureHostPort)
			proxyState.Logger().Debug().
				Msgf("TLS Cert: %s", conf.AdminConfig.TLS.CertFile)
			proxyState.Logger().Debug().
				Msgf("TLS Key: %s", conf.AdminConfig.TLS.KeyFile)
			if err := secureServer.ListenAndServeTLS(
				conf.AdminConfig.TLS.CertFile,
				conf.AdminConfig.TLS.KeyFile,
			); err != nil {
				panic(err)
			}
		}
	}()

	// Start Test Server
	if conf.TestServerConfig != nil {
		if !conf.Debug {
			proxyState.Logger().Warn().
				Msg("Test server is disabled in non-debug mode")
		} else {
			go func() {
				testHostPort := fmt.Sprintf("%s:%d",
					conf.TestServerConfig.Host, conf.TestServerConfig.Port)
				mux := chi.NewRouter()
				mux.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
					if strings.HasPrefix(r.URL.Path, "/debug") {
						// strip /debug prefix
						r.URL.Path = strings.TrimPrefix(r.URL.Path, "/debug")
						middleware.Profiler().ServeHTTP(w, r)
						return
					}
					respMap := map[string]any{}
					if waitStr := r.URL.Query().Get("wait"); waitStr != "" {
						if waitTime, err := time.ParseDuration(waitStr); err != nil {
							util.JsonResponse(w, http.StatusBadRequest, map[string]string{
								"error": fmt.Sprintf("Invalid wait time: %s", waitStr),
							})
							return
						} else {
							respMap["waited"] = waitTime.String()
							time.Sleep(waitTime)
						}
					}
					respMap["method"] = r.Method
					respMap["path"] = r.URL.String()
					if body, err := io.ReadAll(r.Body); err == nil {
						respMap["body"] = string(body)
					}
					respMap["host"] = r.Host
					respMap["remote_addr"] = r.RemoteAddr
					respMap["req_headers"] = r.Header
					if conf.TestServerConfig.EnableEnvVars {
						respMap["env"] = os.Environ()
					}
					respMap["global_headers"] = conf.TestServerConfig.GlobalHeaders
					for k, v := range conf.TestServerConfig.GlobalHeaders {
						w.Header().Set(k, v)
					}
					util.JsonResponse(w, http.StatusOK, respMap)
				})

				testServerLogger := proxyState.Logger(
					proxy.WithComponentLogger("test-server-http"),
					proxy.WithDefaultLevel(zerolog.InfoLevel),
				)
				testServer := &http.Server{
					Addr:     testHostPort,
					Handler:  mux,
					ErrorLog: log.New(testServerLogger, "", 0),
				}
				if conf.TestServerConfig.EnableHTTP2 {
					h2Server := &http2.Server{}
					err := http2.ConfigureServer(testServer, h2Server)
					if err != nil {
						panic(err)
					}
					if conf.TestServerConfig.EnableH2C {
						testServer.Handler = h2c.NewHandler(mux, h2Server)
					}
				}
				proxyState.Logger().Info().
					Msgf("Starting test server on %s", testHostPort)

				if err := testServer.ListenAndServe(); err != nil {
					panic(err)
				}
			}()
		}
	}
	go func() {
		adminHttpLogger := proxyState.Logger(
			proxy.WithComponentLogger("admin-http"),
			proxy.WithDefaultLevel(zerolog.InfoLevel),
		)
		hostPort := fmt.Sprintf("%s:%d",
			conf.AdminConfig.Host, conf.AdminConfig.Port)
		proxyState.Logger().Info().
			Msgf("Starting admin api on %s", hostPort)
		server := &http.Server{
			Addr:     hostPort,
			Handler:  mux,
			ErrorLog: log.New(adminHttpLogger, "", 0),
		}
		if err := server.ListenAndServe(); err != nil {
			panic(err)
		}
	}()
}

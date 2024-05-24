package admin

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dgate-io/chi-router"
	"github.com/dgate-io/chi-router/middleware"
	"github.com/dgate-io/dgate/internal/admin/changestate"
	"github.com/dgate-io/dgate/internal/config"
	"github.com/dgate-io/dgate/pkg/util"

	"log/slog"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func StartAdminAPI(conf *config.DGateConfig, cs changestate.ChangeState) {
	if conf.AdminConfig == nil {
		cs.Logger().Warn("Admin API is disabled")
		return
	}

	// Start HTTP Server
	mux := chi.NewRouter()
	configureRoutes(mux, cs, conf)

	// Start HTTPS Server
	go func() {
		if conf.AdminConfig.TLS != nil {
			adminHttpsLogHandler := cs.Logger().
				Handler().WithGroup("admin-https")
			secureHostPort := fmt.Sprintf("%s:%d",
				conf.AdminConfig.Host, conf.AdminConfig.TLS.Port)
			secureServer := &http.Server{
				Addr:     secureHostPort,
				Handler:  mux,
				ErrorLog: slog.NewLogLogger(adminHttpsLogHandler, slog.LevelInfo),
			}
			cs.Logger().Info("Starting secure admin api on" + secureHostPort)
			cs.Logger().Debug("TLS Cert",
				"cert_file", conf.AdminConfig.TLS.CertFile,
				"key_file", conf.AdminConfig.TLS.KeyFile,
			)
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
			cs.Logger().Warn("Test server is disabled in non-debug mode")
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

				testServerLogger := cs.Logger().
					WithGroup("test-server")

				testServer := &http.Server{
					Addr:     testHostPort,
					Handler:  mux,
					ErrorLog: slog.NewLogLogger(testServerLogger.Handler(), slog.LevelInfo),
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
				cs.Logger().Info("Starting test server on " + testHostPort)

				if err := testServer.ListenAndServe(); err != nil {
					panic(err)
				}
			}()
		}
	}
	go func() {
		adminHttpLogger := cs.Logger().WithGroup("admin-http")
		hostPort := fmt.Sprintf("%s:%d",
			conf.AdminConfig.Host, conf.AdminConfig.Port)
		cs.Logger().Info("Starting admin api on " + hostPort)
		server := &http.Server{
			Addr:     hostPort,
			Handler:  mux,
			ErrorLog: slog.NewLogLogger(adminHttpLogger.Handler(), slog.LevelInfo),
		}
		if err := server.ListenAndServe(); err != nil {
			panic(err)
		}
	}()
}

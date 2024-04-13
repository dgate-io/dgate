package admin

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"runtime"
	"strings"

	"github.com/dgate-io/chi-router"
	"github.com/dgate-io/dgate/internal/admin/routes"
	"github.com/dgate-io/dgate/internal/config"
	"github.com/dgate-io/dgate/internal/proxy"
	"github.com/dgate-io/dgate/pkg/util"
	"github.com/dgate-io/dgate/pkg/util/iplist"
	"github.com/hashicorp/raft"
)

func configureRoutes(server *chi.Mux, proxyState *proxy.ProxyState, conf *config.DGateConfig) {
	adminConfig := conf.AdminConfig
	server.Use(func(next http.Handler) http.Handler {
		ipList := iplist.NewIPList()
		for _, address := range adminConfig.AllowList {
			if strings.Contains(address, "/") {
				err := ipList.AddCIDRString(address)
				if err != nil {
					panic(fmt.Sprintf("invalid cidr address in admin.allow_list: %s", address))
				}
			} else {
				err := ipList.AddIPString(address)
				if err != nil {
					panic(fmt.Sprintf("invalid ip address in admin.allow_list: %s", address))
				}
			}
		}
		// basic auth
		var userMap map[string]string
		// key auth
		var keyMap map[string]struct{}

		switch adminConfig.AuthMethod {
		case config.AuthMethodBasicAuth:
			userMap = make(map[string]string)
			if len(adminConfig.BasicAuth.Users) > 0 {
				for i, user := range adminConfig.BasicAuth.Users {
					if user.Username == "" || user.Password == "" {
						panic(fmt.Sprintf("both username and password are required: admin.basic_auth.users[%d]", i))
					}
					userMap[user.Username] = user.Password
				}
			}
		case config.AuthMethodKeyAuth:
			keyMap = make(map[string]struct{})
			if adminConfig.KeyAuth != nil && len(adminConfig.KeyAuth.Keys) > 0 {
				if adminConfig.KeyAuth.QueryParamName != "" && adminConfig.KeyAuth.HeaderName != "" {
					panic("only one of admin.key_auth.query_param_name or admin.key_auth.header_name can be set")
				}
				for _, key := range adminConfig.KeyAuth.Keys {
					keyMap[key] = struct{}{}
				}
			}
		case config.AuthMethodJWTAuth:
			panic("JWT Auth is not supported yet")
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ipList.Len() > 0 {
				remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					remoteHost = r.RemoteAddr
				}
				allowed, err := ipList.Contains(remoteHost)
				if !allowed && adminConfig.XForwardedForDepth > 0 {
					xForwardedForIps := r.Header.Values("X-Forwarded-For")
					count := min(adminConfig.XForwardedForDepth, len(xForwardedForIps))
					for i := 0; i < count; i++ {
						allowed, err = ipList.Contains(xForwardedForIps[i])
						if err != nil {
							proxyState.Logger().Error().Err(err).Msgf("error checking x-forwarded-for ip: %s", xForwardedForIps[i])
							if conf.Debug {
								http.Error(w, "Bad Request: could not parse x-forwarded-for IP address", http.StatusBadRequest)
							}
							http.Error(w, "Bad Request", http.StatusBadRequest)
							return
						}
						if allowed {
							break
						}
					}
				}

				if err != nil {
					if conf.Debug {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				if !allowed {
					if conf.Debug {
						http.Error(w, "Unauthorized IP Address: "+remoteHost, http.StatusUnauthorized)
						return
					}
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			}
			// basic auth
			if adminConfig.AuthMethod == config.AuthMethodBasicAuth {
				if len(adminConfig.BasicAuth.Users) == 0 {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				user, pass, ok := r.BasicAuth()
				if userPass, userFound := userMap[user]; !ok || !userFound || userPass != pass {
					w.Header().Set("WWW-Authenticate", `Basic realm="Access to DGate Admin API"`)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			} else if adminConfig.KeyAuth != nil && len(adminConfig.KeyAuth.Keys) > 0 {
				// key auth
				var key string
				if adminConfig.KeyAuth.QueryParamName != "" {
					key = r.URL.Query().Get(adminConfig.KeyAuth.QueryParamName)
				} else if adminConfig.KeyAuth.HeaderName != "" {
					key = r.Header.Get(adminConfig.KeyAuth.HeaderName)
				} else {
					key = r.Header.Get("X-API-Key")
				}
				if _, keyFound := keyMap[key]; !keyFound {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			}
			raftInstance := proxyState.Raft()
			if r.Method == http.MethodPut && raftInstance != nil {
				leader := raftInstance.Leader()
				if leader == "" {
					util.JsonError(w, http.StatusServiceUnavailable, "raft: no leader")
					return
				}
				if raftInstance.State() != raft.Leader {
					r.URL.Host = string(leader)
					http.Redirect(w, r, r.URL.String(), http.StatusTemporaryRedirect)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	})

	server.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("X-DGate-Raft", fmt.Sprintf("%t", proxyState.Raft() != nil))
		w.Header().Set("X-DGate-WatchOnly", fmt.Sprintf("%t", adminConfig.WatchOnly))
		w.Header().Set("X-DGate-ChangeHash", fmt.Sprintf("%d", proxyState.ChangeHash()))
		w.Header().Set("X-DGate-AdminAPI", "true")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("DGate Admin API"))
	}))

	if adminConfig.Replication != nil {
		setupRaft(conf, server, proxyState)
	}
	if adminConfig != nil && !adminConfig.WatchOnly {
		server.Route("/api/v1", func(api chi.Router) {
			routes.ConfigureRouteAPI(api, proxyState, conf)
			routes.ConfigureModuleAPI(api, proxyState, conf)
			routes.ConfigureServiceAPI(api, proxyState, conf)
			routes.ConfigureNamespaceAPI(api, proxyState, conf)
			routes.ConfigureDomainAPI(api, proxyState, conf)
			routes.ConfigureCollectionAPI(api, proxyState, conf)
		})
	}

	server.Group(func(misc chi.Router) {
		routes.ConfigureChangeLogAPI(misc, proxyState, conf)
		routes.ConfigureHealthAPI(misc, proxyState, conf)

		misc.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
			snapshot := proxyState.Stats().Snapshot()
			b, err := json.Marshal(snapshot)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Error getting stats: " + err.Error()))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(b))
		})

		misc.Get("/system", func(w http.ResponseWriter, r *http.Request) {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			systemStats := map[string]any{
				"goroutines":         runtime.NumGoroutine(),
				"mem_alloc_mb":       bToMb(m.Alloc),
				"mem_total_alloc_mb": bToMb(m.TotalAlloc),
				"mem_sys_mb":         bToMb(m.Sys),
				"mem_num_gc":         m.NumGC,
			}
			b, err := json.Marshal(systemStats)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Error getting stats: " + err.Error()))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(b))
		})
	})
}

func bToMb(b uint64) float64 {
	v := float64(b) / 1048576
	return math.Round(v*100) / 100
}

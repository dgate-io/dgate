package routes

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/dgate-io/chi-router"
	"github.com/dgate-io/dgate/internal/config"
	"github.com/dgate-io/dgate/internal/proxy"
	"github.com/dgate-io/dgate/pkg/util"
)

func ConfigureChangeLogAPI(server chi.Router, proxyState *proxy.ProxyState, appConfig *config.DGateConfig) {
	server.Get("/changelog/hash", func(w http.ResponseWriter, r *http.Request) {
		if repl := proxyState.Raft(); repl != nil {
			future := repl.Barrier(time.Second * 5)
			if err := future.Error(); err != nil {
				util.JsonError(w, http.StatusInternalServerError, err.Error())
				return
			}
			// TODO: find a way to get the raft log hash
			//  perhaps generate based on current log commands and computed hash
		}

		if b, err := json.Marshal(map[string]any{
			"hash": proxyState.ChangeHash(),
		}); err != nil {
			util.JsonError(w, http.StatusInternalServerError, err.Error())
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(b))
		}
	})
	server.Get("/changelog/rm", func(w http.ResponseWriter, r *http.Request) {
		if b, err := json.Marshal(proxyState.ResourceManager()); err != nil {
			util.JsonError(w, http.StatusInternalServerError, err.Error())
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(b))
		}
	})
}

func ConfigureHealthAPI(server chi.Router, ps *proxy.ProxyState, _ *config.DGateConfig) {
	healthlyResp := []byte(
		`{"status":"ok","version":"` +
			ps.Version() + `"}`,
	)
	server.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(healthlyResp)
	})

	server.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r := ps.Raft(); r != nil {
			if r.Leader() == "" {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"status":"no leader"}`))
				return
			} else if !ps.Ready() {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"status":"not ready"}`))
				return
			}
		}
		w.Write(healthlyResp)
	})
}

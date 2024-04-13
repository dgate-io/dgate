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
		}
		b, err := json.Marshal(map[string]any{
			"hash": proxyState.ChangeHash(),
		})
		if err != nil {
			util.JsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(b))
	})
}

func ConfigureHealthAPI(server chi.Router, ps *proxy.ProxyState, _ *config.DGateConfig) {
	server.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	server.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if r := ps.Raft(); r != nil {
			if r.Leader() == "" {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"status":"no leader"}`))
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
}

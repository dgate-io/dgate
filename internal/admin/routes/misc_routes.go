package routes

import (
	"encoding/json"
	"net/http"

	"github.com/dgate-io/chi-router"
	"github.com/dgate-io/dgate/internal/admin/changestate"
	"github.com/dgate-io/dgate/internal/config"
	"github.com/dgate-io/dgate/pkg/util"
)

func ConfigureChangeLogAPI(
	server chi.Router,
	cs changestate.ChangeState,
	conf *config.DGateConfig,
) {
	server.Get("/changelog/hash", func(w http.ResponseWriter, r *http.Request) {
		if b, err := json.Marshal(map[string]any{
			"hash": cs.ChangeHash(),
		}); err != nil {
			util.JsonError(w, http.StatusInternalServerError, err.Error())
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(b))
		}
	})
}

func ConfigureHealthAPI(server chi.Router, version string, cs changestate.ChangeState) {
	healthlyResp := []byte(`{"status":"ok"}`)
	server.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(healthlyResp)
	})

	server.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !cs.Ready() {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"not ready"}`))
			return
		}
		w.Write(healthlyResp)
	})
}

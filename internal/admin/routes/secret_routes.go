package routes

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/dgate-io/chi-router"
	"github.com/dgate-io/dgate/internal/config"
	"github.com/dgate-io/dgate/internal/proxy"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/dgate-io/dgate/pkg/util"
)

func ConfigureSecretAPI(server chi.Router, proxyState *proxy.ProxyState, appConfig *config.DGateConfig) {
	rm := proxyState.ResourceManager()
	server.Put("/secret", func(w http.ResponseWriter, r *http.Request) {
		eb, err := io.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			util.JsonError(w, http.StatusBadRequest, "error reading body")
			return
		}
		scrt := spec.Secret{}
		err = json.Unmarshal(eb, &scrt)
		if err != nil {
			util.JsonError(w, http.StatusBadRequest, "error unmarshalling body")
			return
		}
		if scrt.Data == "" {
			util.JsonError(w, http.StatusBadRequest, "payload is required")
			return
		}
		if scrt.NamespaceName == "" {
			if appConfig.DisableDefaultNamespace {
				util.JsonError(w, http.StatusBadRequest, "namespace is required")
				return
			}
			scrt.NamespaceName = spec.DefaultNamespace.Name
		}
		cl := spec.NewChangeLog(&scrt, scrt.NamespaceName, spec.AddSecretCommand)
		if err = proxyState.ApplyChangeLog(cl); err != nil {
			util.JsonError(w, http.StatusBadRequest, err.Error())
			return
		}
		if repl := proxyState.Raft(); repl != nil {
			future := repl.Barrier(time.Second * 5)
			if err := future.Error(); err != nil {
				util.JsonError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		util.JsonResponse(w, http.StatusCreated,
			spec.TransformDGateSecrets(true,
				rm.GetSecretsByNamespace(scrt.NamespaceName)...))
	})

	server.Delete("/secret", func(w http.ResponseWriter, r *http.Request) {
		eb, err := io.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			util.JsonError(w, http.StatusBadRequest, "error reading body")
			return
		}
		scrt := spec.Secret{}
		err = json.Unmarshal(eb, &scrt)
		if err != nil {
			util.JsonError(w, http.StatusBadRequest, "error unmarshalling body")
			return
		}
		if scrt.NamespaceName == "" {
			if appConfig.DisableDefaultNamespace {
				util.JsonError(w, http.StatusBadRequest, "namespace is required")
				return
			}
			scrt.NamespaceName = spec.DefaultNamespace.Name
		}
		cl := spec.NewChangeLog(&scrt, scrt.NamespaceName, spec.DeleteSecretCommand)
		if err = proxyState.ApplyChangeLog(cl); err != nil {
			util.JsonError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})

	server.Get("/secret", func(w http.ResponseWriter, r *http.Request) {
		nsName := r.URL.Query().Get("namespace")
		if nsName == "" {
			if appConfig.DisableDefaultNamespace {
				util.JsonError(w, http.StatusBadRequest, "namespace is required")
				return
			}
			nsName = spec.DefaultNamespace.Name
		} else {
			if _, ok := rm.GetNamespace(nsName); !ok {
				util.JsonError(w, http.StatusBadRequest, "namespace not found: "+nsName)
				return
			}
		}
		util.JsonResponse(w, http.StatusCreated,
			spec.TransformDGateSecrets(true,
				rm.GetSecretsByNamespace(nsName)...))
	})
}

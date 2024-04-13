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

func ConfigureServiceAPI(server chi.Router, proxyState *proxy.ProxyState, appConfig *config.DGateConfig) {
	rm := proxyState.ResourceManager()
	server.Put("/service", func(w http.ResponseWriter, r *http.Request) {
		eb, err := io.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			util.JsonError(w, http.StatusBadRequest, "error reading body")
			return
		}
		svc := spec.Service{}
		err = json.Unmarshal(eb, &svc)
		if err != nil {
			util.JsonError(w, http.StatusBadRequest, "error unmarshalling body")
			return
		}
		if svc.Retries == nil {
			retries := 3
			svc.Retries = &retries
		} else {
			if *svc.Retries < 0 {
				util.JsonError(w, http.StatusBadRequest, "retries must be greater than 0")
				return
			}
		}
		if svc.RetryTimeout != nil && *svc.RetryTimeout < 0 {
			util.JsonError(w, http.StatusBadRequest, "retry timeout must be greater than 0")
			return
		}
		if svc.NamespaceName == "" {
			if appConfig.DisableDefaultNamespace {
				util.JsonError(w, http.StatusBadRequest, "namespace is required")
				return
			}
			svc.NamespaceName = spec.DefaultNamespace.Name
		}
		cl := spec.NewChangeLog(&svc, svc.NamespaceName, spec.AddServiceCommand)
		err = proxyState.ApplyChangeLog(cl)
		if err != nil {
			util.JsonError(w, http.StatusBadRequest, err.Error())
			return
		}

		if repl := proxyState.Raft(); repl != nil {
			proxyState.Logger().Debug().
				Msg("Waiting for raft barrier")
			future := repl.Barrier(time.Second * 5)
			if err := future.Error(); err != nil {
				util.JsonError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		util.JsonResponse(w, http.StatusCreated, spec.TransformDGateServices(rm.GetServicesByNamespace(svc.NamespaceName)...))
	})

	server.Delete("/service", func(w http.ResponseWriter, r *http.Request) {
		eb, err := io.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			util.JsonError(w, http.StatusBadRequest, "error reading body")
			return
		}
		svc := spec.Service{}
		err = json.Unmarshal(eb, &svc)
		if err != nil {
			util.JsonError(w, http.StatusBadRequest, "error unmarshalling body")
			return
		}
		if svc.NamespaceName == "" {
			if appConfig.DisableDefaultNamespace {
				util.JsonError(w, http.StatusBadRequest, "namespace is required")
				return
			}
			svc.NamespaceName = spec.DefaultNamespace.Name
		}
		cl := spec.NewChangeLog(&svc, svc.NamespaceName, spec.DeleteServiceCommand)
		err = proxyState.ApplyChangeLog(cl)
		if err != nil {
			util.JsonError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	server.Get("/service", func(w http.ResponseWriter, r *http.Request) {
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
				util.JsonError(w, http.StatusBadRequest, "namespace not found: "+nsName)
				return
			}
		}
		util.JsonResponse(w, http.StatusOK, spec.TransformDGateServices(rm.GetServicesByNamespace(nsName)...))
	})
}

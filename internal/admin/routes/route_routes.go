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

func ConfigureRouteAPI(server chi.Router, proxyState *proxy.ProxyState, appConfig *config.DGateConfig) {
	rm := proxyState.ResourceManager()
	server.Put("/route", func(w http.ResponseWriter, r *http.Request) {
		eb, err := io.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			util.JsonError(w, http.StatusBadRequest, "error reading body")
			return
		}
		route := spec.Route{}
		err = json.Unmarshal(eb, &route)
		if err != nil {
			util.JsonError(w, http.StatusBadRequest, "error unmarshalling body")
			return
		}

		if route.Name == "" {
			util.JsonError(w, http.StatusBadRequest, "name is required")
			return
		}

		if route.NamespaceName == "" {
			if appConfig.DisableDefaultNamespace {
				util.JsonError(w, http.StatusBadRequest, "namespace is required")
				return
			}
			route.NamespaceName = spec.DefaultNamespace.Name
		}

		cl := spec.NewChangeLog(&route, route.NamespaceName, spec.AddRouteCommand)
		err = proxyState.ApplyChangeLog(cl)
		if err != nil {
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

		util.JsonResponse(w, http.StatusCreated, spec.TransformDGateRoutes(rm.GetRoutesByNamespace(route.NamespaceName)...))
	})

	server.Delete("/route", func(w http.ResponseWriter, r *http.Request) {
		eb, err := io.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			util.JsonError(w, http.StatusBadRequest, "error reading body")
			return
		}
		route := spec.Route{}
		err = json.Unmarshal(eb, &route)
		if err != nil {
			util.JsonError(w, http.StatusBadRequest, "error unmarshalling body")
			return
		}

		if route.Name == "" {
			util.JsonError(w, http.StatusBadRequest, "name is required")
			return
		}

		if route.NamespaceName == "" {
			if appConfig.DisableDefaultNamespace {
				util.JsonError(w, http.StatusBadRequest, "namespace is required")
				return
			}
			route.NamespaceName = spec.DefaultNamespace.Name
		}

		cl := spec.NewChangeLog(&route, route.NamespaceName, spec.DeleteRouteCommand)
		err = proxyState.ApplyChangeLog(cl)
		if err != nil {
			util.JsonError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})

	server.Get("/route", func(w http.ResponseWriter, r *http.Request) {
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
		b, err := json.Marshal(spec.TransformDGateRoutes(rm.GetRoutesByNamespace(nsName)...))
		if err != nil {
			util.JsonError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(b))
	})
}

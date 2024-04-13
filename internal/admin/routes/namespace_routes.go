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

func ConfigureNamespaceAPI(server chi.Router, proxyState *proxy.ProxyState, _ *config.DGateConfig) {
	rm := proxyState.ResourceManager()
	server.Put("/namespace", func(w http.ResponseWriter, r *http.Request) {
		eb, err := io.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			util.JsonError(w, http.StatusBadRequest, "error reading body")
			return
		}
		namespace := spec.Namespace{}
		err = json.Unmarshal(eb, &namespace)
		if err != nil {
			util.JsonError(w, http.StatusBadRequest, "error unmarshalling body")
			return
		}

		if namespace.Name == "" {
			util.JsonError(w, http.StatusBadRequest, "name is required")
			return
		}

		cl := spec.NewChangeLog(&namespace, namespace.Name, spec.AddNamespaceCommand)
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

		util.JsonResponse(w, http.StatusCreated, spec.TransformDGateNamespaces(rm.GetNamespaces()...))
	})

	server.Delete("/namespace", func(w http.ResponseWriter, r *http.Request) {
		eb, err := io.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil {
			util.JsonError(w, http.StatusBadRequest, "error reading body")
			return
		}
		namespace := spec.Namespace{}
		err = json.Unmarshal(eb, &namespace)
		if err != nil {
			util.JsonError(w, http.StatusBadRequest, "error unmarshalling body")
			return
		}

		if namespace.Name == "" {
			util.JsonError(w, http.StatusBadRequest, "name is required")
			return
		}

		cl := spec.NewChangeLog(&namespace, namespace.Name, spec.DeleteNamespaceCommand)
		err = proxyState.ApplyChangeLog(cl)
		if err != nil {
			util.JsonError(w, http.StatusBadRequest, err.Error())
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})

	server.Get("/namespace", func(w http.ResponseWriter, r *http.Request) {
		util.JsonResponse(w, http.StatusOK, spec.TransformDGateNamespaces(rm.GetNamespaces()...))
	})
}

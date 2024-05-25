package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dgate-io/chi-router"
	"github.com/dgate-io/dgate/internal/admin/changestate"
	"github.com/dgate-io/dgate/internal/config"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/dgate-io/dgate/pkg/util"
	"go.uber.org/zap"
)

func ConfigureServiceAPI(server chi.Router, logger *zap.Logger, cs changestate.ChangeState, appConfig *config.DGateConfig) {
	rm := cs.ResourceManager()
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
			if ute, ok := err.(*json.UnmarshalTypeError); ok {
				err = fmt.Errorf("error unmarshalling body: field %s expected type %s", ute.Field, ute.Type.String())
				util.JsonError(w, http.StatusBadRequest, err.Error())
				return
			} else if se, ok := err.(*json.SyntaxError); ok {
				err = fmt.Errorf("error unmarshalling body: syntax error at byte offset %d", se.Offset)
				util.JsonError(w, http.StatusBadRequest, err.Error())
				return
			} else {
				util.JsonError(w, http.StatusBadRequest, "error unmarshalling body")
				return
			}
		}
		if svc.Retries == nil {
			retries := 3
			svc.Retries = &retries
		} else if *svc.Retries < 0 {
			util.JsonError(w, http.StatusBadRequest, "retries must be greater than 0")
			return
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
		err = cs.ApplyChangeLog(cl)
		if err != nil {
			util.JsonError(w, http.StatusBadRequest, err.Error())
			return
		}

		if repl := cs.Raft(); repl != nil {
			logger.Debug("Waiting for raft barrier")
			future := repl.Barrier(time.Second * 5)
			if err := future.Error(); err != nil {
				util.JsonError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		svcs := rm.GetServicesByNamespace(svc.NamespaceName)
		util.JsonResponse(w, http.StatusCreated,
			spec.TransformDGateServices(svcs...))
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
		err = cs.ApplyChangeLog(cl)
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
				return
			}
		}
		util.JsonResponse(w, http.StatusOK, spec.TransformDGateServices(rm.GetServicesByNamespace(nsName)...))
	})

	server.Get("/service/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		nsName := r.URL.Query().Get("namespace")
		if nsName == "" {
			if appConfig.DisableDefaultNamespace {
				util.JsonError(w, http.StatusBadRequest, "namespace is required")
				return
			}
			nsName = spec.DefaultNamespace.Name
		}
		svc, ok := rm.GetService(name, nsName)
		if !ok {
			util.JsonError(w, http.StatusNotFound, "service not found")
			return
		}
		util.JsonResponse(w, http.StatusOK, spec.TransformDGateService(svc))
	})
}

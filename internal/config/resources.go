package config

import (
	"errors"

	"github.com/dgate-io/dgate/pkg/spec"
)

func (resources *DGateResources) Validate() error {
	if resources == nil || resources.SkipValidation {
		return nil
	}
	namespaces := make(map[string]*spec.Namespace)
	services := make(map[string]*spec.Service)
	routes := make(map[string]*spec.Route)
	domains := make(map[string]*spec.Domain)
	modules := make(map[string]*spec.Module)
	collections := make(map[string]*spec.Collection)
	documents := make(map[string]*spec.Document)

	for _, ns := range resources.Namespaces {
		if _, ok := namespaces[ns.Name]; ok {
			return errors.New("duplicate namespace: " + ns.Name)
		}
		if ns.Name == "" {
			return errors.New("namespace name must be specified")
		}
		namespaces[ns.Name] = &ns
	}

	for _, mod := range resources.Modules {
		key := mod.Name + "-" + mod.NamespaceName
		if _, ok := modules[key]; ok {
			return errors.New("duplicate module: " + mod.Name)
		}
		if mod.Name == "" {
			return errors.New("module name must be specified")
		}
		if mod.NamespaceName != "" {
			if _, ok := namespaces[mod.NamespaceName]; !ok {
				return errors.New("module (" + mod.Name + ") references non-existent namespace (" + mod.NamespaceName + ")")
			}
		} else {
			return errors.New("module (" + mod.Name + ") must specify namespace")
		}
		if mod.Payload == "" && mod.PayloadFile == "" {
			return errors.New("module payload or payload file must be specified")
		}
		if mod.Payload != "" && mod.PayloadFile != "" {
			return errors.New("module payload and payload file cannot both be specified")
		}
		modules[key] = &mod.Module
	}

	for _, svc := range resources.Services {
		key := svc.Name + "-" + svc.NamespaceName
		if _, ok := services[key]; ok {
			return errors.New("duplicate service: " + svc.Name)
		}
		if svc.Name == "" {
			return errors.New("service name must be specified")
		}
		if svc.NamespaceName != "" {
			if _, ok := namespaces[svc.NamespaceName]; !ok {
				return errors.New("service (" + svc.Name + ") references non-existent namespace (" + svc.NamespaceName + ")")
			}
		} else {
			return errors.New("service (" + svc.Name + ") must specify namespace")
		}
		services[key] = &svc
	}

	for _, route := range resources.Routes {
		key := route.Name + "-" + route.NamespaceName
		if _, ok := routes[key]; ok {
			return errors.New("duplicate route: " + route.Name)
		}
		if route.Name == "" {
			return errors.New("route name must be specified")
		}
		if route.ServiceName != "" {
			if _, ok := services[route.ServiceName+"-"+route.NamespaceName]; !ok {
				return errors.New("route (" + route.Name + ") references non-existent service (" + route.ServiceName + ")")
			}
		}
		if route.NamespaceName != "" {
			if _, ok := namespaces[route.NamespaceName]; !ok {
				return errors.New("route (" + route.Name + ") references non-existent namespace (" + route.NamespaceName + ")")
			}
		} else {
			return errors.New("route (" + route.Name + ") must specify namespace")
		}
		for _, modName := range route.Modules {
			if _, ok := modules[modName+"-"+route.NamespaceName]; !ok {
				return errors.New("route (" + route.Name + ") references non-existent module (" + modName + ")")
			}
		}
		routes[key] = &route
	}

	for _, dom := range resources.Domains {
		key := dom.Name + "-" + dom.NamespaceName
		if _, ok := domains[key]; ok {
			return errors.New("duplicate domain: " + dom.Name)
		}
		if dom.Name == "" {
			return errors.New("domain name must be specified")
		}
		if dom.NamespaceName != "" {
			if _, ok := namespaces[dom.NamespaceName]; !ok {
				return errors.New("domain (" + dom.Name + ") references non-existent namespace (" + dom.NamespaceName + ")")
			}
		} else {
			return errors.New("domain (" + dom.Name + ") must specify namespace")
		}
		if dom.Cert != "" && dom.CertFile != "" {
			return errors.New("domain cert and cert file cannot both be specified")
		}
		if dom.Key != "" && dom.KeyFile != "" {
			return errors.New("domain key and key file cannot both be specified")
		}
		if (dom.Cert == "") != (dom.Key == "") {
			return errors.New("domain cert (file) and key (file) must both be specified, or neither")
		}
		domains[key] = &dom.Domain
	}

	for _, col := range resources.Collections {
		key := col.Name + "-" + col.NamespaceName
		if _, ok := collections[key]; ok {
			return errors.New("duplicate collection: " + col.Name)
		}
		if col.Name == "" {
			return errors.New("collection name must be specified")
		}
		if col.NamespaceName != "" {
			if _, ok := namespaces[col.NamespaceName]; !ok {
				return errors.New("collection (" + col.Name + ") references non-existent namespace (" + col.NamespaceName + ")")
			}
		} else {
			return errors.New("collection (" + col.Name + ") must specify namespace")
		}
		if col.Schema == nil {
			return errors.New("collection (" + col.Name + ") must specify schema")
		}
		if col.Type != spec.CollectionTypeDocument && col.Type != spec.CollectionTypeFetcher {
			return errors.New("collection (" + col.Name + ") must specify type")
		}
		if col.Visibility != spec.CollectionVisibilityPublic && col.Visibility != spec.CollectionVisibilityPrivate {
			return errors.New("collection (" + col.Name + ") must specify visibility")
		}
		for _, modName := range col.Modules {
			if _, ok := modules[modName+"-"+col.NamespaceName]; !ok {
				return errors.New("collection (" + col.Name + ") references non-existent module (" + modName + ")")
			}
		}
		collections[key] = &col
	}

	for _, doc := range resources.Documents {
		key := doc.ID + "-" + doc.NamespaceName
		if _, ok := documents[key]; ok {
			return errors.New("duplicate document: " + doc.ID)
		}
		if doc.ID == "" {
			return errors.New("document ID must be specified")
		}
		if doc.NamespaceName != "" {
			if _, ok := namespaces[doc.NamespaceName]; !ok {
				return errors.New("document (" + doc.ID + ") references non-existent namespace (" + doc.NamespaceName + ")")
			}
		} else {
			return errors.New("document (" + doc.ID + ") must specify namespace")
		}
		if doc.CollectionName != "" {
			if _, ok := collections[doc.CollectionName+"-"+doc.NamespaceName]; !ok {
				return errors.New("document (" + doc.ID + ") references non-existent collection (" + doc.CollectionName + ")")
			}
		}
		documents[key] = &doc
	}

	return nil
}

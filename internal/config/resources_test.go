package config

import (
	"testing"

	"github.com/dgate-io/dgate/pkg/spec"
)

func TestValidate(t *testing.T) {
	resources := &DGateResources{
		SkipValidation: false,
		Namespaces: []spec.Namespace{
			{
				Name: "default",
				Tags: []string{"default"},
			},
		},
		Services: []spec.Service{
			{
				Name:          "default",
				NamespaceName: "default",
				Tags:          []string{"default"},
			},
		},
		Routes: []spec.Route{
			{
				Name:          "default",
				Tags:          []string{"default"},
				NamespaceName: "default",
			},
		},
		Domains: []DomainSpec{
			{
				Domain: spec.Domain{
					Name:          "default",
					NamespaceName: "default",
					Tags:          []string{"default"},
				},
			},
		},
		Modules: []ModuleSpec{
			{
				Module: spec.Module{
					Name:          "default",
					NamespaceName: "default",
					Payload:       "void(0)",
					Tags:          []string{"default"},
				},
			},
		},
	}
	err := resources.Validate()
	if err != nil {
		t.Error(err)
	}
}

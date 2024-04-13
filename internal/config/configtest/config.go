package configtest

import (
	"github.com/dgate-io/dgate/internal/config"
	"github.com/dgate-io/dgate/pkg/spec"
)

func NewTestDGateConfig() *config.DGateConfig {
	return &config.DGateConfig{
		LogLevel: "panic",
		Debug:    true,
		Version:  "v1",
		Tags:     []string{"test"},
		Storage: config.DGateStorageConfig{
			StorageType: config.StorageTypeMemory,
		},
		ProxyConfig: config.DGateProxyConfig{
			Host: "localhost",
			Port: 8080,
			InitResources: &config.DGateResources{
				Namespaces: []spec.Namespace{
					{
						Name: "test",
					},
				},
				Routes: []spec.Route{
					{
						Name:          "test",
						Paths:         []string{"/", "/test"},
						Methods:       []string{"GET", "PUT"},
						Modules:       []string{"test"},
						ServiceName:   "test",
						NamespaceName: "test",
						Tags:          []string{"test"},
					},
				},
				Services: []spec.Service{
					{
						Name:          "test",
						URLs:          []string{"http://localhost:8080"},
						NamespaceName: "test",
						Tags:          []string{"test"},
					},
				},
				Modules: []config.ModuleSpec{
					{
						Module: spec.Module{
							Name:          "test",
							NamespaceName: "test",
							Payload:       EmptyAsyncModuleFunctionsTS,
							Tags:          []string{"test"},
						},
					},
				},
			},
		},
	}
}

func NewTest2DGateConfig() *config.DGateConfig {
	return &config.DGateConfig{
		LogLevel: "panic",
		Debug:    true,
		Version:  "v1",
		Tags:     []string{"test"},
		Storage: config.DGateStorageConfig{
			StorageType: config.StorageTypeMemory,
		},
		ProxyConfig: config.DGateProxyConfig{
			Host: "localhost",
			Port: 16436,
			InitResources: &config.DGateResources{
				Namespaces: []spec.Namespace{
					{
						Name: "test",
					},
				},
				Routes: []spec.Route{
					{
						Name:          "test",
						Paths:         []string{"/", "/test"},
						Methods:       []string{"GET", "PUT"},
						Modules:       []string{"test"},
						NamespaceName: "test",
						Tags:          []string{"test"},
					},
				},
				Modules: []config.ModuleSpec{
					{
						Module: spec.Module{
							Name:          "test",
							NamespaceName: "test",
							Payload:       EmptyAsyncModuleFunctionsTS,
							Tags:          []string{"test"},
						},
					},
				},
			},
		},
	}
}


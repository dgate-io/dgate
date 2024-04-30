package configtest

import (
	"github.com/dgate-io/dgate/internal/config"
	"github.com/dgate-io/dgate/pkg/spec"
)

func NewTestDGateConfig() *config.DGateConfig {
	return &config.DGateConfig{
		LogLevel:                "disabled",
		DisableDefaultNamespace: true,
		Debug:                   true,
		Version:                 "v1",
		Tags:                    []string{"test"},
		Storage: config.DGateStorageConfig{
			StorageType: config.StorageTypeMemory,
		},
		ProxyConfig: config.DGateProxyConfig{
			AllowedDomains: []string{"*test.com", "localhost"},
			Host:           "localhost",
			Port:           8080,
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
	conf := NewTestDGateConfig()
	conf.ProxyConfig = config.DGateProxyConfig{
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
	}
	return conf
}

func NewTestDGateConfig_DomainAndNamespaces() *config.DGateConfig {
	conf := NewTestDGateConfig()
	conf.ProxyConfig.InitResources.Namespaces = []spec.Namespace{
		{Name: "test"}, {Name: "test2"}, {Name: "test3"},
	}
	conf.ProxyConfig.InitResources.Domains = []config.DomainSpec{
		{
			Domain: &spec.Domain{
				Name:          "test-dm",
				NamespaceName: "test",
				Patterns:      []string{"example.com"},
				Priority:      1,
				Tags:          []string{"test"},
			},
		},
		{
			Domain: &spec.Domain{
				Name:          "test-dm2",
				NamespaceName: "test2",
				Patterns:      []string{`*test.com`},
				Priority:      2,
				Tags:          []string{"test"},
			},
		},
		{
			Domain: &spec.Domain{
				Name:          "test-dm3",
				NamespaceName: "test3",
				Patterns:      []string{`/^(abc|cba).test.com$/`},
				Priority:      3,
				Tags:          []string{"test"},
			},
			CertFile: "testdata/domain.crt",
			KeyFile:  "testdata/domain.key",
		},
	}
	return conf
}

func NewTestDGateConfig_DomainAndNamespaces2() *config.DGateConfig {
	conf := NewTestDGateConfig_DomainAndNamespaces()
	conf.DisableDefaultNamespace = false
	return conf
}

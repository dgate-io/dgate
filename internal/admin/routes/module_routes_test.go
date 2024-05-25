package routes_test

import (
	"encoding/base64"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/dgate-io/chi-router"
	"github.com/dgate-io/dgate/internal/admin/changestate/testutil"
	"github.com/dgate-io/dgate/internal/admin/routes"
	"github.com/dgate-io/dgate/internal/config/configtest"
	"github.com/dgate-io/dgate/internal/proxy"
	"github.com/dgate-io/dgate/pkg/dgclient"
	"github.com/dgate-io/dgate/pkg/resources"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func TestAdminRoutes_Module(t *testing.T) {
	namespaces := []string{"default", "test"}
	for _, ns := range namespaces {
		config := configtest.NewTest3DGateConfig()
		ps := proxy.NewProxyState(zap.NewNop(), config)
		mux := chi.NewMux()
		mux.Route("/api/v1", func(r chi.Router) {
			routes.ConfigureModuleAPI(r, zap.NewNop(), ps, config)
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		client := dgclient.NewDGateClient()
		if err := client.Init(server.URL, server.Client(),
			dgclient.WithVerboseLogging(true),
		); err != nil {
			t.Fatal(err)
		}

		if err := client.CreateModule(&spec.Module{
			Name:          "test",
			NamespaceName: ns,
			Payload: base64.StdEncoding.EncodeToString(
				[]byte("\"use test\""),
			),
			Type: spec.ModuleTypeJavascript,
			Tags: []string{"test123"},
		}); err != nil {
			t.Fatal(err)
		}
		rm := ps.ResourceManager()
		if _, ok := rm.GetModule("test", ns); !ok {
			t.Fatal("module not found")
		}
		if modules, err := client.ListModule(ns); err != nil {
			t.Fatal(err)
		} else {
			mods := rm.GetModulesByNamespace(ns)
			assert.Equal(t, len(mods), len(modules))
			assert.Equal(t, spec.TransformDGateModules(mods...), modules)
		}
		if module, err := client.GetModule("test", ns); err != nil {
			t.Fatal(err)
		} else {
			mod, ok := rm.GetModule("test", ns)
			assert.True(t, ok)
			module2 := spec.TransformDGateModule(mod)
			assert.Equal(t, module2, module)
		}
		if err := client.DeleteModule("test", ns); err != nil {
			t.Fatal(err)
		} else if _, ok := rm.GetModule("test", ns); ok {
			t.Fatal("module not deleted")
		}
	}
}

func TestAdminRoutes_ModuleError(t *testing.T) {
	namespaces := []string{"default", "test", ""}
	for _, ns := range namespaces {
		config := configtest.NewTest3DGateConfig()
		rm := resources.NewManager()
		cs := testutil.NewMockChangeState()
		cs.On("ApplyChangeLog", mock.Anything).
			Return(errors.New("test error"))
		cs.On("ResourceManager").Return(rm)
		mux := chi.NewMux()
		mux.Route("/api/v1", func(r chi.Router) {
			routes.ConfigureModuleAPI(r, zap.NewNop(), cs, config)
		})
		server := httptest.NewServer(mux)
		defer server.Close()

		client := dgclient.NewDGateClient()
		if err := client.Init(server.URL, server.Client(),
			dgclient.WithVerboseLogging(true),
		); err != nil {
			t.Fatal(err)
		}

		if err := client.CreateModule(&spec.Module{
			Name:          "test",
			NamespaceName: ns,
			Payload:       "\"use test\"",
			Type:          spec.ModuleTypeJavascript,
			Tags:          []string{"test123"},
		}); err == nil {
			t.Fatal("expected error")
		}
		if _, err := client.GetModule("", ns); err == nil {
			t.Fatal("expected error")
		}
		if err := client.DeleteModule("", ns); err == nil {
			t.Fatal("expected error")
		}
	}
}

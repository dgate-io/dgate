package commands

import (
	"os"
	"strings"
	"testing"

	"github.com/dgate-io/dgate/pkg/dgclient"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/stretchr/testify/mock"
)

const version = "test"

func TestGenericCommands(t *testing.T) {
	stdout := os.Stdout
	os.Stdout = os.NewFile(0, os.DevNull)
	defer func() { os.Stdout = stdout }()

	resources := []string{
		"namespace", "route", "service",
		"module", "domain", "secret",
		"collection", "document",
	}
	actions := []string{
		"get", "list",
		"create", "delete",
	}

	for _, resource := range resources {
		for _, action := range actions {
			os.Args = []string{
				"dgate-cli",
				"--admin=localhost.com",
				resource, action,
			}
			switch action {
			case "delete", "get", "create":
				if resource == "document" {
					os.Args = append(
						os.Args,
						"id=test",
					)
				} else {
					os.Args = append(os.Args, "name=test")
				}
			}
			if resource == "document" {
				os.Args = append(
					os.Args,
					"collection=test",
				)
			}
			if action == "create" {
				switch resource {
				case "route":
					os.Args = append(os.Args, "paths=/", "methods=GET")
				case "service":
					os.Args = append(os.Args, "urls=http://localhost.net")
				case "module":
					os.Args = append(os.Args, "payload=QUJD")
				case "domain":
					os.Args = append(os.Args, "patterns=*")
				case "secret":
					os.Args = append(os.Args, "data=123")
				case "collection":
					os.Args = append(os.Args, "schema:={}")
				case "document":
					os.Args = append(os.Args, "data:={}")
				}
			}
			mockClient := new(mockDGClient)
			funcName := firstUpper(action) + firstUpper(resource)
			mockClient.On(
				funcName,
				mock.Anything,
				mock.Anything,
				mock.Anything,
			)
			mockClient.On("Init", "localhost.com").Return(nil)
			Run(mockClient, version)
		}
	}
}

func firstUpper(s string) string {
	if len(s) <= 1 {
		return strings.ToUpper(s)
	}
	rs := []rune(s)
	firstChar := strings.ToUpper(string(rs[0]))
	return firstChar + string(rs[1:])
}

type mockDGClient struct {
	mock.Mock
}

var _ dgclient.DGateClient = &mockDGClient{}

func (m *mockDGClient) Init(baseUrl string, opts ...dgclient.Options) error {
	args := m.Called(baseUrl)
	if len(args) == 0 {
		return nil
	}
	return args.Error(0)
}

func (m *mockDGClient) BaseUrl() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockDGClient) GetRoute(name, namespace string) (*spec.Route, error) {
	args := m.Called(name, namespace)
	if len(args) == 0 {
		return nil, nil
	}
	return args[0].(*spec.Route), args.Error(1)
}

func (m *mockDGClient) CreateRoute(rt *spec.Route) error {
	args := m.Called(rt)
	if len(args) == 0 {
		return nil
	}
	return args.Error(0)
}

func (m *mockDGClient) DeleteRoute(name, namespace string) error {
	args := m.Called(name, namespace)
	if len(args) == 0 {
		return nil
	}
	return args.Error(0)
}

func (m *mockDGClient) ListRoute(namespace string) ([]*spec.Route, error) {
	args := m.Called(namespace)
	if len(args) == 0 {
		return nil, nil
	}
	return args[0].([]*spec.Route), args.Error(1)
}

func (m *mockDGClient) GetNamespace(name string) (*spec.Namespace, error) {
	args := m.Called(name)
	if len(args) == 0 {
		return nil, nil
	}
	return args[0].(*spec.Namespace), args.Error(1)
}

func (m *mockDGClient) CreateNamespace(ns *spec.Namespace) error {
	args := m.Called(ns)
	if len(args) == 0 {
		return nil
	}
	return args.Error(0)
}

func (m *mockDGClient) DeleteNamespace(name string) error {
	args := m.Called(name)
	if len(args) == 0 {
		return nil
	}
	return args.Error(0)
}
func (m *mockDGClient) ListNamespace() ([]*spec.Namespace, error) {
	args := m.Called()
	if len(args) == 0 {
		return nil, nil
	}
	return args[0].([]*spec.Namespace), args.Error(1)
}

func (m *mockDGClient) CreateSecret(sec *spec.Secret) error {
	args := m.Called(sec)
	if len(args) == 0 {
		return nil
	}
	return args.Error(0)
}

func (m *mockDGClient) DeleteSecret(name, namespace string) error {
	args := m.Called(name, namespace)
	if len(args) == 0 {
		return nil
	}
	return args.Error(0)
}

func (m *mockDGClient) GetSecret(name, namespace string) (*spec.Secret, error) {
	args := m.Called(name, namespace)
	if len(args) == 0 {
		return nil, nil
	}
	return args[0].(*spec.Secret), args.Error(1)
}

func (m *mockDGClient) ListSecret(namespace string) ([]*spec.Secret, error) {
	args := m.Called(namespace)
	if len(args) == 0 {
		return nil, nil
	}
	return args[0].([]*spec.Secret), args.Error(1)
}
func (m *mockDGClient) GetService(name string, namespace string) (*spec.Service, error) {
	args := m.Called(name, namespace)
	if len(args) == 0 {
		return nil, nil
	}
	return args[0].(*spec.Service), args.Error(1)
}

func (m *mockDGClient) CreateService(svc *spec.Service) error {
	args := m.Called(svc)
	if len(args) == 0 {
		return nil
	}
	return args.Error(0)
}

func (m *mockDGClient) DeleteService(name, namespace string) error {
	args := m.Called(name, namespace)
	if len(args) == 0 {
		return nil
	}
	return args.Error(0)
}

func (m *mockDGClient) ListService(namespace string) ([]*spec.Service, error) {
	args := m.Called(namespace)
	if len(args) == 0 {
		return nil, nil
	}
	return args[0].([]*spec.Service), args.Error(1)
}

func (m *mockDGClient) GetModule(name, namespace string) (*spec.Module, error) {
	args := m.Called(name, namespace)
	if len(args) == 0 {
		return nil, nil
	}
	return args[0].(*spec.Module), args.Error(1)
}

func (m *mockDGClient) CreateModule(mod *spec.Module) error {
	args := m.Called(mod)
	if len(args) == 0 {
		return nil
	}
	return args.Error(0)
}

func (m *mockDGClient) DeleteModule(name, namespace string) error {
	args := m.Called(name, namespace)
	if len(args) == 0 {
		return nil
	}
	return args.Error(0)
}

func (m *mockDGClient) ListModule(namespace string) ([]*spec.Module, error) {
	args := m.Called(namespace)
	if len(args) == 0 {
		return nil, nil
	}
	return args[0].([]*spec.Module), args.Error(1)
}

func (m *mockDGClient) CreateDomain(domain *spec.Domain) error {
	args := m.Called(domain)
	if len(args) == 0 {
		return nil
	}
	return args.Error(0)
}

func (m *mockDGClient) DeleteDomain(name, namespace string) error {
	args := m.Called(name, namespace)
	if len(args) == 0 {
		return nil
	}
	return args.Error(0)
}

func (m *mockDGClient) GetDomain(name, namespace string) (*spec.Domain, error) {
	args := m.Called(name, namespace)
	if len(args) == 0 {
		return nil, nil
	}
	return args[0].(*spec.Domain), args.Error(1)
}

func (m *mockDGClient) ListDomain(namespace string) ([]*spec.Domain, error) {
	args := m.Called(namespace)
	if len(args) == 0 {
		return nil, nil
	}
	return args[0].([]*spec.Domain), args.Error(1)
}

func (m *mockDGClient) CreateCollection(svc *spec.Collection) error {
	args := m.Called(svc)
	if len(args) == 0 {
		return nil
	}
	return args.Error(0)
}

func (m *mockDGClient) DeleteCollection(name, namespace string) error {
	args := m.Called(name, namespace)
	if len(args) == 0 {
		return nil
	}
	return args.Error(0)
}

func (m *mockDGClient) ListCollection(namespace string) ([]*spec.Collection, error) {
	args := m.Called(namespace)
	if len(args) == 0 {
		return nil, nil
	}
	return args[0].([]*spec.Collection), args.Error(1)
}

func (m *mockDGClient) GetCollection(name, namespace string) (*spec.Collection, error) {
	args := m.Called(name, namespace)
	if len(args) == 0 {
		return nil, nil
	}
	return args[0].(*spec.Collection), args.Error(1)
}

func (m *mockDGClient) CreateDocument(doc *spec.Document) error {
	args := m.Called(doc)
	if len(args) == 0 {
		return nil
	}
	return args.Error(0)
}

func (m *mockDGClient) DeleteDocument(id, collection, namespace string) error {
	args := m.Called(id, collection, namespace)
	if len(args) == 0 {
		return nil
	}
	return args.Error(0)
}

func (m *mockDGClient) ListDocument(namespace, collection string) ([]*spec.Document, error) {
	args := m.Called(namespace)
	if len(args) == 0 {
		return nil, nil
	}
	return args[0].([]*spec.Document), args.Error(1)
}

func (m *mockDGClient) GetDocument(id, collection, namespace string) (*spec.Document, error) {
	args := m.Called(id, collection, namespace)
	if len(args) == 0 {
		return nil, nil
	}
	return args[0].(*spec.Document), args.Error(1)
}

func (m *mockDGClient) DeleteAllDocument(namespace string, collection string) error {
	args := m.Called(namespace)
	if len(args) == 0 {
		return nil
	}
	return args.Error(0)
}

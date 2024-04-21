package spec

import (
	"crypto/tls"
	"errors"
	"net/url"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

type Name string

type Named interface {
	GetName() string
}

type DGateRoute struct {
	Name         string          `json:"name"`
	Paths        []string        `json:"paths"`
	Methods      []string        `json:"methods"`
	StripPath    bool            `json:"stripPath"`
	PreserveHost bool            `json:"preserveHost"`
	Service      *DGateService   `json:"service"`
	Namespace    *DGateNamespace `json:"namespace"`
	Modules      []*DGateModule  `json:"modules"`
	Tags         []string        `json:"tags,omitempty"`
}

func (r *DGateRoute) GetName() string {
	return r.Name
}

type DGateService struct {
	Name      string          `json:"name"`
	URLs      []*url.URL      `json:"urls"`
	Tags      []string        `json:"tags,omitempty"`
	Namespace *DGateNamespace `json:"namespace"`

	DisableQueryParams bool          `json:"disableQueryParams,omitempty"`
	Retries            int           `json:"retries,omitempty"`
	RetryTimeout       time.Duration `json:"retryTimeout,omitempty"`
	ConnectTimeout     time.Duration `json:"connectTimeout,omitempty"`
	RequestTimeout     time.Duration `json:"requestTimeout,omitempty"`
	TLSSkipVerify      bool          `json:"tlsSkipVerify,omitempty"`
	HTTP2Only          bool          `json:"http2_only,omitempty"`
	HideDGateHeaders   bool          `json:"hideDGateHeaders,omitempty"`
}

func (s *DGateService) GetName() string {
	return s.Name
}

type DGateNamespace struct {
	Name string   `json:"name"`
	Tags []string `json:"tags,omitempty"`
}

func (ns *DGateNamespace) GetName() string {
	return ns.Name
}

type DGateDomain struct {
	Name      string           `json:"name"`
	Namespace *DGateNamespace  `json:"namespace"`
	Patterns  []string         `json:"pattern"`
	TLSCert   *tls.Certificate `json:"tls_config"`
	Priority  int             `json:"priority"`
	Cert      string           `json:"cert"`
	Key       string           `json:"key"`
	Tags      []string         `json:"tags,omitempty"`
}

var DefaultNamespace = &Namespace{
	Name: "default",
	Tags: []string{"default"},
}

type ModuleType string

const (
	ModuleTypeJavascript ModuleType = "javascript"
	ModuleTypeTypescript ModuleType = "typescript"
	// ModuleTypeWasm       ModuleType = "wasm"
)

func (m ModuleType) Valid() bool {
	switch m {
	case ModuleTypeJavascript, ModuleTypeTypescript:
		return true
	default:
		return false
	}
}

func (m ModuleType) String() string {
	return string(m)
}

type DGateModule struct {
	Name        string          `json:"name"`
	Namespace   *DGateNamespace `json:"namespace"`
	Payload     string          `json:"payload"`
	Type        ModuleType      `json:"module_type"`
	Permissions []string        `json:"permissions,omitempty"`
	Tags        []string        `json:"tags,omitempty"`
}

func (m *DGateModule) GetName() string {
	return m.Name
}

type DGateCollection struct {
	Name          string               `json:"name"`
	Namespace     *DGateNamespace      `json:"namespace"`
	Schema        *jsonschema.Schema   `json:"schema"`
	SchemaPayload string               `json:"schema_payload"`
	Type          CollectionType       `json:"type"`
	Visibility    CollectionVisibility `json:"visibility"`
	// Modules       []*DGateModule       `json:"modules"`
	Tags []string `json:"tags,omitempty"`
}

func (n *DGateCollection) GetName() string {
	return n.Name
}

type DGateDocument struct {
	ID         string           `json:"id"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
	Namespace  *DGateNamespace  `json:"namespace"`
	Collection *DGateCollection `json:"collection"`
	Data       string           `json:"data"`
}

func (n *DGateDocument) GetName() string {
	return n.ID
}

func ErrNamespaceNotFound(ns string) error {
	return errors.New("namespace not found: " + ns)
}

func ErrCollectionNotFound(col string) error {
	return errors.New("collection not found: " + col)
}

func ErrDocumentNotFound(doc string) error {
	return errors.New("document not found: " + doc)
}

func ErrModuleNotFound(mod string) error {
	return errors.New("module not found: " + mod)
}

func ErrServiceNotFound(svc string) error {
	return errors.New("service not found: " + svc)
}

func ErrRouteNotFound(route string) error {
	return errors.New("route not found: " + route)
}

func ErrDomainNotFound(domain string) error {
	return errors.New("domain not found: " + domain)
}

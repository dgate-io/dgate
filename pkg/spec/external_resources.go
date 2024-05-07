package spec

import (
	"time"
)

type Namespace struct {
	Name string   `json:"name" koanf:"name"`
	Tags []string `json:"tags,omitempty" koanf:"tags"`
}

func (n *Namespace) GetName() string {
	return n.Name
}

type Service struct {
	Name               string         `json:"name" koanf:"name"`
	URLs               []string       `json:"urls" koanf:"urls"`
	NamespaceName      string         `json:"namespace" koanf:"namespace"`
	Retries            *int           `json:"retries,omitempty" koanf:"retries"`
	RetryTimeout       *time.Duration `json:"retryTimeout,omitempty" koanf:"retryTimeout"`
	ConnectTimeout     *time.Duration `json:"connectTimeout,omitempty"  koanf:"connectTimeout"`
	RequestTimeout     *time.Duration `json:"requestTimeout,omitempty"  koanf:"requestTimeout"`
	TLSSkipVerify      *bool          `json:"tlsSkipVerify,omitempty" koanf:"tlsSkipVerify"`
	HTTP2Only          *bool          `json:"http2Only,omitempty" koanf:"http2Only"`
	HideDGateHeaders   *bool          `json:"hideDGateHeaders,omitempty" koanf:"hideDGateHeaders"`
	DisableQueryParams *bool          `json:"disableQueryParams,omitempty" koanf:"disableQueryParams"`
	Tags               []string       `json:"tags,omitempty" koanf:"tags"`
}

func (s *Service) GetName() string {
	return s.Name
}

type Route struct {
	Name          string   `json:"name" koanf:"name"`
	Paths         []string `json:"paths" koanf:"paths"`
	Methods       []string `json:"methods" koanf:"methods"`
	Schemes       []string `json:"schemes,omitempty" koanf:"schemes"`
	PreserveHost  bool     `json:"preserveHost" koanf:"preserveHost"`
	StripPath     bool     `json:"stripPath" koanf:"stripPath"`
	ServiceName   string   `json:"service,omitempty" koanf:"service"`
	NamespaceName string   `json:"namespace" koanf:"namespace"`
	Modules       []string `json:"modules,omitempty" koanf:"modules"`
	Tags          []string `json:"tags,omitempty" koanf:"tags"`
}

func (m *Route) GetName() string {
	return m.Name
}

type Module struct {
	Name          string     `json:"name" koanf:"name"`
	NamespaceName string     `json:"namespace" koanf:"namespace"`
	Payload       string     `json:"payload" koanf:"payload"`
	Type          ModuleType `json:"moduleType,omitempty" koanf:"moduleType"`
	Permissions   []string   `json:"permissions,omitempty" koanf:"permissions"`
	Tags          []string   `json:"tags,omitempty" koanf:"tags"`
}

func (m *Module) GetName() string {
	return m.Name
}

type Domain struct {
	Name          string   `json:"name" koanf:"name"`
	NamespaceName string   `json:"namespace" koanf:"namespace"`
	Patterns      []string `json:"patterns" koanf:"patterns"`
	Priority      int      `json:"priority" koanf:"priority"`
	Cert          string   `json:"cert" koanf:"cert"`
	Key           string   `json:"key" koanf:"key"`
	Tags          []string `json:"tags,omitempty" koanf:"tags"`
}

func (n *Domain) GetName() string {
	return n.Name
}

type Collection struct {
	Name          string               `json:"name" koanf:"name"`
	NamespaceName string               `json:"namespace" koanf:"namespace"`
	Schema        any                  `json:"schema" koanf:"schema"`
	Type          CollectionType       `json:"type" koanf:"type"`
	Visibility    CollectionVisibility `json:"visibility" koanf:"visibility"`
	// Modules       []string             `json:"modules,omitempty" koanf:"modules"`
	Tags []string `json:"tags,omitempty" koanf:"tags"`
}

type CollectionType string

const (
	CollectionTypeDocument CollectionType = "document"
	CollectionTypeFetcher  CollectionType = "fetcher"
)

type CollectionVisibility string

const (
	CollectionVisibilityPublic  CollectionVisibility = "public"
	CollectionVisibilityPrivate CollectionVisibility = "private"
)

func (n *Collection) GetName() string {
	return n.Name
}

type Document struct {
	ID             string    `json:"id"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	NamespaceName  string    `json:"namespace"`
	CollectionName string    `json:"collection"`
	Data           any       `json:"data"`
}

type Secret struct {
	Name          string   `json:"name"`
	NamespaceName string   `json:"namespace"`
	Data          string   `json:"data,omitempty"`
	Tags          []string `json:"tags,omitempty"`
}

func (n *Secret) GetName() string {
	return n.Name
}

func (n *Document) GetName() string {
	return n.ID
}

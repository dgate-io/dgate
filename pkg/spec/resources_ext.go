package spec

import (
	"encoding/json"
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
	Name          string   `json:"name" koanf:"name"`
	URLs          []string `json:"urls" koanf:"urls"`
	NamespaceName string   `json:"namespace" koanf:"namespace"`
	Tags          []string `json:"tags,omitempty" koanf:"tags"`

	Retries            *int           `json:"retries" koanf:"retries"`
	RetryTimeout       *time.Duration `json:"retryTimeout" koanf:"retryTimeout"`
	ConnectTimeout     *time.Duration `json:"connectTimeout"  koanf:"connectTimeout"`
	RequestTimeout     *time.Duration `json:"requestTimeout"  koanf:"requestTimeout"`
	TLSSkipVerify      *bool          `json:"tlsSkipVerify" koanf:"tlsSkipVerify"`
	HTTP2Only          *bool          `json:"http2Only" koanf:"http2Only"`
	HideDGateHeaders   *bool          `json:"hideDGateHeaders" koanf:"hideDGateHeaders"`
	DisableQueryParams *bool          `json:"disableQueryParams" koanf:"disableQueryParams"`
}

func (s *Service) GetName() string {
	return s.Name
}

type Route struct {
	Name          string   `json:"name" koanf:"name"`
	Paths         []string `json:"paths" koanf:"paths"`
	Methods       []string `json:"methods" koanf:"methods"`
	Schemes       []string `json:"schemes" koanf:"schemes"`
	PreserveHost  bool     `json:"preserveHost" koanf:"preserveHost"`
	StripPath     bool     `json:"stripPath" koanf:"stripPath"`
	ServiceName   string   `json:"service" koanf:"service"`
	NamespaceName string   `json:"namespace,omitempty" koanf:"namespace"`
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
	Priority      uint     `json:"priority" koanf:"priority"`
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
	Modules       []string             `json:"modules,omitempty" koanf:"modules"`
	Tags          []string             `json:"tags,omitempty" koanf:"tags"`
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

type RFC3339Time time.Time

func (t RFC3339Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).Format(time.RFC3339))
}

func (t *RFC3339Time) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	if str == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return err
	}

	*t = RFC3339Time(parsed)
	return nil
}

func (n *Document) GetName() string {
	return n.ID
}

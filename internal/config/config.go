package config

import (
	"time"

	"github.com/dgate-io/dgate/pkg/spec"
)

type (
	DGateConfig struct {
		Version                 string                 `koanf:"version"`
		LogLevel                string                 `koanf:"log_level"`
		Debug                   bool                   `koanf:"debug"`
		Tags                    []string               `koanf:"tags"`
		Storage                 DGateStorageConfig     `koanf:"storage"`
		ProxyConfig             DGateProxyConfig       `koanf:"proxy"`
		AdminConfig             *DGateAdminConfig      `koanf:"admin"`
		TestServerConfig        *DGateTestServerConfig `koanf:"test_server"`
		DisableDefaultNamespace bool                   `koanf:"disable_default_namespace"`
	}

	DGateProxyConfig struct {
		Host                     string                   `koanf:"host"`
		Port                     int                      `koanf:"port"`
		TLS                      *DGateTLSConfig          `koanf:"tls"`
		EnableH2C                bool                     `koanf:"enable_h2c"`
		EnableHTTP2              bool                     `koanf:"enable_http2"`
		EnableConsoleLogger      bool                     `koanf:"enable_console_logger"`
		RedirectHttpsDomains     []string                 `koanf:"redirect_https"`
		AllowedDomains           []string                 `koanf:"allowed_domains"`
		GlobalHeaders            map[string]string        `koanf:"global_headers"`
		Transport                DGateHttpTransportConfig `koanf:"client_transport"`
		InitResources            *DGateResources          `koanf:"init_resources"`
		DisableXForwardedHeaders bool                     `koanf:"disable_x_forwarded_headers"`
	}

	DGateTestServerConfig struct {
		Host          string            `koanf:"host"`
		Port          int               `koanf:"port"`
		EnableH2C     bool              `koanf:"enable_h2c"`
		EnableHTTP2   bool              `koanf:"enable_http2"`
		EnableEnvVars bool              `koanf:"enable_env_vars"`
		GlobalHeaders map[string]string `koanf:"global_headers"`
	}

	DGateNativeModulesConfig struct {
		Name string `koanf:"name"`
		Path string `koanf:"path"`
	}

	DGateAdminConfig struct {
		Host               string                  `koanf:"host"`
		Port               int                     `koanf:"port"`
		AllowList          []string                `koanf:"allow_list"`
		XForwardedForDepth int                     `koanf:"x_forwarded_for_depth"`
		WatchOnly          bool                    `koanf:"watch_only"`
		Replication        *DGateReplicationConfig `koanf:"replication,omitempty"`
		Dashboard          *DGateDashboardConfig   `koanf:"dashboard"`
		TLS                *DGateTLSConfig         `koanf:"tls"`
		AuthMethod         AuthMethodType          `koanf:"auth_method"`
		BasicAuth          *DGateBasicAuthConfig   `koanf:"basic_auth"`
		KeyAuth            *DGateKeyAuthConfig     `koanf:"key_auth"`
		JWTAuth            *DGateJWTAuthConfig     `koanf:"jwt_auth"`
	}

	DGateReplicationConfig struct {
		RaftID           string      `koanf:"id"`
		SharedKey        string      `koanf:"shared_key"`
		BootstrapCluster bool        `koanf:"bootstrap_cluster"`
		DiscoveryDomain  string      `koanf:"discovery_domain"`
		ClusterAddrs     []string    `koanf:"cluster_address"`
		AdvertAddr       string      `koanf:"advert_address"`
		AdvertScheme     string      `koanf:"advert_scheme"`
		RaftConfig       *RaftConfig `koanf:"raft_config"`
	}

	RaftConfig struct {
		HeartbeatTimeout   time.Duration `koanf:"heartbeat_timeout"`
		ElectionTimeout    time.Duration `koanf:"election_timeout"`
		CommitTimeout      time.Duration `koanf:"commit_timeout"`
		SnapshotInterval   time.Duration `koanf:"snapshot_interval"`
		SnapshotThreshold  int           `koanf:"snapshot_threshold"`
		MaxAppendEntries   int           `koanf:"max_append_entries"`
		TrailingLogs       int           `koanf:"trailing_logs"`
		LeaderLeaseTimeout time.Duration `koanf:"leader_lease_timeout"`
	}

	DGateDashboardConfig struct {
		Enable bool `koanf:"enable"`
	}

	AuthMethodType string

	DGateBasicAuthConfig struct {
		Users []DGateUserCredentials `koanf:"users"`
	}

	DGateUserCredentials struct {
		Username string `koanf:"username"`
		Password string `koanf:"password"`
	}

	DGateKeyAuthConfig struct {
		QueryParamName string   `koanf:"query_param_name"`
		HeaderName     string   `koanf:"header_name"`
		Keys           []string `koanf:"keys"`
	}

	DGateJWTAuthConfig struct {
		// HeaderName is the name of the header to extract the JWT token from
		HeaderName string         `koanf:"header_name"`
		Algorithm  string         `koanf:"algorithm"`
		SignatureConfig     map[string]any `koanf:",remain"`
	}

	AsymmetricSignatureConfig struct {
		// ES256, ES384, ES512
		// PS256, PS384, PS512
		// RS256, RS384, RS512
		Algorithm     string `koanf:"algorithm"`
		PublicKey     string `koanf:"public_key"`
		PublicKeyFile string `koanf:"public_key_file"`
	}

	SymmetricSignatureConfig struct {
		// HS256, HS384, HS512
		Algorithm string `koanf:"algorithm"`
		Key       string `koanf:"key"`
	}

	DGateTLSConfig struct {
		Port     int    `koanf:"port"`
		CertFile string `koanf:"cert_file"`
		KeyFile  string `koanf:"key_file"`
	}

	DGateHttpTransportConfig struct {
		DNSServer              string        `koanf:"dns_server"`
		DNSTimeout             time.Duration `koanf:"dns_timeout"`
		DNSPreferGo            bool          `koanf:"dns_prefer_go"`
		MaxIdleConns           int           `koanf:"max_idle_conns"`
		MaxIdleConnsPerHost    int           `koanf:"max_idle_conns_per_host"`
		MaxConnsPerHost        int           `koanf:"max_conns_per_host"`
		IdleConnTimeout        time.Duration `koanf:"idle_conn_timeout"`
		ForceAttemptHttp2      bool          `koanf:"force_attempt_http2"`
		DisableCompression     bool          `koanf:"disable_compression"`
		TLSHandshakeTimeout    time.Duration `koanf:"tls_handshake_timeout"`
		ExpectContinueTimeout  time.Duration `koanf:"expect_continue_timeout"`
		MaxResponseHeaderBytes int64         `koanf:"max_response_header_bytes"`
		WriteBufferSize        int           `koanf:"write_buffer_size"`
		ReadBufferSize         int           `koanf:"read_buffer_size"`
		MaxConnsPerClient      int           `koanf:"max_conns_per_client"`
		MaxBodyBytes           int           `koanf:"max_body_bytes"`
		DisableKeepAlives      bool          `koanf:"disable_keep_alives"`
		KeepAlive              time.Duration `koanf:"keep_alive"`
		ResponseHeaderTimeout  time.Duration `koanf:"response_header_timeout"`
		DialTimeout            time.Duration `koanf:"dial_timeout"`
	}

	DGateStorageConfig struct {
		StorageType StorageType    `koanf:"type"`
		Config      map[string]any `koanf:",remain"`
	}

	DGateFileConfig struct {
		Dir string `koanf:"dir"`
	}

	DGateResources struct {
		SkipValidation bool              `koanf:"skip_validation"`
		Namespaces     []spec.Namespace  `koanf:"namespaces"`
		Services       []spec.Service    `koanf:"services"`
		Routes         []spec.Route      `koanf:"routes"`
		Modules        []ModuleSpec      `koanf:"modules"`
		Domains        []DomainSpec      `koanf:"domains"`
		Collections    []spec.Collection `koanf:"-"`
		Documents      []spec.Document   `koanf:"documents"`
	}

	DomainSpec struct {
		spec.Domain `koanf:",squash"`
		CertFile    string `koanf:"cert_file"`
		KeyFile     string `koanf:"key_file"`
	}

	ModuleSpec struct {
		spec.Module `koanf:",squash"`
		PayloadFile string `koanf:"payload_file"`
	}
)

const (
	AuthMethodNone      AuthMethodType = "none"
	AuthMethodBasicAuth AuthMethodType = "basic"
	AuthMethodKeyAuth   AuthMethodType = "key"
	AuthMethodJWTAuth   AuthMethodType = "jwt"
)

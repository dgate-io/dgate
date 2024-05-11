package dgclient

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type DGateClient interface {
	Init(baseUrl string, opts ...Options) error
	BaseUrl() string

	DGateNamespaceClient
	DGateModuleClient
	DGateRouteClient
	DGateServiceClient
	DGateDomainClient
	DGateCollectionClient
	DGateDocumentClient
	DGateSecretClient
}

type dgateClient struct {
	client  *http.Client
	baseUrl *url.URL
}

type Options func(DGateClient)

func NewDGateClient() DGateClient {
	return &dgateClient{}
}

func (d *dgateClient) Init(baseUrl string, opts ...Options) error {
	if !strings.Contains(baseUrl, "://") {
		baseUrl = "http://" + baseUrl
	}
	bUrl, err := url.Parse(baseUrl)
	if err != nil {
		return err
	}

	if bUrl.Scheme == "" {
		bUrl.Scheme = "http"
	} else if bUrl.Host == "" {
		return url.InvalidHostError("host is empty")
	}

	d.client = http.DefaultClient
	d.baseUrl = bUrl

	for _, opt := range opts {
		opt(d)
	}
	return nil
}

func (d *dgateClient) BaseUrl() string {
	return d.baseUrl.String()
}

func WithHttpClient(client *http.Client) Options {
	return func(dc DGateClient) {
		if d, ok := dc.(*dgateClient); ok {
			d.client = client
		}
	}
}

type customTransport struct {
	UserAgent  string
	Username   string
	Password   string
	VerboseLog bool
	Transport  http.RoundTripper
}

func (ct *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	if ct.Username != "" || ct.Password != "" {
		req.SetBasicAuth(ct.Username, ct.Password)
	}
	if ct.UserAgent != "" {
		req.Header.Set("User-Agent", ct.UserAgent)
	}
	resp, err := ct.Transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if ct.VerboseLog {
		fmt.Printf("%s %s %s - %s %v\n",
			resp.Proto, req.Method, req.URL,
			resp.Status, time.Since(start),
		)
	}
	return resp, err
}

func WithBasicAuth(username, password string) Options {
	if username == "" || password == "" {
		return nil
	}
	return func(dc DGateClient) {
		if d, ok := dc.(*dgateClient); ok {
			if d.client.Transport == nil {
				d.client.Transport = http.DefaultTransport
			}
			if ct, ok := d.client.Transport.(*customTransport); ok {
				ct.Username = username
				ct.Password = password
			} else {
				d.client.Transport = &customTransport{
					Username:  username,
					Password:  password,
					Transport: d.client.Transport,
				}
			}
		}
	}
}

func WithFollowRedirect(follow bool) Options {
	return func(dc DGateClient) {
		if d, ok := dc.(*dgateClient); ok {
			if follow {
				return
			}
			d.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}
		}
	}
}

func WithUserAgent(ua string) Options {
	if ua == "" {
		return nil
	}
	return func(dc DGateClient) {
		if d, ok := dc.(*dgateClient); ok {
			if d.client.Transport == nil {
				d.client.Transport = http.DefaultTransport
			}
			if ct, ok := d.client.Transport.(*customTransport); ok {
				ct.UserAgent = ua
				ct.Transport = http.DefaultTransport
			} else {
				d.client.Transport = &customTransport{
					UserAgent: ua,
					Transport: d.client.Transport,
				}
			}
		}
	}
}

func WithVerboseLogging(on bool) Options {
	return func(dc DGateClient) {
		if d, ok := dc.(*dgateClient); ok {
			if d.client.Transport == nil {
				d.client.Transport = http.DefaultTransport
			}
			if ct, ok := d.client.Transport.(*customTransport); ok {
				ct.VerboseLog = on
			} else {
				d.client.Transport = &customTransport{
					VerboseLog: on,
					Transport:  d.client.Transport,
				}
			}
		}
	}
}

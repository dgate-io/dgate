package dgclient

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type DGateClient struct {
	client  *http.Client
	baseUrl *url.URL
}

type Options func(*DGateClient)

func NewDGateClient(baseUrl string, opts ...Options) (*DGateClient, error) {
	dgc := &DGateClient{}
	err := dgc.Init(baseUrl)
	if err != nil {
		return nil, err
	}
	return dgc, nil
}

func (d *DGateClient) Init(baseUrl string, opts ...Options) error {
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

func (d *DGateClient) BaseUrl() string {
	return d.baseUrl.String()
}

func WithHttpClient(client *http.Client) Options {
	return func(d *DGateClient) {
		d.client = client
	}
}

type customTransport struct {
	UserAgent string
	Username  string
	Password  string
	Transport http.RoundTripper
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
	fmt.Printf("%s %s - %s %v\n", req.Method,
		req.URL.String(), resp.Status, time.Since(start))
	return resp, err
}

func WithBasicAuth(username, password string) Options {
	if username == "" || password == "" {
		return func(d *DGateClient) {}
	}
	return func(d *DGateClient) {
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

func WithUserAgent(ua string) Options {
	if ua == "" {
		return func(dc *DGateClient) {}
	}
	return func(d *DGateClient) {
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

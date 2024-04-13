package proxy

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"github.com/dgate-io/dgate/internal/config"
	"golang.org/x/net/http2"
)

func setupTranportFromConfig(
	c config.DGateHttpTransportConfig,
	modifyTransport func(*net.Dialer, *http.Transport),
) http.RoundTripper {
	t1 := http.DefaultTransport.(*http.Transport).Clone()
	dailer := &net.Dialer{
		Timeout:   c.DialTimeout,
		KeepAlive: c.KeepAlive,
		Resolver: &net.Resolver{
			PreferGo: c.DNSPreferGo,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: c.DNSTimeout,
				}
				return d.DialContext(ctx, network, c.DNSServer)
			},
		},
	}
	if t1.DisableKeepAlives {
		dailer.KeepAlive = -1
	}
	t1.DialContext = dailer.DialContext
	t1.MaxIdleConns = c.MaxIdleConns
	t1.IdleConnTimeout = c.IdleConnTimeout
	t1.TLSHandshakeTimeout = c.TLSHandshakeTimeout
	t1.ExpectContinueTimeout = c.ExpectContinueTimeout
	t1.MaxIdleConnsPerHost = c.MaxIdleConnsPerHost
	t1.MaxConnsPerHost = c.MaxConnsPerHost
	t1.MaxResponseHeaderBytes = c.MaxResponseHeaderBytes
	t1.WriteBufferSize = c.WriteBufferSize
	t1.ReadBufferSize = c.ReadBufferSize
	t1.DisableKeepAlives = c.DisableKeepAlives
	t1.DisableCompression = c.DisableCompression
	t1.ForceAttemptHTTP2 = c.ForceAttemptHttp2
	t1.ResponseHeaderTimeout = c.ResponseHeaderTimeout
	if modifyTransport != nil {
		modifyTransport(dailer, t1)
	}

	return newRoundTripper(t1)
}

func newRoundTripper(transport *http.Transport) http.RoundTripper {
	transportH2C := &h2cTransport{
		transport: &http2.Transport{
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
			AllowHTTP: true,
		},
	}

	return &dynamicRoundTripper{
		hx:  transport,
		h2c: transportH2C,
	}
}

// dynamicRoundTripper implements RoundTrip while making sure that HTTP/2 is not used
// with protocols that start with a Connection Upgrade, such as SPDY or Websocket.
type dynamicRoundTripper struct {
	hx  *http.Transport
	h2c *h2cTransport
}

func (m *dynamicRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.ProtoAtLeast(2, 0) && (req.URL.Scheme == "h2c") {
		return m.h2c.RoundTrip(req)
	}
	return m.hx.RoundTrip(req)
}

type h2cTransport struct {
	transport *http2.Transport
}

func (t *h2cTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	return t.transport.RoundTrip(req)
}

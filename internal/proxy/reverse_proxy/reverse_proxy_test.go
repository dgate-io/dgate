package reverse_proxy_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/dgate-io/dgate/internal/proxy/proxytest"
	"github.com/dgate-io/dgate/internal/proxy/reverse_proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type ProxyParams struct {
	host    string
	newHost string

	upstreamUrl   *url.URL
	newUpsteamURL *url.URL

	proxyPattern string
	proxyPath    string
}

type RewriteParams struct {
	stripPath          bool
	preserveHost       bool
	disableQueryParams bool
	xForwardedHeaders  bool
}

func testDGateProxyRewrite(
	t *testing.T, params ProxyParams,
	rewriteParams RewriteParams,
) {
	mockTp := proxytest.CreateMockTransport()
	header := make(http.Header)
	header.Add("X-Testing", "testing")
	rp, err := reverse_proxy.NewBuilder().
		Transport(mockTp).
		ProxyRewrite(
			rewriteParams.stripPath,
			rewriteParams.preserveHost,
			rewriteParams.disableQueryParams,
			rewriteParams.xForwardedHeaders,
		).Build(params.upstreamUrl, params.proxyPattern)
	if err != nil {
		t.Fatal(err)
	}

	req := proxytest.CreateMockRequest("GET", params.proxyPath)
	req.RemoteAddr = "::1"
	req.Host = params.host
	req.URL.Scheme = ""
	req.URL.Host = ""
	mockRw := proxytest.CreateMockResponseWriter()
	mockRw.On("Header").Return(header)
	mockRw.On("WriteHeader", mock.Anything).Return()
	mockRw.On("Write", mock.Anything).Return(0, nil)
	req = req.WithContext(context.WithValue(
		context.Background(), proxytest.S("testing"), "testing"))

	mockTp.On("RoundTrip", mock.Anything).Run(func(args mock.Arguments) {
		req := args.Get(0).(*http.Request)
		if req.URL.String() != params.newUpsteamURL.String() {
			t.Errorf("FAIL: Expected URL %s, got %s", params.newUpsteamURL, req.URL)
		} else {
			t.Logf("PASS: upstreamUrl: %s, proxyPattern: %s, proxyPath: %s, newUpsteamURL: %s",
				params.upstreamUrl, params.proxyPattern, params.proxyPath, params.newUpsteamURL)
		}
		if params.newHost != "" && req.Host != params.newHost {
			t.Errorf("FAIL: Expected Host %s, got %s", params.newHost, req.Host)
		}
		if rewriteParams.xForwardedHeaders {
			if req.Header.Get("X-Forwarded-For") == "" {
				t.Errorf("FAIL: Expected X-Testing header, got %s", req.Header.Get("X-Testing"))
			}
			if req.Header.Get("X-Real-IP") == "" {
				t.Errorf("FAIL: Expected X-Testing header, got %s", req.Header.Get("X-Testing"))
			}
			if req.Header.Get("X-Forwarded-Host") == "" {
				t.Errorf("FAIL: Expected X-Testing header, got %s", req.Header.Get("X-Testing"))
			}
			if req.Header.Get("X-Forwarded-Proto") == "" {
				t.Errorf("FAIL: Expected X-Testing header, got %s", req.Header.Get("X-Testing"))
			}
		} else {
			if req.Header.Get("X-Forwarded-For") != "" {
				t.Errorf("FAIL: Expected no X-Testing header, got %s", req.Header.Get("X-Testing"))
			}
			if req.Header.Get("X-Real-IP") != "" {
				t.Errorf("FAIL: Expected X-Testing header, got %s", req.Header.Get("X-Testing"))
			}
			if req.Header.Get("X-Forwarded-Host") != "" {
				t.Errorf("FAIL: Expected X-Testing header, got %s", req.Header.Get("X-Testing"))
			}
			if req.Header.Get("X-Forwarded-Proto") != "" {
				t.Errorf("FAIL: Expected X-Testing header, got %s", req.Header.Get("X-Testing"))
			}
		}
	}).Return(&http.Response{
		StatusCode:    200,
		ContentLength: 0,
		Header:        header,
		Body:          io.NopCloser(strings.NewReader("")),
	}, nil).Once()

	rp.ServeHTTP(mockRw, req)

	// ensure roundtrip is called at least once
	mockTp.AssertExpectations(t)
	// ensure retries are called
	// ensure context is passed through
	assert.Equal(t, "testing", req.Context().Value(proxytest.S("testing")))
}

func mustParseURL(t *testing.T, s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestDGateProxyError(t *testing.T) {
	mockTp := proxytest.CreateMockTransport()
	header := make(http.Header)
	header.Add("X-Testing", "testing")
	mockTp.On("RoundTrip", mock.Anything).
		Return(nil, errors.New("testing error")).
		Times(4)
	mockTp.On("RoundTrip", mock.Anything).Return(&http.Response{
		StatusCode:    200,
		ContentLength: 0,
		Header:        header,
		Body:          io.NopCloser(strings.NewReader("")),
	}, nil).Once()

	upstreamUrl, _ := url.Parse("http://example.com")
	rp, err := reverse_proxy.NewBuilder().
		Clone().FlushInterval(-1).
		Transport(mockTp).
		ProxyRewrite(
			true, true,
			true, true,
		).Build(upstreamUrl, "/test/*")
	if err != nil {
		t.Fatal(err)
	}

	req := proxytest.CreateMockRequest("GET", "http://localhost")
	req.RemoteAddr = "::1"
	mockRw := proxytest.CreateMockResponseWriter()
	mockRw.On("Header").Return(header)
	mockRw.On("WriteHeader", mock.Anything).Return()
	mockRw.On("Write", mock.Anything).Return(0, nil)
	req = req.WithContext(context.WithValue(
		context.Background(), proxytest.S("testing"), "testing"))
	rp.ServeHTTP(mockRw, req)

	// ensure roundtrip is called at least once
	mockTp.AssertCalled(t, "RoundTrip", mock.Anything)
	// ensure retries are called
	// ensure context is passed through
	assert.Equal(t, "testing", req.Context().Value(proxytest.S("testing")))
}

func TestDGateProxyRewriteStripPath(t *testing.T) {
	// if proxy pattern is a prefix (ends with *)
	testDGateProxyRewrite(t, ProxyParams{
		host:    "test.net",
		newHost: "example.com",

		upstreamUrl:   mustParseURL(t, "http://example.com"),
		newUpsteamURL: mustParseURL(t, "http://example.com/test/ing"),

		proxyPattern: "/test/*",
		proxyPath:    "/test/test/ing",
	}, RewriteParams{
		stripPath:          true,
		preserveHost:       false,
		disableQueryParams: false,
		xForwardedHeaders:  false,
	})

	testDGateProxyRewrite(t, ProxyParams{
		host:    "test.net",
		newHost: "example.com",

		upstreamUrl:   mustParseURL(t, "http://example.com/pre"),
		newUpsteamURL: mustParseURL(t, "http://example.com/pre"),

		proxyPattern: "/test/*",
		proxyPath:    "/test/",
	}, RewriteParams{
		stripPath:          true,
		preserveHost:       false,
		disableQueryParams: false,
		xForwardedHeaders:  false,
	})
}

func TestDGateProxyRewritePreserveHost(t *testing.T) {
	testDGateProxyRewrite(t, ProxyParams{
		upstreamUrl:   mustParseURL(t, "http://example.com"),
		newUpsteamURL: mustParseURL(t, "http://example.com/test"),

		host:    "test.net",
		newHost: "test.net",

		proxyPattern: "/test",
		proxyPath:    "/test",
	}, RewriteParams{
		stripPath:          false,
		preserveHost:       true,
		disableQueryParams: false,
		xForwardedHeaders:  false,
	})
}

func TestDGateProxyRewriteDisableQueryParams(t *testing.T) {
	testDGateProxyRewrite(t, ProxyParams{
		upstreamUrl:   mustParseURL(t, "http://example.com"),
		newUpsteamURL: mustParseURL(t, "http://example.com/test"),

		host:    "test.net",
		newHost: "example.com",

		proxyPattern: "/test",
		proxyPath:    "/test?testing=testing",
	}, RewriteParams{
		stripPath:          false,
		preserveHost:       false,
		disableQueryParams: true,
		xForwardedHeaders:  false,
	})

	testDGateProxyRewrite(t, ProxyParams{
		upstreamUrl:   mustParseURL(t, "http://example.com"),
		newUpsteamURL: mustParseURL(t, "http://example.com/test?testing=testing"),

		host:    "test.net",
		newHost: "example.com",

		proxyPattern: "/test",
		proxyPath:    "/test?testing=testing",
	}, RewriteParams{
		stripPath:          false,
		preserveHost:       false,
		disableQueryParams: false,
		xForwardedHeaders:  false,
	})
}

func TestDGateProxyRewriteXForwardedHeaders(t *testing.T) {
	testDGateProxyRewrite(t, ProxyParams{
		upstreamUrl:   mustParseURL(t, "http://example.com"),
		newUpsteamURL: mustParseURL(t, "http://example.com/test"),

		host:    "test.net",
		newHost: "example.com",

		proxyPattern: "/test",
		proxyPath:    "/test",
	}, RewriteParams{
		stripPath:          false,
		preserveHost:       false,
		disableQueryParams: false,
		xForwardedHeaders:  true,
	})

	testDGateProxyRewrite(t, ProxyParams{
		upstreamUrl:   mustParseURL(t, "http://example.com"),
		newUpsteamURL: mustParseURL(t, "http://example.com/test"),

		host:    "test.net",
		newHost: "example.com",

		proxyPattern: "/test",
		proxyPath:    "/test",
	}, RewriteParams{
		stripPath:          false,
		preserveHost:       false,
		disableQueryParams: false,
		xForwardedHeaders:  false,
	})
}

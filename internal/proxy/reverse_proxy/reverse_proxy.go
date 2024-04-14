package reverse_proxy

import (
	"errors"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
	"time"
)

type RewriteFunc func(*http.Request, *http.Request)

type ModifyResponseFunc func(*http.Response) error

type ErrorHandlerFunc func(http.ResponseWriter, *http.Request, error)

type Builder interface {
	// FlushInterval sets the flush interval for flushable response bodies.
	FlushInterval(time.Duration) Builder

	// Rewrite sets the rewrite function for the reverse proxy.
	CustomRewrite(RewriteFunc) Builder

	// ModifyResponse sets the modify response function for the reverse proxy.
	ModifyResponse(ModifyResponseFunc) Builder

	// ErrorHandler sets the error handler function for the reverse proxy.
	ErrorHandler(ErrorHandlerFunc) Builder

	// Transport sets the transport for the reverse proxy.
	Transport(http.RoundTripper) Builder

	// ErrorLogger sets the (go) logger for the reverse proxy.
	ErrorLogger(*log.Logger) Builder

	// ProxyRewrite sets the proxy rewrite function for the reverse proxy.
	ProxyRewrite(
		stripPath bool,
		preserveHost bool,
		disableQueryParams bool,
		xForwardedHeaders bool,
	) Builder

	// Build builds the reverse proxy executor.
	Build(
		upstreamUrl *url.URL,
		proxyPattern string,
	) (http.Handler, error)

	// Clone clones the builder.
	Clone() Builder
}

var _ Builder = (*reverseProxyBuilder)(nil)

type reverseProxyBuilder struct {
	rewrite        RewriteFunc
	errorLogger    *log.Logger
	customRewrite  RewriteFunc
	upstreamUrl    *url.URL
	proxyPattern   string
	transport      http.RoundTripper
	flushInterval  time.Duration
	modifyResponse ModifyResponseFunc
	errorHandler   ErrorHandlerFunc
}

func NewBuilder() Builder {
	return &reverseProxyBuilder{}
}

func (b *reverseProxyBuilder) Clone() Builder {
	return &reverseProxyBuilder{
		rewrite:        b.rewrite,
		errorLogger:    b.errorLogger,
		customRewrite:  b.customRewrite,
		upstreamUrl:    b.upstreamUrl,
		proxyPattern:   b.proxyPattern,
		transport:      b.transport,
		flushInterval:  b.flushInterval,
		modifyResponse: b.modifyResponse,
		errorHandler:   b.errorHandler,
	}
}

func (b *reverseProxyBuilder) FlushInterval(interval time.Duration) Builder {
	b.flushInterval = interval
	return b
}

func (b *reverseProxyBuilder) CustomRewrite(rewrite RewriteFunc) Builder {
	b.customRewrite = rewrite
	return b
}

func (b *reverseProxyBuilder) ModifyResponse(modifyResponse ModifyResponseFunc) Builder {
	b.modifyResponse = modifyResponse
	return b
}

func (b *reverseProxyBuilder) ErrorHandler(errorHandler ErrorHandlerFunc) Builder {
	b.errorHandler = errorHandler
	return b
}

func (b *reverseProxyBuilder) ErrorLogger(logger *log.Logger) Builder {
	b.errorLogger = logger
	return b
}

func (b *reverseProxyBuilder) Transport(transport http.RoundTripper) Builder {
	b.transport = transport
	return b
}

func (b *reverseProxyBuilder) ProxyRewrite(
	stripPath bool,
	preserveHost bool,
	disableQueryParams bool,
	xForwardedHeaders bool,
) Builder {
	b.rewrite = func(in, out *http.Request) {
		in.URL.Scheme = b.upstreamUrl.Scheme
		in.URL.Host = b.upstreamUrl.Host

		b.stripPath(stripPath)(in, out)
		b.preserveHost(preserveHost)(in, out)
		b.disableQueryParams(disableQueryParams)(in, out)
		b.xForwardedHeaders(xForwardedHeaders)(in, out)
	}
	return b
}

func (b *reverseProxyBuilder) stripPath(strip bool) RewriteFunc {
	return func(in, out *http.Request) {
		reqCall := in.URL.Path
		proxyPatternPath := b.proxyPattern
		upstreamPath := b.upstreamUrl.Path
		in.URL = b.upstreamUrl
		if strip {
			if strings.HasSuffix(proxyPatternPath, "*") {
				// this will remove the proxy path before the wildcard from the upstream url
				// ex. (upstreamPath: /v1, proxyPattern: '/path/*', reqCall: '/path/test') -> '/v1/test'
				proxyPattern := strings.TrimSuffix(proxyPatternPath, "*")
				reqCallNoProxy := strings.TrimPrefix(reqCall, proxyPattern)
				out.URL.Path = path.Join(upstreamPath, reqCallNoProxy)
			} else {
				// this will remove the proxy path from the upstream url
				// ex. (upstreamPath: /v1, proxyPattern: '/path/{id}', reqCall: '/path/1') -> '/v1'
				out.URL.Path = upstreamPath
			}
		} else {
			// ex. (upstreamPath: /v1, proxyPattern: '/path/*', reqCall: '/path/test') -> '/v1/path/test'
			out.URL.Path = path.Join(upstreamPath, reqCall)
		}
	}
}

func (b *reverseProxyBuilder) preserveHost(preserve bool) RewriteFunc {
	return func(in, out *http.Request) {
		scheme := "http"
		out.URL.Host = b.upstreamUrl.Host
		if preserve {
			out.Host = in.Host
			if out.Host == "" {
				out.Host = out.URL.Host
			}
			if in.TLS != nil {
				scheme = "https"
			}
		} else {
			out.Host = out.URL.Host
			scheme = b.upstreamUrl.Scheme
		}
		if out.URL.Scheme == "" {
			out.URL.Scheme = scheme
		}
	}
}

func (b *reverseProxyBuilder) disableQueryParams(disableQueryParams bool) RewriteFunc {
	return func(in, out *http.Request) {
		if !disableQueryParams {
			targetQuery := b.upstreamUrl.RawQuery
			if targetQuery == "" || in.URL.RawQuery == "" {
				in.URL.RawQuery = targetQuery + in.URL.RawQuery
			} else {
				in.URL.RawQuery = targetQuery + "&" + in.URL.RawQuery
			}
		} else {
			out.URL.RawQuery = ""
		}
	}
}

func (b *reverseProxyBuilder) xForwardedHeaders(xForwardedHeaders bool) RewriteFunc {
	return func(in, out *http.Request) {
		if xForwardedHeaders {
			clientIP, _, err := net.SplitHostPort(in.RemoteAddr)
			if err == nil {
				out.Header.Add("X-Forwarded-For", clientIP)
				out.Header.Set("X-Real-IP", clientIP)
			} else {
				out.Header.Add("X-Forwarded-For", in.RemoteAddr)
				out.Header.Set("X-Real-IP", in.RemoteAddr)
			}
			out.Header.Set("X-Forwarded-Host", in.Host)
			if in.TLS == nil {
				out.Header.Set("X-Forwarded-Proto", "http")
			} else {
				out.Header.Set("X-Forwarded-Proto", "https")
			}
		} else {
			out.Header.Del("X-Forwarded-For")
			out.Header.Del("X-Forwarded-Host")
			out.Header.Del("X-Forwarded-Proto")
			out.Header.Del("X-Real-IP")
		}
	}
}

var (
	ErrNilUpstreamUrl    = errors.New("upstream url cannot be nil")
	ErrEmptyProxyPattern = errors.New("proxy pattern cannot be empty")
)

func (b *reverseProxyBuilder) Build(
	upstreamUrl *url.URL,
	proxyPattern string,
) (http.Handler, error) {
	if upstreamUrl == nil {
		return nil, ErrNilUpstreamUrl
	}
	b.upstreamUrl = upstreamUrl

	if proxyPattern == "" {
		return nil, ErrEmptyProxyPattern
	}
	b.proxyPattern = proxyPattern

	if b.transport == nil {
		b.transport = http.DefaultTransport
	}

	if b.flushInterval == 0 {
		b.flushInterval = time.Millisecond * 100
	}

	if b.errorHandler == nil {
		b.errorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
			http.Error(rw, err.Error(), http.StatusBadGateway)
		}
	}
	proxy := &httputil.ReverseProxy{}
	proxy.ErrorHandler = b.errorHandler
	proxy.FlushInterval = b.flushInterval
	proxy.ModifyResponse = b.modifyResponse
	proxy.Transport = b.transport
	proxy.ErrorLog = b.errorLogger
	proxy.Rewrite = func(pr *httputil.ProxyRequest) {
		if b.customRewrite != nil {
			b.customRewrite(pr.In, pr.Out)
		}
		if b.rewrite != nil {
			b.rewrite(pr.In, pr.Out)
		}
		if pr.Out.URL.Path == "/" {
			pr.Out.URL.Path = ""
		}
	}
	return proxy, nil
}

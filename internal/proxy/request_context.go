package proxy

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"github.com/dgate-io/dgate/internal/proxy/reverse_proxy"
	"github.com/dgate-io/dgate/pkg/spec"
)

type S string

type RequestContextProvider struct {
	ctx    context.Context
	route  *spec.DGateRoute
	rpb    reverse_proxy.Builder
	modBuf ModuleBuffer
}

type RequestContext struct {
	pattern  string
	ctx      context.Context
	route    *spec.DGateRoute
	rw       spec.ResponseWriterTracker
	req      *http.Request
	provider *RequestContextProvider
}

func NewRequestContextProvider(
	route *spec.DGateRoute,
	ps *ProxyState,
) *RequestContextProvider {
	ctx := context.Background()

	ctx = context.WithValue(ctx, spec.Name("route"), route.Name)
	ctx = context.WithValue(ctx, spec.Name("namespace"), route.Namespace.Name)

	var rpb reverse_proxy.Builder
	if route.Service != nil {
		ctx = context.WithValue(ctx, spec.Name("service"), route.Service.Name)
		transport := setupTranportsFromConfig(
			ps.config.ProxyConfig.Transport,
			func(dialer *net.Dialer, t *http.Transport) {
				t.TLSClientConfig = &tls.Config{
					InsecureSkipVerify: route.Service.TLSSkipVerify,
				}
				dialer.Timeout = route.Service.ConnectTimeout
				t.ForceAttemptHTTP2 = route.Service.HTTP2Only
			},
		)
		proxy, err := ps.ProxyTransportBuilder.Clone().
			Transport(transport).
			Retries(route.Service.Retries).
			RetryTimeout(route.Service.RetryTimeout).
			RequestTimeout(route.Service.RequestTimeout).
			Build()
		if err != nil {
			panic(err)
		}
		rpb = ps.ReverseProxyBuilder.Clone().
			Transport(proxy).
			ProxyRewrite(
				route.StripPath,
				route.PreserveHost,
				route.Service.DisableQueryParams,
				ps.config.ProxyConfig.DisableXForwardedHeaders,
			)

	}

	return &RequestContextProvider{
		ctx:   ctx,
		route: route,
		rpb:   rpb,
	}
}

func (reqCtxProvider *RequestContextProvider) SetModuleBuffer(mb ModuleBuffer) {
	reqCtxProvider.modBuf = mb
}

func (reqCtxProvider *RequestContextProvider) CreateRequestContext(
	ctx context.Context,
	rw http.ResponseWriter,
	req *http.Request,
	pattern string,
) *RequestContext {
	return &RequestContext{
		rw:       spec.NewResponseWriterTracker(rw),
		req:      req.WithContext(ctx),
		route:    reqCtxProvider.route,
		provider: reqCtxProvider,
		pattern:  pattern,
		ctx:      ctx,
	}
}

func (reqCtx *RequestContext) Context() context.Context {
	return reqCtx.ctx
}

func (reqCtx *RequestContext) Route() *spec.DGateRoute {
	return reqCtx.route
}

func (reqCtx *RequestContext) Pattern() string {
	return reqCtx.pattern
}

func (reqCtx *RequestContext) Request() *http.Request {
	return reqCtx.req
}

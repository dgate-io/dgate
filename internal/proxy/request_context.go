package proxy

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"

	"github.com/dgate-io/chi-router"
	"github.com/dgate-io/dgate/internal/proxy/reverse_proxy"
	"github.com/dgate-io/dgate/pkg/spec"
)

type S string

type RequestContextProvider struct {
	ctx    context.Context
	route  *spec.DGateRoute
	rpb    reverse_proxy.Builder
	mtx    *sync.Mutex
	modBuf ModulePool
}

type RequestContext struct {
	pattern  string
	ctx      context.Context
	route    *spec.DGateRoute
	rw       spec.ResponseWriterTracker
	req      *http.Request
	provider *RequestContextProvider
	params   map[string]string
}

func NewRequestContextProvider(route *spec.DGateRoute, ps *ProxyState) *RequestContextProvider {
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
		mtx:   &sync.Mutex{},
	}
}

func (reqCtxProvider *RequestContextProvider) SetModulePool(mb ModulePool) {
	reqCtxProvider.mtx.Lock()
	defer reqCtxProvider.mtx.Unlock()
	if reqCtxProvider.modBuf != nil {
		reqCtxProvider.modBuf.Close()
	}
	reqCtxProvider.modBuf = mb
}

func (reqCtxProvider *RequestContextProvider) ModulePool() ModulePool {
	reqCtxProvider.mtx.Lock()
	defer reqCtxProvider.mtx.Unlock()
	return reqCtxProvider.modBuf
}

func (reqCtxProvider *RequestContextProvider) CreateRequestContext(
	ctx context.Context, rw http.ResponseWriter,
	req *http.Request, pattern string,
) *RequestContext {
	pathParams := make(map[string]string)
	if chiCtx := chi.RouteContext(req.Context()); chiCtx != nil {
		for i, key := range chiCtx.URLParams.Keys {
			pathParams[key] = chiCtx.URLParams.Values[i]
		}
	}
	return &RequestContext{
		ctx:      ctx,
		pattern:  pattern,
		params:   pathParams,
		provider: reqCtxProvider,
		route:    reqCtxProvider.route,
		req:      req.WithContext(ctx),
		rw:       spec.NewResponseWriterTracker(rw),
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

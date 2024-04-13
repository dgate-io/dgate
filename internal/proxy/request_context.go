package proxy

import (
	"context"
	"net/http"

	"github.com/dgate-io/dgate/pkg/spec"
)

type S string

type RequestContextProvider struct {
	ctx    context.Context
	route  *spec.DGateRoute
	modBuf ModuleBuffer
}

type RequestContext struct {
	pattern  string
	context  context.Context
	route    *spec.DGateRoute
	rw       spec.ResponseWriterTracker
	req      *http.Request
	provider *RequestContextProvider
}

func NewRequestContextProvider(route *spec.DGateRoute) *RequestContextProvider {
	ctx := context.Background()

	// set context values
	ctx = context.WithValue(ctx, spec.Name("route"), route.Name)
	ctx = context.WithValue(ctx, spec.Name("namespace"), route.Namespace.Name)
	serviceName := ""
	if route.Service != nil {
		serviceName = route.Service.Name
	}
	ctx = context.WithValue(ctx, spec.Name("service"), serviceName)

	return &RequestContextProvider{
		ctx:   ctx,
		route: route,
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
		context:  ctx,
	}
}

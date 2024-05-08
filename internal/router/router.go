package router

import (
	"net/http"
	"sync"

	"github.com/dgate-io/chi-router"
)

// Router is a wrapper around chi.Router
type DynamicRouter struct {
	router   *chi.Mux
	routeCtx *chi.Context
	lock     sync.RWMutex
}

// NewRouter creates a new router
func NewRouterWithMux(mux *chi.Mux) *DynamicRouter {
	return &DynamicRouter{
		mux,
		chi.NewRouteContext(),
		sync.RWMutex{},
	}
}

func NewMux() *chi.Mux {
	chi.NewRouter()
	return chi.NewRouter()
}

func (r *DynamicRouter) ModifyMux(fn func(*chi.Mux)) {
	r.lock.Lock()
	defer r.lock.Unlock()
	fn(r.router)
}

// ReplaceRouter replaces the router
func (r *DynamicRouter) ReplaceMux(router *chi.Mux) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.routeCtx = chi.NewRouteContext()
	r.router = router
}

// ServeHTTP is a wrapper around chi.Router.ServeHTTP
func (r *DynamicRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// r.lock.RLock()
	// defer r.lock.RUnlock()
	r.router.ServeHTTP(w, req)
}

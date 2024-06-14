package router

import (
	"net/http"
	"sync"

	"github.com/dgate-io/chi-router"
)

// Router is a wrapper around chi.Router
type DynamicRouter struct {
	router *chi.Mux
	lock   *sync.RWMutex
}

// NewRouter creates a new router
func NewRouterWithMux(mux *chi.Mux) *DynamicRouter {
	return &DynamicRouter{mux, &sync.RWMutex{}}
}

func NewMux() *chi.Mux {
	return chi.NewRouter()
}

// ReplaceRouter replaces the router
func (r *DynamicRouter) ReplaceMux(router *chi.Mux) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.router = router
}

// ServeHTTP is a wrapper around chi.Router.ServeHTTP
func (r *DynamicRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	r.router.ServeHTTP(w, req)
}

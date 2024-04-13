package proxy

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/dgate-io/dgate/pkg/eventloop"
	"github.com/dgate-io/dgate/pkg/modules"
	"github.com/dgate-io/dgate/pkg/resources"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/dgate-io/dgate/pkg/typescript"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

// RuntimeContext is the context for the runtime. one per request
type runtimeContext struct {
	reqCtx *RequestContext

	loop    *eventloop.EventLoop
	state   modules.StateManager
	rm      *resources.ResourceManager
	route   *spec.Route
	modules []*spec.Module
}

func NewRuntimeContext(
	proxyState *ProxyState,
	route *spec.DGateRoute,
	modules ...*spec.DGateModule,
) *runtimeContext {
	rtCtx := &runtimeContext{
		state:   proxyState,
		rm:      proxyState.ResourceManager(),
		modules: spec.TransformDGateModules(modules...),
		route:   spec.TransformDGateRoute(route),
		// cache:   proxyState.sharedCache,
	}
	sort.Slice(rtCtx.modules, func(i, j int) bool {
		return rtCtx.modules[i].Name < rtCtx.modules[j].Name
	})
	// func (r *Registry) getCompiledSource(p string) wraps the code in a modular function
	reg := require.NewRegistryWithLoader(func(path string) ([]byte, error) {
		requireMod := strings.Replace(path, "node_modules/", "", 1)
		// 'https://' - requires network permissions and must be enabled in the config
		// 'file://' - requires file system permissions and must be enabled in the config
		// 'module://' - requires a module lookup and module permissions
		if mod, ok := findInSortedWith(rtCtx.modules, requireMod,
			func(m *spec.Module) string { return m.Name }); !ok {
			return nil, errors.New(requireMod + " not found")
		} else {
			if mod.Type == spec.ModuleTypeJavascript {
				return []byte(mod.Payload), nil
			}
			// TODO: add transpilation cache somewhere
			payload, err := typescript.Transpile(mod.Payload)
			if err != nil {
				return nil, err
			}
			return []byte(payload), nil
		}
	})
	rtCtx.loop = eventloop.NewEventLoop(eventloop.WithRegistry(reg))
	return rtCtx
}

var _ modules.RuntimeContext = &runtimeContext{}

// UseRequestContext sets the request context
func (rtCtx *runtimeContext) SetRequestContext(
	reqCtx *RequestContext, pathParams map[string]string,
) {
	if reqCtx != nil {
		if err := reqCtx.context.Err(); err != nil {
			panic("context is already closed: " + err.Error())
		}
	}
	rtCtx.reqCtx = reqCtx
}

func (rtCtx *runtimeContext) Clean() {
	rtCtx.reqCtx = nil
}

func (rtCtx *runtimeContext) Context() context.Context {
	if rtCtx.reqCtx == nil {
		panic("request context is not set")
	}
	return rtCtx.reqCtx.context
}

func (rtCtx *runtimeContext) EventLoop() *eventloop.EventLoop {
	return rtCtx.loop
}

func (rtCtx *runtimeContext) Runtime() *goja.Runtime {
	return rtCtx.loop.Runtime()
}

func (rtCtx *runtimeContext) State() modules.StateManager {
	return rtCtx.state
}

package proxy

import (
	"context"

	"github.com/dgate-io/dgate/pkg/eventloop"
	"github.com/dgate-io/dgate/pkg/modules"
	"github.com/dgate-io/dgate/pkg/resources"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/dop251/goja"
)

// RuntimeContext is the context for the runtime. one per request
type runtimeContext struct {
	reqCtx  *RequestContext
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
	}

	// TODO: setup module import logic
	// sort.Slice(rtCtx.modules, func(i, j int) bool {
	// 	return rtCtx.modules[i].Name < rtCtx.modules[j].Name
	// })
	// reg := require.NewRegistryWithLoader(func(path string) ([]byte, error) {
	// 	requireMod := strings.Replace(path, "node_modules/", "", 1)
	// 	// 'https://' - requires network permissions and must be enabled in the config
	// 	// 'file://' - requires file system permissions and must be enabled in the config
	// 	// 'module://' - requires a module lookup and module permissions
	// 	if mod, ok := findInSortedWith(rtCtx.modules, requireMod,
	// 		func(m *spec.Module) string { return m.Name }); !ok {
	// 		return nil, errors.New(requireMod + " not found")
	// 	} else {
	// 		if mod.Type == spec.ModuleTypeJavascript {
	// 			return []byte(mod.Payload), nil
	// 		}
	// 		var err error
	// 		var key string
	// 		transpileBucket := proxyState.sharedCache.Bucket("ts-transpile")
	// 		if key, err = HashString(0, mod.Payload); err == nil {
	// 			if code, ok := transpileBucket.Get(key); ok {
	// 				return code.([]byte), nil
	// 			}
	// 		}
	// 		payload, err := typescript.Transpile(mod.Payload)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		transpileBucket.SetWithTTL(key, []byte(payload), time.Minute*30)
	// 		return []byte(payload), nil
	// 	}
	// })
	rtCtx.loop = eventloop.NewEventLoop(
	// eventloop.WithRegistry(reg),
	)
	return rtCtx
}

var _ modules.RuntimeContext = &runtimeContext{}

// UseRequestContext sets the request context
func (rtCtx *runtimeContext) Use(reqCtx *RequestContext) (*runtimeContext, error) {
	if reqCtx != nil {
		if err := reqCtx.ctx.Err(); err != nil {
			return nil, err
		}
	}
	rtCtx.reqCtx = reqCtx
	return rtCtx, nil
}

func (rtCtx *runtimeContext) Clean() {
	rtCtx.loop.StopNoWait()
	rtCtx.loop = nil
	rtCtx.reqCtx = nil
	rtCtx.route = nil
	rtCtx.modules = nil
}

func (rtCtx *runtimeContext) Context() context.Context {
	return rtCtx.reqCtx.ctx
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

package extractors

import (
	"errors"
	"reflect"

	"github.com/dgate-io/dgate/pkg/eventloop"
	"github.com/dgate-io/dgate/pkg/modules"
	"github.com/dgate-io/dgate/pkg/modules/dgate"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/buffer"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	"github.com/dop251/goja_nodejs/url"
	"github.com/stoewer/go-strcase"
)

var _ console.Printer = &NoopPrinter{}

type NoopPrinter struct{}

func (p *NoopPrinter) Log(string)   {}
func (p *NoopPrinter) Warn(string)  {}
func (p *NoopPrinter) Error(string) {}

type RuntimeOptions struct {
	Env map[string]string
}

func prepareRuntime(rt *goja.Runtime) {
	rt.SetFieldNameMapper(&smartFieldNameMapper{})

	exports := rt.NewObject()
	module := rt.NewObject()
	module.Set("exports", exports)
	rt.Set("exports", exports)
	rt.GlobalObject().Set("process", processObject(rt))
	rt.Set("module", module)
}

func processObject(rt *goja.Runtime) *goja.Object {
	obj := rt.NewObject()
	obj.Set("env", rt.NewObject())
	obj.Set("args", rt.NewObject())
	return obj
}

func NewModuleEventLoop(
	printer console.Printer,
	modCtx modules.RuntimeContext,
	programs ...*goja.Program,
) (*eventloop.EventLoop, error) {
	loop := modCtx.EventLoop()

	rt := loop.Runtime()
	prepareRuntime(rt)

	registry := loop.Registry()

	if registerModules("dgate", rt,
		registry, modCtx, dgate.New(modCtx),
	); printer == nil {
		printer = &NoopPrinter{}
	}

	registry.RegisterNativeModule(
		"dgate_internal:console",
		console.RequireWithPrinter(printer),
	)

	url.Enable(rt)
	buffer.Enable(rt)
	console.Enable(rt)

	rt.Set("console", require.Require(rt, "dgate_internal:console").ToObject(rt))
	rt.Set("fetch", require.Require(rt, "dgate/http").ToObject(rt).Get("fetch"))
	rt.Set("disableSetInterval", disableSetInterval)

	for _, program := range programs {
		_, err := rt.RunProgram(program)
		if err != nil {
			return nil, err
		}
	}

	return loop, nil
}

// registerModules registers a module and its children with the registry (recursively)
func registerModules(
	modName string,
	rt *goja.Runtime,
	reg *require.Registry,
	modCtx modules.RuntimeContext,
	mod modules.GoModule,
) *goja.Object {
	exports := rt.NewObject()
	// defaultExports := rt.NewObject()
	// TODO: Default exports are being ignore, check to see how we can use both named and default together

	if exportsRaw := mod.Exports(); exportsRaw != nil {
		for childName, childMod := range exportsRaw.Named {
			if inst, ok := childMod.(modules.GoModule); ok {
				// only register children if they are modules
				m := registerModules(
					modName+"/"+childName,
					rt, reg, modCtx, inst,
				)
				exports.Set(childName, m)
				// defaultExports.Set(childName, childMod)
				continue
			}
			exports.Set(childName, childMod)
			// defaultExports.Set(childName, childMod)

			reg.RegisterNativeModule(modName, func(runtime *goja.Runtime, module *goja.Object) {
				if exportsRaw.Default != nil {
					exports.Set("default", exportsRaw.Default)
				}
				module.Set("exports", exports)
			})
		}
	}
	// reg.RegisterNativeModule(modName, func(runtime *goja.Runtime, module *goja.Object) {
	// 	exports.Set("default", defaultExports)
	// 	module.Set("exports", exports)
	// })
	return exports
}

type smartFieldNameMapper struct{}

var _ goja.FieldNameMapper = &smartFieldNameMapper{}

func (*smartFieldNameMapper) FieldName(_ reflect.Type, f reflect.StructField) string {
	if f.Tag.Get("json") != "" {
		return f.Tag.Get("json")
	}
	return strcase.LowerCamelCase(f.Name)
}

func (*smartFieldNameMapper) MethodName(_ reflect.Type, m reflect.Method) string {
	return strcase.LowerCamelCase(m.Name)
}

func disableSetInterval(fc *goja.FunctionCall) (goja.Value, error) {
	return nil, errors.New("setInterval is disabled")
}

package extractors_test

import (
	"testing"

	"github.com/dgate-io/dgate/internal/proxy"
	"github.com/dgate-io/dgate/pkg/modules/extractors"
	"github.com/dgate-io/dgate/pkg/modules/testutil"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/dgate-io/dgate/pkg/typescript"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
)

const TS_PAYLOAD = `
let customFunc = (req: any, upstream: any) => {
	console.log("log 1")
	console.warn("log 2")
	console.error("log 3")
}
export { customFunc }  
`

const JS_PAYLOAD = `
let customFunc = (req, upstream) => {
	console.log("log 1")
	console.warn("log 2")
	console.error("log 3")
}
module.exports = { customFunc }
`

type consolePrinter struct {
	calls map[string]int
}

var _ console.Printer = &consolePrinter{}

func (cp *consolePrinter) Log(string) {
	if _, ok := cp.calls["Log"]; !ok {
		cp.calls["Log"] = 1
	} else {
		cp.calls["Log"]++
	}
}

func (cp *consolePrinter) Warn(string) {
	if _, ok := cp.calls["Warn"]; !ok {
		cp.calls["Warn"] = 1
	} else {
		cp.calls["Warn"]++
	}
}

func (cp *consolePrinter) Error(string) {
	if _, ok := cp.calls["Error"]; !ok {
		cp.calls["Error"] = 1
	} else {
		cp.calls["Error"]++
	}
}

func TestNewModuleRuntimeJS(t *testing.T) {
	programs := map[string]*goja.Program{
		"javascript": testutil.CreateJSProgram(t, JS_PAYLOAD),
		"typescript": testutil.CreateTSProgram(t, TS_PAYLOAD),
	}
	for testName, program := range programs {
		t.Run(testName, func(t *testing.T) {
			printer := testutil.NewMockPrinter()
			printer.On("Log", "log 1").Return().Once()
			printer.On("Warn", "log 2").Return().Once()
			printer.On("Error", "log 3").Return().Once()
			modCtx := testutil.NewMockRuntimeContext()
			loop, err := extractors.NewModuleEventLoop(
				printer, modCtx, program,
			)
			if err != nil {
				t.Fatal(err)
			}
			rt := loop.Start()
			defer loop.Stop()
			val := rt.Get("customFunc")
			if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
				t.Fatal("customFunc not found")
			}
			customFunc, ok := goja.AssertFunction(val)
			if !ok {
				t.Fatal("customFunc is not a function")
			}
			_, err = customFunc(nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			printer.AssertExpectations(t)
		})
	}
}

func TestPrinter(t *testing.T) {
	program := testutil.CreateJSProgram(t, JS_PAYLOAD)
	cp := &consolePrinter{make(map[string]int)}
	rt := &spec.DGateRoute{Namespace: &spec.DGateNamespace{}}
	rtCtx := proxy.NewRuntimeContext(nil, rt)
	loop, err := extractors.NewModuleEventLoop(
		cp, rtCtx, program,
	)
	if err != nil {
		t.Fatal(err)
	}
	loop.RunOnLoop(func(rt *goja.Runtime) {
		val := rt.Get("customFunc")
		if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
			t.Fatal("customFunc not found")
		}
	})
}

func BenchmarkNewModuleRuntime(b *testing.B) {
	program := testutil.CreateTSProgram(b, TS_PAYLOAD)

	b.ResetTimer()
	b.Run("CreateModuleRuntime", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StartTimer()
			rt := &spec.DGateRoute{Namespace: &spec.DGateNamespace{}}
			rtCtx := proxy.NewRuntimeContext(nil, rt)
			_, err := extractors.NewModuleEventLoop(nil, rtCtx, program)
			b.StopTimer()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Transpile-TS", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StartTimer()
			_, err := typescript.Transpile(TS_PAYLOAD)
			if err != nil {
				b.Fatal(err)
			}

			b.StopTimer()
		}
	})

	b.Run("CreateNewProgram-TS", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StartTimer()
			testutil.CreateTSProgram(b, TS_PAYLOAD)
			b.StopTimer()
		}
	})

	b.Run("CreateNewProgram-JS", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StartTimer()
			testutil.CreateJSProgram(b, JS_PAYLOAD)
			b.StopTimer()
		}
	})

}

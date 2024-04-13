package types

import "github.com/dop251/goja"

type Symbolic interface {
	symbols(rt *goja.Runtime) map[string]goja.Value
}

func ToValue(rt *goja.Runtime, v any) goja.Value {
	val := rt.ToValue(v)
	if obj, ok := val.(*goja.Object); ok {
		if sym, ok := v.(Symbolic); ok {
			for name, value := range sym.symbols(rt) {
				obj.SetSymbol(goja.NewSymbol(name), value)
			}
		}
	}
	return val
}

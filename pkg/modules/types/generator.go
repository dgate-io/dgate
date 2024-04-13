package types

import (
	"github.com/dgate-io/dgate/pkg/eventloop"
	"github.com/dop251/goja"
)

type generatorFunc[T any] func(rt *goja.Runtime) (T, bool, error)

func asyncGenerator[T any](loop *eventloop.EventLoop, fn generatorFunc[T]) goja.Value {
	rt := loop.Runtime()
	return rt.ToValue(map[string]any{
		"next": func() (*goja.Promise, error) {
			prom, resolve, reject := rt.NewPromise()
			loop.RunOnLoop(func(rt *goja.Runtime) {
				result, done, err := fn(rt)
				if err != nil {
					reject(rt.NewGoError(err))
					return
				}
				resultObject := map[string]any{"done": done}
				if !done {
					resultObject["value"] = result
				}
				resolve(rt.ToValue(resultObject))
			})
			return prom, nil
		},
	})
}

func generator[T any](loop *eventloop.EventLoop, fn generatorFunc[T]) goja.Value {
	rt := loop.Runtime()
	return rt.ToValue(map[string]any{
		"next": func() (goja.Value, error) {
			result, done, err := fn(rt)
			if err != nil {
				return nil, err
			}
			resultObject := map[string]any{"done": done}
			if !done {
				resultObject["value"] = result
			}
			return rt.ToValue(result), nil
		},
	})
}

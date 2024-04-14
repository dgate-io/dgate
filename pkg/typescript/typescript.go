package typescript

import (
	_ "embed"

	"github.com/clarkmcc/go-typescript"
	"github.com/dop251/goja"
)

// typescript v5.3.3
//
//go:embed typescript.min.js
var tscSource string

func Transpile(src string) (string, error) {
	// transpiles TS into JS with commonjs module and targets es5
	return typescript.TranspileString(src,
		WithCachedTypescriptSource(),
		typescript.WithCompileOptions(map[string]any{
			"module": "commonjs",
			"target": "es5",
		}),
	)
}

var tsSrcProgram = goja.MustCompile("", tscSource, true)

func WithCachedTypescriptSource() typescript.TranspileOptionFunc {
	return func(config *typescript.Config) {
		config.TypescriptSource = tsSrcProgram
	}
}

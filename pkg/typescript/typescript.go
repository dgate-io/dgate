package typescript

import (
	"context"
	_ "embed"
	"strings"

	"github.com/clarkmcc/go-typescript"
	"github.com/dop251/goja"
)

// typescript v5.3.3
//
//go:embed typescript.min.js
var tscSource string

func Transpile(ctx context.Context, src string) (string, error) {
	srcReader := strings.NewReader(src)
	// transpiles TS into JS with commonjs module and targets es5
	return typescript.TranspileCtx(
		ctx, srcReader,
		WithCachedTypescriptSource(),
		typescript.WithPreventCancellation(),
		typescript.WithCompileOptions(map[string]any{
			"module":          "commonjs",
			"target":          "es5",
			"inlineSourceMap": true,
		}),
	)
}

var tsSrcProgram = goja.MustCompile("", tscSource, true)

func WithCachedTypescriptSource() typescript.TranspileOptionFunc {
	return func(config *typescript.Config) {
		config.TypescriptSource = tsSrcProgram
	}
}

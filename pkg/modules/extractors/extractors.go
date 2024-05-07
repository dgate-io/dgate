package extractors

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/dgate-io/dgate/pkg/eventloop"
	"github.com/dgate-io/dgate/pkg/modules/types"
	"github.com/dop251/goja"
)

type (
	// this can be used to create custom load balancing strategies, by default it uses round robin
	FetchUpstreamUrlFunc func(*types.ModuleContext) (*url.URL, error)
	RequestModifierFunc  func(*types.ModuleContext) error
	ResponseModifierFunc func(*types.ModuleContext, *http.Response) error
	ErrorHandlerFunc     func(*types.ModuleContext, error) error
	RequestHandlerFunc   func(*types.ModuleContext) error
)

type Results struct {
	Result  goja.Value
	IsError bool
}

// RunAndWaitForResult can execute a goja function and wait for the result
// if the result is a promise, it will wait for the promise to resolve
func RunAndWaitForResult(
	rt *goja.Runtime,
	fn goja.Callable,
	args ...goja.Value,
) (res goja.Value, err error) {
	tracker := newAsyncTracker()
	rt.SetAsyncContextTracker(tracker)
	defer rt.SetAsyncContextTracker(nil)

	if res, err = fn(nil, args...); err != nil {
		return nil, err
	} else if prom, ok := res.Export().(*goja.Promise); ok {
		ctx, cancel := context.WithTimeout(
			context.TODO(), 30*time.Second,
		)
		defer cancel()
		if err := tracker.waitTimeout(ctx, func() bool {
			return prom.State() != goja.PromiseStatePending
		}); err != nil {
			return nil, errors.New("promise timed out: " + err.Error())
		}
		if prom.State() == goja.PromiseStateRejected {
			return nil, errors.New(prom.Result().String())
		}
		results := prom.Result()
		if nully(results) {
			return nil, nil
		}
		return results, nil
	} else {
		return res, nil
	}
}

func nully(val goja.Value) bool {
	return val == nil || goja.IsUndefined(val) || goja.IsNull(val)
}

func DefaultFetchUpstreamFunction() FetchUpstreamUrlFunc {
	roundRobinIndex := 0
	return func(ctx *types.ModuleContext) (*url.URL, error) {
		svc := ctx.Service()
		if svc.URLs == nil || len(svc.URLs) == 0 {
			return nil, errors.New("service has no URLs")
		}
		roundRobinIndex = (roundRobinIndex + 1) % len(svc.URLs)
		curUrl, err := url.Parse(svc.URLs[roundRobinIndex])
		if err != nil {
			return nil, err
		}
		return curUrl, nil
	}
}

// const _ goja.AsyncContextTracker = ExtractorContextTracker{""}
func ExtractFetchUpstreamFunction(
	loop *eventloop.EventLoop,
) (fetchUpstream FetchUpstreamUrlFunc, err error) {
	rt := loop.Runtime()
	fetchUpstreamRaw := rt.Get("fetchUpstream")
	if call, ok := goja.AssertFunction(fetchUpstreamRaw); ok {
		fetchUpstream = func(modCtx *types.ModuleContext) (*url.URL, error) {
			res, err := RunAndWaitForResult(
				rt, call, rt.ToValue(modCtx),
			)
			if err != nil {
				return nil, err
			}
			upstreamUrlString := res.String()
			if goja.IsUndefined(res) || goja.IsNull(res) || upstreamUrlString == "" {
				return nil, errors.New("fetchUpstream returned an invalid URL")
			}
			upstreamUrl, err := url.Parse(upstreamUrlString)
			if err != nil {
				return nil, err
			}
			// perhaps add default scheme if not present
			return upstreamUrl, err
		}
	} else {
		fetchUpstream = DefaultFetchUpstreamFunction()
	}
	return fetchUpstream, nil
}

func ExtractRequestModifierFunction(
	loop *eventloop.EventLoop,
) (requestModifier RequestModifierFunc, err error) {
	rt := loop.Runtime()
	if call, ok := goja.AssertFunction(rt.Get("requestModifier")); ok {
		requestModifier = func(modCtx *types.ModuleContext) error {
			_, err := RunAndWaitForResult(
				rt, call, types.ToValue(rt, modCtx),
			)
			return err
		}
	}
	return requestModifier, nil
}

func ExtractResponseModifierFunction(
	loop *eventloop.EventLoop,
) (responseModifier ResponseModifierFunc, err error) {
	rt := loop.Runtime()
	if call, ok := goja.AssertFunction(rt.Get("responseModifier")); ok {
		responseModifier = func(modCtx *types.ModuleContext, res *http.Response) error {
			modCtx = types.ModuleContextWithResponse(modCtx, res)
			_, err := RunAndWaitForResult(
				rt, call, types.ToValue(rt, modCtx),
			)
			return err
		}
	}
	return responseModifier, nil
}

func DefaultErrorHandlerFunction() ErrorHandlerFunc {
	return func(modCtx *types.ModuleContext, err error) error {
		rwt := types.GetModuleContextResponseWriterTracker(modCtx)
		if rwt.HeadersSent() {
			return nil
		}
		status := http.StatusBadGateway
		var text string
		switch err {
		case context.Canceled, io.ErrUnexpectedEOF:
			status = 499
			text = "Client Closed Request"
		default:
			text = http.StatusText(status)
		}
		rwt.WriteHeader(status)
		rwt.Write([]byte(text))
		return nil
	}
}

func ExtractErrorHandlerFunction(
	loop *eventloop.EventLoop,
) (errorHandler ErrorHandlerFunc, err error) {
	rt := loop.Runtime()
	if call, ok := goja.AssertFunction(rt.Get("errorHandler")); ok {
		errorHandler = func(modCtx *types.ModuleContext, upstreamErr error) error {
			if modCtx == nil {
				return upstreamErr
			}
			modCtx = types.ModuleContextWithError(modCtx, upstreamErr)
			_, err := RunAndWaitForResult(
				rt, call, rt.ToValue(modCtx),
				rt.ToValue(rt.NewGoError(upstreamErr)),
			)
			return err
		}
	} else {
		errorHandler = DefaultErrorHandlerFunction()
	}
	return errorHandler, nil
}

func ExtractRequestHandlerFunction(
	loop *eventloop.EventLoop,
) (requestHandler RequestHandlerFunc, err error) {
	rt := loop.Runtime()
	if call, ok := goja.AssertFunction(rt.Get("requestHandler")); ok {
		requestHandler = func(modCtx *types.ModuleContext) error {
			if modCtx == nil {
				return errors.New("module context is nil")
			}
			_, err := RunAndWaitForResult(
				rt, call, rt.ToValue(modCtx),
			)
			return err
		}
	}
	return requestHandler, nil
}

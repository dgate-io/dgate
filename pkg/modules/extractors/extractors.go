package extractors

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
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

var _ goja.AsyncContextTracker = &asyncTracker{}

type asyncTracker struct {
	count atomic.Int32
}

type TrackerEvent int

const (
	Exited TrackerEvent = iota
	Resumed
)

func newAsyncTracker() *asyncTracker {
	return &asyncTracker{
		count: atomic.Int32{},
	}
}

// Exited is called when an async function is done
func (t *asyncTracker) Exited() {
	t.count.Add(-1)
}

// Grab is called when an async function is scheduled
func (t *asyncTracker) Grab() any {
	t.count.Add(1)
	return nil
}

// Resumed is called when an async function is executed (ignore)
func (t *asyncTracker) Resumed(any) {}

func (t *asyncTracker) waitTimeout(ctx context.Context) error {
	ticker := time.NewTicker(500 * time.Microsecond)
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout: %s", ctx.Err())
		case <-ticker.C:
			if t.count.Load() == 0 {
				return nil
			}
		}
	}
}

// runAndWaitForResult can execute a goja function and wait for the result
// if the result is a promise, it will wait for the promise to resolve
func runAndWaitForResult(
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
		// return waitForPromise(rt, prom, res)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err := tracker.waitTimeout(ctx)
		if err != nil {
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

func waitForPromise(
	rt *goja.Runtime,
	prom *goja.Promise,
	promVal goja.Value,
) (goja.Value, error) {
	results := make(chan Results, 1)
	promObj := promVal.ToObject(rt)

	if then, ok := goja.AssertFunction(promObj.Get("then")); !ok {
		return nil, errors.New("promise does not have a then function")
	} else {
		val := rt.ToValue(func(call goja.FunctionCall) {
			result := goja.Undefined()
			if len(call.Arguments) > 0 {
				result = call.Arguments[0]
			}
			results <- Results{result, false}
		})
		if _, err := then(promObj, val); err != nil {
			return nil, err
		}
	}

	if catch, ok := goja.AssertFunction(promObj.Get("catch")); !ok {
		return nil, errors.New("promise does not have a then function")
	} else {
		val := rt.ToValue(func(call goja.FunctionCall) {
			var result goja.Value
			if len(call.Arguments) > 0 {
				result = call.Arguments[0]
			}
			results <- Results{result, true}
		})
		if _, err := catch(promObj, val); err != nil {
			return nil, err
		}
	}

	select {
	case res := <-results:
		if res.IsError {
			if !nully(prom.Result()) {
				return nil, errors.New(prom.Result().String())
			}
			if !nully(res.Result) {
				return nil, errors.New(res.Result.String())
			}
			return nil, errors.New("promise rejected: unknown reason")
		}
		if prom.Result() != nil {
			return prom.Result(), nil
		}
		return res.Result, nil
	case <-time.After(30 * time.Second):
		return nil, errors.New("promise timed out")
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
			res, err := runAndWaitForResult(
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
			_, err := runAndWaitForResult(
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
			_, err := runAndWaitForResult(
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
			modCtx = types.ModuleContextWithError(modCtx, upstreamErr)
			_, err := runAndWaitForResult(
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
			_, err := runAndWaitForResult(
				rt, call, rt.ToValue(modCtx),
			)
			return err
		}
	}
	return requestHandler, nil
}

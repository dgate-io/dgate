package proxy

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/dgate-io/chi-router"
	"github.com/dgate-io/dgate/pkg/modules/types"
	"github.com/dgate-io/dgate/pkg/util"
)

type ProxyHandlerFunc func(ps *ProxyState, reqCtx *RequestContext)

func proxyHandler(ps *ProxyState, reqCtx *RequestContext) {
	defer ps.metrics.MeasureProxyRequest(reqCtx, time.Now())

	defer func() {
		if reqCtx.req.Body != nil {
			// Ensure that the request body is drained/closed, so the connection can be reused
			io.Copy(io.Discard, reqCtx.req.Body)
			reqCtx.req.Body.Close()
		}

		event := ps.logger.Debug().
			Str("route", reqCtx.route.Name).
			Str("namespace", reqCtx.route.Namespace.Name)

		if reqCtx.route.Service != nil {
			event = event.
				Str("service", reqCtx.route.Service.Name)
		}
		event.Msg("Request")
	}()

	var modExt ModuleExtractor
	if len(reqCtx.route.Modules) != 0 {
		runtimeStart := time.Now()
		if reqCtx.provider.modBuf == nil {
			ps.logger.Error().Msg("Error getting module buffer: invalid state")
			util.WriteStatusCodeError(reqCtx.rw, http.StatusInternalServerError)
			return
		}
		var ok bool
		modExt, ok = reqCtx.provider.modBuf.Borrow()
		if !ok || modExt == nil {
			ps.logger.Error().Msg("Error borrowing module")
			util.WriteStatusCodeError(reqCtx.rw, http.StatusInternalServerError)
			return
		}

		if rtCtx, ok := modExt.RuntimeContext().(*runtimeContext); ok {
			pathParams := make(map[string]string)
			if chiCtx := chi.RouteContext(reqCtx.req.Context()); chiCtx != nil {
				for i, key := range chiCtx.URLParams.Keys {
					pathParams[key] = chiCtx.URLParams.Values[i]
				}
			}
			// set request context for runtime
			rtCtx.SetRequestContext(reqCtx, pathParams)
			// set module context for runtime
			modExt.SetModuleContext(
				types.NewModuleContext(
					rtCtx.loop, reqCtx.rw, reqCtx.req,
					rtCtx.reqCtx.route, pathParams,
				),
			)
			// TODO: consider passing context to properly close
			modExt.Start()

			defer func() {
				rtCtx.Clean()
				modExt.Stop(true)
				reqCtx.provider.modBuf.Return(modExt)
			}()
		} else {
			ps.metrics.MeasureModuleDuration(
				reqCtx, "module_extract",
				runtimeStart, errors.New("invalid runtime context"),
			)
			ps.logger.Error().Msg("Error getting runtime context")
			util.WriteStatusCodeError(reqCtx.rw, http.StatusInternalServerError)
			return
		}

		ps.metrics.MeasureModuleDuration(
			reqCtx, "module_extract",
			runtimeStart, nil,
		)
	} else {
		modExt = NewEmptyModuleExtractor()
	}

	if reqCtx.route.Service != nil {
		handleServiceProxy(ps, reqCtx, modExt)
	} else {
		requestHandlerModule(ps, reqCtx, modExt)
	}
}

func handleServiceProxy(ps *ProxyState, reqCtx *RequestContext, modExt ModuleExtractor) {
	var host string
	if fetchUpstreamUrl, ok := modExt.FetchUpstreamUrlFunc(); ok {
		fetchUpstreamStart := time.Now()
		hostUrl, err := fetchUpstreamUrl(modExt.ModuleContext())
		ps.metrics.MeasureModuleDuration(
			reqCtx, "fetch_upstream",
			fetchUpstreamStart, err,
		)
		if err != nil {
			ps.logger.Err(err).Msg("Error fetching upstream")
			util.WriteStatusCodeError(reqCtx.rw, http.StatusInternalServerError)
			return
		}
		host = hostUrl.String()
	} else {
		if reqCtx.route.Service.URLs == nil || len(reqCtx.route.Service.URLs) == 0 {
			ps.logger.Error().Msg("Error getting service urls")
			util.WriteStatusCodeError(reqCtx.rw, http.StatusInternalServerError)
			return
		}
		host = reqCtx.route.Service.URLs[0].String()
	}

	if reqCtx.route.Service.HideDGateHeaders {
		if ps.debugMode {
			reqCtx.rw.Header().Set("X-Upstream-URL", host)
		}
		reqCtx.req.Header.Set("X-DGate-Service", reqCtx.route.Service.Name)
		reqCtx.req.Header.Set("X-DGate-Route", reqCtx.route.Name)
		reqCtx.req.Header.Set("X-DGate-Namespace", reqCtx.route.Namespace.Name)
		for _, tag := range ps.config.Tags {
			reqCtx.req.Header.Add("X-DGate-Tags", tag)
		}
	}
	upstreamUrl, err := url.Parse(host)
	if err != nil {
		ps.logger.Err(err).Msg("Error parsing upstream url")
		util.WriteStatusCodeError(reqCtx.rw, http.StatusBadGateway)
		return
	}

	var upstreamErr error
	rpb := reqCtx.provider.rpb.Clone().
		ModifyResponse(func(res *http.Response) error {
			if reqCtx.route.Service.HideDGateHeaders {
				res.Header.Set("Via", "DGate Proxy")
			}
			if responseModifier, ok := modExt.ResponseModifierFunc(); ok {
				resModifierStart := time.Now()
				err = responseModifier(modExt.ModuleContext(), res)
				ps.metrics.MeasureModuleDuration(
					reqCtx, "response_modifier",
					resModifierStart, err,
				)
				if err != nil {
					ps.logger.Err(err).Msg("Error modifying response")
					return err
				}
			}
			return nil
		}).
		ErrorHandler(func(w http.ResponseWriter, r *http.Request, reqErr error) {
			upstreamErr = reqErr
			ps.logger.Debug().Err(reqErr).Msg("Error proxying request")
			// TODO: add metric for error
			if reqCtx.rw.HeadersSent() {
				return
			}
			if errorHandler, ok := modExt.ErrorHandlerFunc(); ok {
				errorHandlerStart := time.Now()
				err = errorHandler(modExt.ModuleContext(), reqErr)
				ps.metrics.MeasureModuleDuration(
					reqCtx, "error_handler",
					errorHandlerStart, err,
				)
				if err != nil {
					ps.logger.Err(err).Msg("Error handling error")
					util.WriteStatusCodeError(reqCtx.rw, http.StatusInternalServerError)
					return
				}
			} else {
				util.WriteStatusCodeError(reqCtx.rw, http.StatusInternalServerError)
				return
			}
		})

	if requestModifier, ok := modExt.RequestModifierFunc(); ok {
		reqModifierStart := time.Now()
		err = requestModifier(modExt.ModuleContext())
		ps.metrics.MeasureModuleDuration(
			reqCtx, "request_modifier",
			reqModifierStart, err,
		)
		if err != nil {
			ps.logger.Err(err).Msg("Error modifying request")
			util.WriteStatusCodeError(reqCtx.rw, http.StatusInternalServerError)
			return
		}
	}

	rp, err := rpb.Build(upstreamUrl, reqCtx.pattern)
	if err != nil {
		ps.logger.Err(err).Msg("Error creating reverse proxy")
		util.WriteStatusCodeError(reqCtx.rw, http.StatusInternalServerError)
		return
	}
	// Set Upstream Response Headers
	for k, v := range ps.config.ProxyConfig.GlobalHeaders {
		reqCtx.rw.Header().Set(k, v)
	}

	upstreamStart := time.Now()
	rp.ServeHTTP(reqCtx.rw, reqCtx.req)
	ps.metrics.MeasureUpstreamDuration(
		reqCtx, upstreamStart,
		upstreamUrl.String(), upstreamErr,
	)
}

func requestHandlerModule(ps *ProxyState, reqCtx *RequestContext, modExt ModuleExtractor) {
	var err error
	if requestModifier, ok := modExt.RequestModifierFunc(); ok {
		// extract request modifier function from module
		reqModifierStart := time.Now()
		err = requestModifier(modExt.ModuleContext())
		ps.metrics.MeasureModuleDuration(
			reqCtx, "request_modifier",
			reqModifierStart, err,
		)
		if err != nil {
			ps.logger.Err(err).Msg("Error modifying request")
			util.WriteStatusCodeError(reqCtx.rw, http.StatusInternalServerError)
			return
		}
	}
	if requestHandler, ok := modExt.RequestHandlerFunc(); ok {
		requestHandlerStart := time.Now()
		err := requestHandler(modExt.ModuleContext())
		defer ps.metrics.MeasureModuleDuration(
			reqCtx, "request_handler",
			requestHandlerStart, err,
		)
		if err != nil {
			ps.logger.Error().Err(err).Msg("Error @ request_handler module")
			if errorHandler, ok := modExt.ErrorHandlerFunc(); ok {
				// extract error handler function from module
				errorHandlerStart := time.Now()
				err = errorHandler(modExt.ModuleContext(), err)
				ps.metrics.MeasureModuleDuration(
					reqCtx, "error_handler",
					errorHandlerStart, err,
				)
				if err != nil {
					ps.logger.Err(err).Msg("Error handling error")
					util.WriteStatusCodeError(reqCtx.rw, http.StatusInternalServerError)
					return
				}
			} else {
				util.WriteStatusCodeError(reqCtx.rw, http.StatusInternalServerError)
				return
			}
		} else {
			if !reqCtx.rw.HeadersSent() {
				if reqCtx.rw.BytesWritten() > 0 {
					reqCtx.rw.WriteHeader(http.StatusOK)
				} else {
					reqCtx.rw.WriteHeader(http.StatusNoContent)
				}
			}
		}
	} else {
		util.WriteStatusCodeError(reqCtx.rw, http.StatusNotImplemented)
		return
	}
}

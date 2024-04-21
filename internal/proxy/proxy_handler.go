package proxy

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/dgate-io/chi-router"
	"github.com/dgate-io/dgate/pkg/modules/types"
	"github.com/dgate-io/dgate/pkg/util"
)

type ProxyHandlerFunc func(ps *ProxyState, reqCtx *RequestContext)

func proxyHandler(ps *ProxyState, reqCtx *RequestContext) {
	rs := NewRequestStats(reqCtx.route)
	var err error
	defer func() {
		event := ps.logger.Debug().
			Str("route", reqCtx.route.Name).
			Str("namespace", reqCtx.route.Namespace.Name)
		if err != nil {
			event = event.Err(err)
		}
		if reqCtx.route.Service != nil {
			event = event.
				Str("service", reqCtx.route.Service.Name)
			event = event.
				Stringer("upstream", rs.UpstreamRequestDur)
		}
		for k, v := range rs.MiscDurs {
			event = event.Stringer(k, v)
		}
		event.Msg("[STATS] Request Latency")
		ps.stats.AddRequestStats(rs)
	}()

	defer func() {
		if reqCtx.req.Body != nil {
			// Ensure that the request body is drained/closed, so the connection can be reused
			io.Copy(io.Discard, reqCtx.req.Body)
			reqCtx.req.Body.Close()
		}
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
			ps.logger.Error().Msg("Error getting runtime context")
			util.WriteStatusCodeError(reqCtx.rw, http.StatusInternalServerError)
			return
		}

		runtimeElapsed := time.Since(runtimeStart)
		rs.AddMiscDuration("moduleExtract", runtimeElapsed)
		ps.logger.Trace().
			Str("duration", runtimeElapsed.String()).
			Msg("[STATS] Runtime Created")
	} else {
		modExt = NewEmptyModuleExtractor()
	}

	if reqCtx.route.Service != nil {
		handleServiceProxy(ps, reqCtx, modExt, rs)
	} else {
		requestHandlerModule(ps, reqCtx, modExt, rs)
	}
}

func handleServiceProxy(ps *ProxyState, reqCtx *RequestContext, modExt ModuleExtractor, rs *RequestStats) {
	var host string
	if fetchUpstreamUrl, ok := modExt.FetchUpstreamUrlFunc(); ok {
		fetchUpstreamStart := time.Now()
		hostUrl, err := fetchUpstreamUrl(modExt.ModuleContext())
		if err != nil {
			ps.logger.Err(err).Msg("Error fetching upstream")
			util.WriteStatusCodeError(reqCtx.rw, http.StatusInternalServerError)
			return
		}
		host = hostUrl.String()
		fetchUpstreamElapsed := time.Since(fetchUpstreamStart)
		rs.AddMiscDuration("fetchUpstreamUrl", fetchUpstreamElapsed)
		ps.logger.Trace().
			Str("duration", fetchUpstreamElapsed.String()).
			Msg("[STATS] fetch upstream module")
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
	proxyTransport := setupTranportFromConfig(
		ps.config.ProxyConfig.Transport,
		func(dialer *net.Dialer, t *http.Transport) {
			t.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: reqCtx.route.Service.TLSSkipVerify,
			}
			dialer.Timeout = reqCtx.route.Service.ConnectTimeout
			t.ForceAttemptHTTP2 = reqCtx.route.Service.HTTP2Only
		},
	)

	ptb := ps.ProxyTransportBuilder.Clone().
		Transport(proxyTransport).
		Retries(reqCtx.route.Service.Retries).
		RetryTimeout(reqCtx.route.Service.RetryTimeout).
		RequestTimeout(reqCtx.route.Service.RequestTimeout)

	proxy, err := ptb.Build()
	if err != nil {
		ps.logger.Err(err).Msg("Error creating proxy transport")
		util.WriteStatusCodeError(reqCtx.rw, http.StatusInternalServerError)
		return
	}

	rpb := ps.ReverseProxyBuilder.Clone().
		Transport(proxy).FlushInterval(-1).
		ProxyRewrite(
			reqCtx.route.StripPath,
			reqCtx.route.PreserveHost,
			reqCtx.route.Service.DisableQueryParams,
			ps.config.ProxyConfig.DisableXForwardedHeaders,
		).
		ModifyResponse(func(res *http.Response) error {
			if reqCtx.route.Service.HideDGateHeaders {
				res.Header.Set("Via", "DGate Proxy")
			}
			if responseModifier, ok := modExt.ResponseModifierFunc(); ok {
				resModifierStart := time.Now()
				err = responseModifier(modExt.ModuleContext(), res)
				resModifierElapsed := time.Since(resModifierStart)
				if err != nil {
					ps.logger.Err(err).Msg("Error modifying response")
					return err
				}
				rs.AddMiscDuration("responseModifier", resModifierElapsed)
				ps.logger.Trace().
					Str("duration", resModifierElapsed.String()).
					Msg("[STATS] respond modifier module")
			}
			return nil
		}).
		ErrorHandler(func(w http.ResponseWriter, r *http.Request, reqErr error) {
			ps.logger.Debug().Err(reqErr).Msg("Error proxying request")
			if reqCtx.rw.HeadersSent() {
				return
			}
			if errorHandler, ok := modExt.ErrorHandlerFunc(); ok {
				errorHandlerStart := time.Now()
				err = errorHandler(modExt.ModuleContext(), reqErr)
				errorHandlerElapsed := time.Since(errorHandlerStart)
				rs.AddMiscDuration("errorHandler", errorHandlerElapsed)
				ps.logger.Trace().
					Str("duration", errorHandlerElapsed.String()).
					Msg("[STATS] error handler module")
				ps.logger.Trace().Err(reqErr).Msg("Error proxying request")
				if err != nil {
					ps.logger.Err(err).Msg("Error handling error")
					util.WriteStatusCodeError(reqCtx.rw, http.StatusInternalServerError)
					return
				}
			} else {
				ps.logger.Err(reqErr).Msg("Error handling error")
				util.WriteStatusCodeError(reqCtx.rw, http.StatusInternalServerError)
				return
			}
		})

	if requestModifier, ok := modExt.RequestModifierFunc(); ok {
		reqModifierStart := time.Now()
		err = requestModifier(modExt.ModuleContext())
		if err != nil {
			ps.logger.Err(err).Msg("Error modifying request")
			util.WriteStatusCodeError(reqCtx.rw, http.StatusInternalServerError)
			return
		}
		reqModifierElapsed := time.Since(reqModifierStart)
		rs.AddMiscDuration("requestModifier", reqModifierElapsed)
		ps.logger.Trace().
			Str("duration", reqModifierElapsed.String()).
			Msg("[STATS] request modifier module")
	}

	upstreamStart := time.Now()
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

	rp.ServeHTTP(reqCtx.rw, reqCtx.req)

	upstreamElapsed := time.Since(upstreamStart)
	rs.AddUpstreamRequestDuration(upstreamElapsed)
	ps.logger.Trace().
		Str("duration", upstreamElapsed.String()).
		Msg("[STATS] upstream")
}

func requestHandlerModule(ps *ProxyState, reqCtx *RequestContext, modExt ModuleExtractor, rs *RequestStats) {
	var err error
	if requestModifier, ok := modExt.RequestModifierFunc(); ok {
		// extract request modifier function from module
		reqModifierStart := time.Now()
		err = requestModifier(modExt.ModuleContext())
		if err != nil {
			ps.logger.Err(err).Msg("Error modifying request")
			util.WriteStatusCodeError(reqCtx.rw, http.StatusInternalServerError)
			return
		}
		reqModifierElapsed := time.Since(reqModifierStart)
		rs.AddMiscDuration("requestModifier", reqModifierElapsed)
		ps.logger.Trace().
			Str("duration", reqModifierElapsed.String()).
			Msg("[STATS] request modifier module")
	}
	if requestHandler, ok := modExt.RequestHandlerFunc(); ok {
		requestHandlerStart := time.Now()
		if err := requestHandler(modExt.ModuleContext()); err != nil {
			ps.logger.Error().Err(err).Msg("Error handling request")
			if errorHandler, ok := modExt.ErrorHandlerFunc(); ok {
				// extract error handler function from module
				errorHandlerStart := time.Now()
				if err = errorHandler(modExt.ModuleContext(), err); err != nil {
					ps.logger.Err(err).Msg("Error handling error")
					util.WriteStatusCodeError(reqCtx.rw, http.StatusInternalServerError)
					return
				}
				errorHandlerElapsed := time.Since(errorHandlerStart)
				rs.AddMiscDuration("errorHandler", errorHandlerElapsed)
				ps.logger.Trace().
					Str("duration", errorHandlerElapsed.String()).
					Msg("[STATS] error handler module")
			} else {
				ps.logger.Err(err).Msg("Error handling request")
				util.WriteStatusCodeError(reqCtx.rw, http.StatusInternalServerError)
				return
			}
		} else {
			requestHandlerElapsed := time.Since(requestHandlerStart)
			rs.AddMiscDuration("requestHandler", requestHandlerElapsed)
			ps.logger.Trace().
				Str("duration", requestHandlerElapsed.String()).
				Msg("[STATS] request handler module")
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

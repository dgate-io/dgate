package proxy

import (
	"context"
	"time"

	"github.com/dgate-io/dgate/internal/config"
	"github.com/dgate-io/dgate/internal/extensions/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	api "go.opentelemetry.io/otel/metric"
)

type Telemetry struct {
	resolveNamespaceDurInstrument api.Float64Histogram
	resolveCertDurInstrument      api.Float64Histogram
	proxyDurInstrument            api.Float64Histogram
	proxyCountInstrument          api.Int64Counter
	moduleDurInstrument           api.Float64Histogram
	moduleRunCountInstrument      api.Int64Counter
	upstreamDurInstrument         api.Float64Histogram
	errorCountInstrument          api.Int64Counter
}

func NewProxyMetrics() *Telemetry {
	return &Telemetry{}
}

func (tm *Telemetry) Setup(config *config.DGateConfig) {
	if config.DisableMetrics {
		return
	}
	meter := otel.Meter("dgate-proxy-metrics", api.WithInstrumentationAttributes(
		attribute.KeyValue{
			Key: "storage", Value: attribute.StringValue(string(config.Storage.StorageType)),
		},
		attribute.KeyValue{
			Key: "node_id", Value: attribute.StringValue(config.NodeId),
		},
		attribute.KeyValue{
			Key: "tag", Value: attribute.StringSliceValue(config.Tags),
		},
	))

	tm.resolveNamespaceDurInstrument, _ = meter.Float64Histogram(
		"resolve_namespace_duration", api.WithUnit("us"))
	tm.resolveCertDurInstrument, _ = meter.Float64Histogram(
		"resolve_cert_duration", api.WithUnit("ms"))
	tm.proxyDurInstrument, _ = meter.Float64Histogram(
		"request_duration", api.WithUnit("ms"))
	tm.moduleDurInstrument, _ = meter.Float64Histogram(
		"module_duration", api.WithUnit("ms"))
	tm.upstreamDurInstrument, _ = meter.Float64Histogram(
		"upstream_duration", api.WithUnit("ms"))
	tm.proxyCountInstrument, _ = meter.Int64Counter(
		"request_count")
	tm.moduleRunCountInstrument, _ = meter.Int64Counter(
		"module_executions")
	tm.errorCountInstrument, _ = meter.Int64Counter(
		"error_count")
}

func (tm *Telemetry) MeasureProxyRequest(
	reqCtx *RequestContext, start time.Time,
) {
	if tm.proxyDurInstrument == nil || tm.proxyCountInstrument == nil {
		return
	}
	serviceAttr := attribute.NewSet()
	if reqCtx.route.Service != nil {
		serviceAttr = attribute.NewSet(
			attribute.String("service", reqCtx.route.Service.Name),
		)
	}

	elasped := time.Since(start)
	userAgent := reqCtx.req.UserAgent()
	if maxUaLen := 256; len(userAgent) > maxUaLen {
		userAgent = userAgent[:maxUaLen]
	}
	attrSet := attribute.NewSet(
		attribute.String("route", reqCtx.route.Name),
		attribute.String("namespace", reqCtx.route.Namespace.Name),
		attribute.String("method", reqCtx.req.Method),
		attribute.String("path", reqCtx.req.URL.Path),
		attribute.String("pattern", reqCtx.pattern),
		attribute.String("host", reqCtx.req.Host),
		attribute.String("remote_addr", reqCtx.req.RemoteAddr),
		attribute.String("user_agent", userAgent),
		attribute.String("proto", reqCtx.req.Proto),
		attribute.Int64("content_length", reqCtx.req.ContentLength),
		attribute.Int("status_code", reqCtx.rw.Status()),
	)

	tm.proxyDurInstrument.Record(reqCtx.ctx,
		float64(elasped)/float64(time.Millisecond),
		api.WithAttributeSet(attrSet), api.WithAttributeSet(serviceAttr))

	tm.proxyCountInstrument.Add(reqCtx.ctx, 1,
		api.WithAttributeSet(attrSet), api.WithAttributeSet(serviceAttr))
}

func (tm *Telemetry) MeasureModuleDuration(
	reqCtx *RequestContext, moduleFunc string,
	start time.Time, err error,
) {
	if tm.moduleDurInstrument == nil || tm.moduleRunCountInstrument == nil {
		return
	}
	elasped := time.Since(start)
	attrSet := attribute.NewSet(
		attribute.Bool("error", err != nil),
		attribute.String("route", reqCtx.route.Name),
		attribute.String("namespace", reqCtx.route.Namespace.Name),
		attribute.String("moduleFunc", moduleFunc),
		attribute.String("method", reqCtx.req.Method),
		attribute.String("path", reqCtx.req.URL.Path),
		attribute.String("pattern", reqCtx.pattern),
		attribute.String("host", reqCtx.req.Host),
	)
	tm.addError(moduleFunc, err, attrSet)

	tm.moduleDurInstrument.Record(reqCtx.ctx,
		float64(elasped)/float64(time.Millisecond),
		api.WithAttributeSet(attrSet))

	tm.moduleRunCountInstrument.Add(reqCtx.ctx, 1,
		api.WithAttributeSet(attrSet))
}

func (tm *Telemetry) MeasureUpstreamDuration(
	reqCtx *RequestContext, start time.Time,
	upstreamHost string, err error,
) {
	if tm.upstreamDurInstrument == nil {
		return
	}
	elasped := time.Since(start)
	attrSet := attribute.NewSet(
		attribute.Bool("error", err != nil),
		attribute.String("route", reqCtx.route.Name),
		attribute.String("namespace", reqCtx.route.Namespace.Name),
		attribute.String("method", reqCtx.req.Method),
		attribute.String("path", reqCtx.req.URL.Path),
		attribute.String("pattern", reqCtx.pattern),
		attribute.String("host", reqCtx.req.Host),
		attribute.String("service", reqCtx.route.Service.Name),
		attribute.String("upstream_host", upstreamHost),
	)
	tm.addError("upstream_request", err, attrSet)

	tm.upstreamDurInstrument.Record(reqCtx.ctx,
		float64(elasped)/float64(time.Millisecond),
		api.WithAttributeSet(attrSet))
}

func (tm *Telemetry) MeasureNamespaceResolutionDuration(
	start time.Time, host, namespace string, err error,
) {
	if tm.resolveNamespaceDurInstrument == nil {
		return
	}
	elasped := time.Since(start)
	attrSet := attribute.NewSet(
		attribute.String("host", host),
		attribute.String("namespace", namespace),
	)
	tm.addError("namespace_resolution", err, attrSet)

	tm.resolveNamespaceDurInstrument.Record(context.TODO(),
		float64(elasped)/float64(time.Microsecond),
		api.WithAttributeSet(attrSet))
}

func (tm *Telemetry) MeasureCertResolutionDuration(
	start time.Time, host string, cache bool, err error,
) {
	if tm.resolveCertDurInstrument == nil {
		return
	}

	elasped := time.Since(start)
	attrSet := attribute.NewSet(
		attribute.Bool("error", err != nil),
		attribute.String("host", host),
		attribute.Bool("cache", cache),
	)
	tm.addError("cert_resolution", err, attrSet)

	tm.resolveCertDurInstrument.Record(context.TODO(),
		float64(elasped)/float64(time.Millisecond),
		api.WithAttributeSet(attrSet))
}

func (tm *Telemetry) addError(
	namespace string, err error,
	attrs ...attribute.Set,
) {
	if tm.errorCountInstrument == nil || err == nil {
		return
	}
	telemetry.CaptureError(err)
	attrSet := attribute.NewSet(
		attribute.String("error_value", err.Error()),
		attribute.String("namespace", namespace),
	)

	attrSets := []api.AddOption{
		api.WithAttributeSet(attrSet),
	}
	for _, attr := range attrs {
		attrSets = append(attrSets, api.WithAttributeSet(attr))
	}

	tm.errorCountInstrument.Add(context.TODO(), 1, attrSets...)
}

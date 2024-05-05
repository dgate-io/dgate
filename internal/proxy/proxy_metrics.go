package proxy

import (
	"context"
	"time"

	"github.com/dgate-io/dgate/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	api "go.opentelemetry.io/otel/metric"
)

type ProxyMetrics struct {
	resolveNamespaceDurInstrument api.Float64Histogram
	resolveCertDurInstrument      api.Float64Histogram
	proxyDurInstrument            api.Float64Histogram
	proxyCountInstrument          api.Float64Counter
	moduleDurInstrument           api.Float64Histogram
	moduleRunCountInstrument      api.Float64Counter
	upstreamDurInstrument         api.Float64Histogram
	resourceDurInstrument         api.Float64Histogram
}

func NewProxyMetrics() *ProxyMetrics {
	return &ProxyMetrics{}
}

func (pm *ProxyMetrics) Setup(config *config.DGateConfig) {
	meter := otel.Meter("dgate-proxy-metrics", api.WithInstrumentationAttributes(
		attribute.KeyValue{
			Key: "tag", Value: attribute.StringSliceValue(config.Tags),
		},
		attribute.KeyValue{
			Key: "storage", Value: attribute.StringValue(string(config.Storage.StorageType)),
		},
	))

	pm.resolveNamespaceDurInstrument, _ = meter.Float64Histogram(
		"resolve_namespace_duration", api.WithUnit("us"))
	pm.resolveCertDurInstrument, _ = meter.Float64Histogram(
		"resolve_cert_duration", api.WithUnit("ms"))
	pm.proxyDurInstrument, _ = meter.Float64Histogram(
		"request_duration", api.WithUnit("ms"))
	pm.moduleDurInstrument, _ = meter.Float64Histogram(
		"module_duration", api.WithUnit("ms"))
	pm.upstreamDurInstrument, _ = meter.Float64Histogram(
		"upstream_duration", api.WithUnit("ms"))
	pm.resourceDurInstrument, _ = meter.Float64Histogram(
		"resource_duration", api.WithUnit("ms"))
	pm.proxyCountInstrument, _ = meter.Float64Counter(
		"request_count")
	pm.moduleRunCountInstrument, _ = meter.Float64Counter(
		"module_executions")
}

func (pm *ProxyMetrics) MeasureProxyRequest(
	start time.Time, reqCtx *RequestContext,
) {
	if pm.proxyDurInstrument == nil || pm.proxyCountInstrument == nil {
		return
	}
	serviceAttr := attribute.NewSet()
	if reqCtx.route.Service != nil {
		serviceAttr = attribute.NewSet(
			attribute.String("service", reqCtx.route.Service.Name),
			attribute.StringSlice("service_tag", reqCtx.route.Service.Tags),
		)
	}
	elasped := time.Since(start)
	attrSet := attribute.NewSet(
		attribute.String("route", reqCtx.route.Name),
		attribute.String("namespace", reqCtx.route.Namespace.Name),
		attribute.String("method", reqCtx.req.Method),
		attribute.String("path", reqCtx.req.URL.Path),
		attribute.String("pattern", reqCtx.pattern),
		attribute.String("host", reqCtx.req.Host),
		attribute.StringSlice("route_tag", reqCtx.route.Tags),
	)

	pm.proxyDurInstrument.Record(reqCtx.ctx,
		float64(elasped)/float64(time.Millisecond),
		api.WithAttributeSet(attrSet), api.WithAttributeSet(serviceAttr))

	pm.proxyCountInstrument.Add(reqCtx.ctx, 1,
		api.WithAttributeSet(attrSet), api.WithAttributeSet(serviceAttr))
}

func (pm *ProxyMetrics) MeasureModuleDuration(moduleFunc string, start time.Time, reqCtx *RequestContext) {
	if pm.moduleDurInstrument == nil || pm.moduleRunCountInstrument == nil {
		return
	}
	elasped := time.Since(start)
	attrSet := attribute.NewSet(
		attribute.String("route", reqCtx.route.Name),
		attribute.String("namespace", reqCtx.route.Namespace.Name),
		attribute.String("moduleFunc", moduleFunc),
		attribute.String("method", reqCtx.req.Method),
		attribute.String("path", reqCtx.req.URL.Path),
		attribute.String("pattern", reqCtx.pattern),
		attribute.String("host", reqCtx.req.Host),
		attribute.StringSlice("route_tag", reqCtx.route.Tags),
	)

	pm.moduleDurInstrument.Record(reqCtx.ctx,
		float64(elasped)/float64(time.Millisecond),
		api.WithAttributeSet(attrSet))

	pm.moduleRunCountInstrument.Add(reqCtx.ctx, 1,
		api.WithAttributeSet(attrSet))
}

func (pm *ProxyMetrics) MeasureUpstreamDuration(
	start time.Time, upstreamHost string,
	reqCtx *RequestContext,
) {
	if pm.upstreamDurInstrument == nil {
		return
	}
	elasped := time.Since(start)
	attrSet := attribute.NewSet(
		attribute.String("route", reqCtx.route.Name),
		attribute.String("namespace", reqCtx.route.Namespace.Name),
		attribute.String("method", reqCtx.req.Method),
		attribute.String("path", reqCtx.req.URL.Path),
		attribute.String("pattern", reqCtx.pattern),
		attribute.String("host", reqCtx.req.Host),
		attribute.String("service", reqCtx.route.Service.Name),
		attribute.String("upstream_host", upstreamHost),
		attribute.StringSlice("service_tag", reqCtx.route.Service.Tags),
		attribute.StringSlice("route_tag", reqCtx.route.Tags),
	)

	pm.upstreamDurInstrument.Record(reqCtx.ctx,
		float64(elasped)/float64(time.Millisecond),
		api.WithAttributeSet(attrSet))
}

// func (pm *ProxyMetrics) MeasureResourceDuration(
// 	start time.Time, resource, namespace string,
// ) {
// 	if pm.resourceDurInstrument == nil {
// 		return
// 	}
// 	elasped := time.Since(start)
// 	attrSet := attribute.NewSet(
// 		attribute.String("resource", resource),
// 		attribute.String("namespace", namespace),
// 	)

// 	pm.resourceDurInstrument.Record(context.TODO(),
// 		float64(elasped)/float64(time.Millisecond),
// 		api.WithAttributeSet(attrSet))
// }

func (pm *ProxyMetrics) MeasureNamespaceResolutionDuration(
	start time.Time, host, namespace string,
) {
	if pm.resolveNamespaceDurInstrument == nil {
		return
	}
	elasped := time.Since(start)
	attrSet := attribute.NewSet(
		attribute.String("host", host),
		attribute.String("namespace", namespace),
	)

	pm.resolveNamespaceDurInstrument.Record(context.TODO(),
		float64(elasped)/float64(time.Microsecond),
		api.WithAttributeSet(attrSet))
}

func (pm *ProxyMetrics) MeasureCertResolutionDuration(
	start time.Time, host string, cache bool,
) {
	if pm.resolveCertDurInstrument == nil {
		return
	}
	elasped := time.Since(start)
	attrSet := attribute.NewSet(
		attribute.String("host", host),
		attribute.Bool("cache", cache),
	)

	pm.resolveCertDurInstrument.Record(context.TODO(),
		float64(elasped)/float64(time.Millisecond),
		api.WithAttributeSet(attrSet))
}

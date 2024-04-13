package proxy

import (
	"fmt"
	"sync"
	"time"

	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/montanaflynn/stats"
)

type ProxyStats struct {
	serviceRequestCount         map[string]int64
	RouteRequestCount           map[string]int64
	LastNRequestsDurs           []float64
	LastNModExtractDurs         []float64
	lock                        *sync.RWMutex
	count                       int64
	windowSize                  int
	amean, stdev, variance, sum float64
}

type RequestStats struct {
	Service            *spec.DGateService
	Route              *spec.DGateRoute
	UpstreamRequestDur time.Duration
	MiscDurs           map[string]time.Duration
}

func NewProxyStats(windowSize int) *ProxyStats {
	return &ProxyStats{
		lock:                &sync.RWMutex{},
		windowSize:          windowSize,
		serviceRequestCount: make(map[string]int64),
		RouteRequestCount:   make(map[string]int64),
		LastNRequestsDurs:   make([]float64, 0, windowSize),
		LastNModExtractDurs: make([]float64, 0, windowSize),
	}
}

func NewRequestStats(route *spec.DGateRoute) *RequestStats {
	return &RequestStats{
		Service:  route.Service,
		Route:    route,
		MiscDurs: make(map[string]time.Duration),
	}
}

func (rs *RequestStats) AddMiscDuration(name string, dur time.Duration) {
	rs.MiscDurs[name] = dur
}

func (rs *RequestStats) AddUpstreamRequestDuration(dur time.Duration) {
	rs.UpstreamRequestDur = dur
}

func (rs *RequestStats) String() string {
	return fmt.Sprintf(
		"service=%s route=%s upstream_request_dur=%s module_durs=%s",
		rs.Service.Name,
		rs.Route.Name,
		rs.UpstreamRequestDur,
		rs.MiscDurs,
	)
}

func (ps *ProxyStats) AddRequestStats(rs *RequestStats) {
	reqDur := float64(rs.UpstreamRequestDur.Milliseconds())
	ps.lock.Lock()
	defer ps.lock.Unlock()
	ps.count++

	if ps.count > 1 {
		ps.amean, _ = stats.Mean(ps.LastNRequestsDurs)
		ps.variance, _ = stats.Variance(ps.LastNRequestsDurs)
		ps.sum, _ = stats.Sum(ps.LastNRequestsDurs)
		if ps.count < 3 {
			goto SKIP
		}
		ps.stdev, _ = stats.StandardDeviationSample(ps.LastNRequestsDurs)
	}
SKIP:

	if rs.Service != nil {
		if src, ok := ps.serviceRequestCount[rs.Service.Name]; ok {
			ps.serviceRequestCount[rs.Service.Name] = src + 1
		} else {
			ps.serviceRequestCount[rs.Service.Name] = 1
		}
	}

	if rs.Route != nil {
		if src, ok := ps.RouteRequestCount[rs.Route.Name]; ok {
			ps.RouteRequestCount[rs.Route.Name] = src + 1
		} else {
			ps.RouteRequestCount[rs.Route.Name] = 1
		}
	}
	reqDurLen := len(ps.LastNRequestsDurs)
	if reqDurLen >= ps.windowSize {
		ps.LastNRequestsDurs = ps.LastNRequestsDurs[1:]
		ps.LastNModExtractDurs = ps.LastNModExtractDurs[1:]
	}
	ps.LastNRequestsDurs = append(ps.LastNRequestsDurs, reqDur)
	ps.LastNModExtractDurs = append(ps.LastNModExtractDurs, float64(rs.MiscDurs["moduleExtract"].Nanoseconds()))
}

func (ps *ProxyStats) Snapshot() map[string]any {
	data := make(map[string]any)
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	data["req_count"] = ps.count
	data["service_request_count"] = ps.serviceRequestCount
	data["route_request_count"] = ps.RouteRequestCount
	// data["request_durs"] = ps.LastNRequestsDurs
	data["mod_extracts_ns"] = ps.LastNModExtractDurs
	data["req_dur_mean_ms"] = ps.amean
	data["req_dur_stddev_ms"] = ps.stdev
	data["req_dur_variance_ms"] = ps.variance
	data["req_dur_sum_ms"] = ps.sum
	data["req_dur_window_size"] = ps.windowSize
	return data
}

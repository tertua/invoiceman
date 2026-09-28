package metrics

// Package metrics is a minimal Prometheus-compatible observability layer
// with zero new dependencies (stdlib only: sync/atomic).
//
// Recorded per request (see Recorder middleware):
//   - tupay_http_requests_total{method,route,status}
//   - tupay_http_request_duration_seconds (global histogram)
//
// Rendered on scrape:
//   - tupay_app_info{version}
//   - tupay_uptime_seconds
//   - tupay_db_open_conns{backend}, _idle, _in_use
//
// Route labels use the matched Fiber route template (e.g. /api/clients/:id),
// never raw paths, to bound cardinality. /metrics and /events are not recorded.

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v3"
)

// latencyBounds are histogram upper bounds in seconds (Prometheus style).
var latencyBounds = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

type counterKey struct {
	method string
	route  string
	status int
}

// Registry holds counters and the latency histogram.
type Registry struct {
	mu       sync.Mutex
	counters map[counterKey]*atomic.Uint64

	buckets  []atomic.Uint64
	sumNanos atomic.Uint64
	count    atomic.Uint64

	startTime time.Time
	version   string

	// dbStats is an optional hook returning (open, idle, inUse, backend, ok).
	// Wired by main to platform/database to avoid an import cycle.
	dbStats func() (open, idle, inUse int, backend string, ok bool)
}

// New returns an empty registry with the given build version.
func New(version string) *Registry {
	r := &Registry{
		counters:  make(map[counterKey]*atomic.Uint64),
		buckets:   make([]atomic.Uint64, len(latencyBounds)),
		startTime: time.Now(),
		version:   version,
	}
	return r
}

// SetDBStats wires the database pool snapshot hook.
func (r *Registry) SetDBStats(fn func() (open, idle, inUse int, backend string, ok bool)) {
	r.dbStats = fn
}

var (
	defaultOnce     sync.Once
	defaultRegistry *Registry
)

// Init creates the shared registry (version shown in app_info).
func Init(version string) *Registry {
	defaultOnce.Do(func() {
		defaultRegistry = New(version)
	})
	return defaultRegistry
}

// Shared returns the shared registry, creating a dev default when Init
// was not called (tests).
func Shared() *Registry {
	if defaultRegistry == nil {
		return Init("dev")
	}
	return defaultRegistry
}

// Observe records one finished request.
func (r *Registry) Observe(method, route string, status int, d time.Duration) {
	if route == "" {
		route = "unknown"
	}
	key := counterKey{method: method, route: route, status: status}
	r.mu.Lock()
	c, ok := r.counters[key]
	if !ok {
		c = &atomic.Uint64{}
		r.counters[key] = c
	}
	r.mu.Unlock()
	c.Add(1)

	// Durations are non-negative in practice, but a negative value would
	// wrap on conversion; clamp it instead of overflowing the buckets.
	nanos := d.Nanoseconds()
	if nanos < 0 {
		nanos = 0
	}
	unanos := uint64(nanos)
	secs := float64(unanos) / 1e9
	for i, bound := range latencyBounds {
		if secs <= bound {
			r.buckets[i].Add(1)
		}
	}
	r.sumNanos.Add(unanos)
	r.count.Add(1)
}

// Recorder is Fiber middleware recording method/route/status/latency.
// The scrape endpoint is served but never recorded (no self-measurement).
func Recorder() fiber.Handler {
	reg := Shared()
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		if SkipMetrics(c.Path()) {
			return err
		}
		route := "unknown"
		if rt := c.Route(); rt != nil && rt.Path != "" {
			route = rt.Path
		}
		reg.Observe(c.Method(), route, c.Response().StatusCode(), time.Since(start))
		return err
	}
}

// SkipMetrics reports whether the path must not be recorded nor counted
// (the scrape endpoint measuring itself, the never-closing event stream).
func SkipMetrics(path string) bool { return path == "/metrics" || strings.HasSuffix(path, "/events") }

// Render exposes the registry in Prometheus text exposition format.
func (r *Registry) Render() string {
	var b strings.Builder

	fmt.Fprintf(&b, "# HELP tupay_app_info Application build info.\n")
	fmt.Fprintf(&b, "# TYPE tupay_app_info gauge\n")
	fmt.Fprintf(&b, "tupay_app_info{version=%s} 1\n", quote(r.version))

	fmt.Fprintf(&b, "# HELP tupay_uptime_seconds Seconds since process start.\n")
	fmt.Fprintf(&b, "# TYPE tupay_uptime_seconds counter\n")
	fmt.Fprintf(&b, "tupay_uptime_seconds %s\n", formatFloat(time.Since(r.startTime).Seconds()))

	fmt.Fprintf(&b, "# HELP tupay_http_requests_total Total HTTP requests.\n")
	fmt.Fprintf(&b, "# TYPE tupay_http_requests_total counter\n")
	keys := make([]counterKey, 0)
	r.mu.Lock()
	for k := range r.counters {
		keys = append(keys, k)
	}
	r.mu.Unlock()
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].route != keys[j].route {
			return keys[i].route < keys[j].route
		}
		if keys[i].method != keys[j].method {
			return keys[i].method < keys[j].method
		}
		return keys[i].status < keys[j].status
	})
	for _, k := range keys {
		r.mu.Lock()
		v := r.counters[k].Load()
		r.mu.Unlock()
		fmt.Fprintf(&b, "tupay_http_requests_total{method=%s,route=%s,status=%s} %d\n",
			quote(k.method), quote(k.route), quote(strconv.Itoa(k.status)), v)
	}

	fmt.Fprintf(&b, "# HELP tupay_http_request_duration_seconds Request latency.\n")
	fmt.Fprintf(&b, "# TYPE tupay_http_request_duration_seconds histogram\n")
	var cumulative uint64
	for i, bound := range latencyBounds {
		cumulative += r.buckets[i].Load()
		fmt.Fprintf(&b, "tupay_http_request_duration_seconds_bucket{le=%s} %d\n",
			formatFloat(bound), cumulative)
	}
	total := r.count.Load()
	fmt.Fprintf(&b, "tupay_http_request_duration_seconds_bucket{le=%s} %d\n", quote("+Inf"), total)
	fmt.Fprintf(&b, "tupay_http_request_duration_seconds_sum %s\n",
		formatFloat(float64(r.sumNanos.Load())/1e9))
	fmt.Fprintf(&b, "tupay_http_request_duration_seconds_count %d\n", total)

	if r.dbStats != nil {
		if open, idle, inUse, backend, ok := r.dbStats(); ok {
			fmt.Fprintf(&b, "# HELP tupay_db_open_conns Database pool connections.\n")
			fmt.Fprintf(&b, "# TYPE tupay_db_open_conns gauge\n")
			fmt.Fprintf(&b, "tupay_db_open_conns{backend=%s} %d\n", quote(backend), open)
			fmt.Fprintf(&b, "# HELP tupay_db_idle_conns Database idle connections.\n")
			fmt.Fprintf(&b, "# TYPE tupay_db_idle_conns gauge\n")
			fmt.Fprintf(&b, "tupay_db_idle_conns{backend=%s} %d\n", quote(backend), idle)
			fmt.Fprintf(&b, "# HELP tupay_db_in_use_conns Database connections in use.\n")
			fmt.Fprintf(&b, "# TYPE tupay_db_in_use_conns gauge\n")
			fmt.Fprintf(&b, "tupay_db_in_use_conns{backend=%s} %d\n", quote(backend), inUse)
		}
	}

	return b.String()
}

func quote(s string) string { return strconv.Quote(s) }

func formatFloat(f float64) string { return strconv.FormatFloat(f, 'g', -1, 64) }

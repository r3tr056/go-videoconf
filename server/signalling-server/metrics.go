package main

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "videoconf_http_requests_total",
		Help: "HTTP requests by service, method, route pattern and status.",
	}, []string{"service", "method", "path", "status"})

	httpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "videoconf_http_request_duration_seconds",
		Help:    "HTTP request latency by service, method and route pattern.",
		Buckets: prometheus.DefBuckets,
	}, []string{"service", "method", "path"})

	wsConnectionsCurrent = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "videoconf_ws_connections_current",
		Help: "Currently registered signalling WebSocket members.",
	})

	wsMessagesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "videoconf_ws_messages_total",
		Help: "Signalling messages handled by type.",
	}, []string{"type"})
)

// metricsMiddleware records RED metrics per request. Route patterns (not raw
// paths) keep label cardinality bounded; /metrics itself is excluded.
func metricsMiddleware(service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.FullPath() == "/metrics" {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}
		status := strconv.Itoa(c.Writer.Status())
		httpRequestsTotal.WithLabelValues(service, c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(service, c.Request.Method, path).Observe(time.Since(start).Seconds())
	}
}

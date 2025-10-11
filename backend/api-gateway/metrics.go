package main

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// =============================================================================
// GATEWAY METRICS
// =============================================================================

// GatewayMetrics tracks performance and usage metrics for the API Gateway
type GatewayMetrics struct {
	RequestsTotal     *prometheus.CounterVec
	RequestDuration   *prometheus.HistogramVec
	ActiveConnections prometheus.Gauge
	ServiceHealth     *prometheus.GaugeVec
	CacheHits         *prometheus.CounterVec
	RateLimitHits     prometheus.Counter
	GraphQLOperations *prometheus.CounterVec
}

// initializeMetrics creates and registers all Prometheus metrics
func initializeMetrics() *GatewayMetrics {
	return &GatewayMetrics{
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_requests_total",
				Help: "Total number of HTTP requests processed by the gateway",
			},
			[]string{"method", "endpoint", "status_code"},
		),

		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gateway_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),

		ActiveConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "gateway_active_connections",
			Help: "Number of active connections to the gateway",
		}),

		ServiceHealth: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gateway_service_health",
				Help: "Health status of backend services (1=healthy, 0=unhealthy)",
			},
			[]string{"service"},
		),

		CacheHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_cache_operations_total",
				Help: "Total number of cache operations",
			},
			[]string{"operation", "result"}, // hit, miss, error
		),

		RateLimitHits: promauto.NewCounter(prometheus.CounterOpts{
			Name: "gateway_rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		}),

		GraphQLOperations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_graphql_operations_total",
				Help: "Total number of GraphQL operations by type",
			},
			[]string{"operation_type", "operation_name"},
		),
	}
}

// RecordRequest records metrics for an HTTP request
func (m *GatewayMetrics) RecordRequest(method, path string, statusCode int, duration time.Duration) {
	statusStr := getStatusClass(statusCode)

	m.RequestsTotal.WithLabelValues(method, path, statusStr).Inc()
	m.RequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

// RecordServiceHealth updates service health metrics
func (m *GatewayMetrics) RecordServiceHealth(serviceName string, isHealthy bool) {
	var value float64
	if isHealthy {
		value = 1
	}
	m.ServiceHealth.WithLabelValues(serviceName).Set(value)
}

// RecordCacheOperation records cache hit/miss metrics
func (m *GatewayMetrics) RecordCacheOperation(operation, result string) {
	m.CacheHits.WithLabelValues(operation, result).Inc()
}

// RecordRateLimitHit increments rate limit counter
func (m *GatewayMetrics) RecordRateLimitHit() {
	m.RateLimitHits.Inc()
}

// RecordGraphQLOperation records GraphQL operation metrics
func (m *GatewayMetrics) RecordGraphQLOperation(operationType, operationName string) {
	m.GraphQLOperations.WithLabelValues(operationType, operationName).Inc()
}

// getStatusClass converts HTTP status code to class for metrics
func getStatusClass(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return "2xx"
	case statusCode >= 300 && statusCode < 400:
		return "3xx"
	case statusCode >= 400 && statusCode < 500:
		return "4xx"
	case statusCode >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

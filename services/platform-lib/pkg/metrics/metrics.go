package metrics

import (
	"net/http"
	"time"

	"github.com/athena/platform-lib/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics provides Prometheus metrics collection
type Metrics struct {
	registry *prometheus.Registry

	// HTTP metrics
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	httpRequestSize     *prometheus.HistogramVec
	httpResponseSize    *prometheus.HistogramVec

	// Application metrics
	activeConnections prometheus.Gauge
	totalConnections  prometheus.Counter
	errorsTotal       *prometheus.CounterVec
	cacheHits         *prometheus.CounterVec
	cacheMisses       *prometheus.CounterVec
	dbConnections     prometheus.Gauge
	dbQueriesTotal    *prometheus.CounterVec
	dbQueryDuration   *prometheus.HistogramVec

	// Business metrics
	devicesRegistered prometheus.Gauge
	devicesOnline     prometheus.Gauge
	templatesDeployed prometheus.Gauge
	otaUpdatesTotal   *prometheus.CounterVec

	logger logger.Logger
}

// NewMetrics creates a new metrics instance
func NewMetrics(serviceName string, logger logger.Logger) *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		logger:   logger,
	}

	// Initialize HTTP metrics
	m.httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		},
		[]string{"method", "endpoint", "status_code"},
	)

	m.httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "HTTP request duration in seconds",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	m.httpRequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_size_bytes",
			Help: "HTTP request size in bytes",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
			Buckets: []float64{100, 1000, 10000, 100000, 1000000},
		},
		[]string{"method", "endpoint"},
	)

	m.httpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_response_size_bytes",
			Help: "HTTP response size in bytes",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
			Buckets: []float64{100, 1000, 10000, 100000, 1000000},
		},
		[]string{"method", "endpoint"},
	)

	// Initialize application metrics
	m.activeConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_connections",
			Help: "Number of active connections",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		},
	)

	m.totalConnections = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "total_connections",
			Help: "Total number of connections",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		},
	)

	m.errorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "errors_total",
			Help: "Total number of errors",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		},
		[]string{"type", "component"},
	)

	m.cacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		},
		[]string{"cache_type"},
	)

	m.cacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		},
		[]string{"cache_type"},
	)

	m.dbConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections",
			Help: "Number of active database connections",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		},
	)

	m.dbQueriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_queries_total",
			Help: "Total number of database queries",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		},
		[]string{"operation", "table"},
	)

	m.dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "db_query_duration_seconds",
			Help: "Database query duration in seconds",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "table"},
	)

	// Initialize business metrics
	m.devicesRegistered = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "devices_registered_total",
			Help: "Total number of registered devices",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		},
	)

	m.devicesOnline = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "devices_online_total",
			Help: "Number of online devices",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		},
	)

	m.templatesDeployed = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "templates_deployed_total",
			Help: "Total number of deployed templates",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		},
	)

	m.otaUpdatesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ota_updates_total",
			Help: "Total number of OTA updates",
			ConstLabels: prometheus.Labels{
				"service": serviceName,
			},
		},
		[]string{"status", "device_type"},
	)

	// Register all metrics
	m.registry.MustRegister(
		m.httpRequestsTotal,
		m.httpRequestDuration,
		m.httpRequestSize,
		m.httpResponseSize,
		m.activeConnections,
		m.totalConnections,
		m.errorsTotal,
		m.cacheHits,
		m.cacheMisses,
		m.dbConnections,
		m.dbQueriesTotal,
		m.dbQueryDuration,
		m.devicesRegistered,
		m.devicesOnline,
		m.templatesDeployed,
		m.otaUpdatesTotal,
	)

	logger.Info("Metrics initialized", "service", serviceName)
	return m
}

// Middleware returns a Gin middleware for collecting HTTP metrics
func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Increment request counter
		m.httpRequestsTotal.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			"", // Status code will be set after the request
		).Inc()

		// Record request size
		if c.Request.ContentLength > 0 {
			m.httpRequestSize.WithLabelValues(
				c.Request.Method,
				c.FullPath(),
			).Observe(float64(c.Request.ContentLength))
		}

		// Increment active connections
		m.activeConnections.Inc()
		m.totalConnections.Inc()

		// Process request
		c.Next()

		// Decrement active connections
		m.activeConnections.Dec()

		// Record response metrics
		duration := time.Since(start).Seconds()
		statusCode := c.Writer.Status()

		m.httpRequestDuration.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
		).Observe(duration)

		m.httpRequestsTotal.WithLabelValues(
			c.Request.Method,
			c.FullPath(),
			string(rune(statusCode)),
		).Inc()

		// Record response size
		if c.Writer.Size() > 0 {
			m.httpResponseSize.WithLabelValues(
				c.Request.Method,
				c.FullPath(),
			).Observe(float64(c.Writer.Size()))
		}

		// Record errors
		if statusCode >= 400 {
			m.errorsTotal.WithLabelValues(
				"http_error",
				"gin_handler",
			).Inc()
		}
	}
}

// RecordCacheHit records a cache hit
func (m *Metrics) RecordCacheHit(cacheType string) {
	m.cacheHits.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss records a cache miss
func (m *Metrics) RecordCacheMiss(cacheType string) {
	m.cacheMisses.WithLabelValues(cacheType).Inc()
}

// RecordDBQuery records a database query
func (m *Metrics) RecordDBQuery(operation, table string, duration time.Duration) {
	m.dbQueriesTotal.WithLabelValues(operation, table).Inc()
	m.dbQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// RecordError records an error
func (m *Metrics) RecordError(errorType, component string) {
	m.errorsTotal.WithLabelValues(errorType, component).Inc()
}

// SetDevicesRegistered sets the number of registered devices
func (m *Metrics) SetDevicesRegistered(count float64) {
	m.devicesRegistered.Set(count)
}

// SetDevicesOnline sets the number of online devices
func (m *Metrics) SetDevicesOnline(count float64) {
	m.devicesOnline.Set(count)
}

// SetTemplatesDeployed sets the number of deployed templates
func (m *Metrics) SetTemplatesDeployed(count float64) {
	m.templatesDeployed.Set(count)
}

// RecordOTAUpdate records an OTA update
func (m *Metrics) RecordOTAUpdate(status, deviceType string) {
	m.otaUpdatesTotal.WithLabelValues(status, deviceType).Inc()
}

// SetDBConnections sets the number of active database connections
func (m *Metrics) SetDBConnections(count float64) {
	m.dbConnections.Set(count)
}

// Handler returns the Prometheus metrics handler
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// GetRegistry returns the Prometheus registry
func (m *Metrics) GetRegistry() *prometheus.Registry {
	return m.registry
}

// RegisterCustomMetric registers a custom metric
func (m *Metrics) RegisterCustomMetric(metric prometheus.Collector) error {
	return m.registry.Register(metric)
}

// UnregisterMetric unregisters a metric
func (m *Metrics) UnregisterMetric(metric prometheus.Collector) bool {
	return m.registry.Unregister(metric)
}

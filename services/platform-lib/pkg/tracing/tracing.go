package tracing

import (
	"context"
	"fmt"
	"time"

	"github.com/athena/platform-lib/pkg/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// TracerConfig holds tracing configuration
type TracerConfig struct {
	ServiceName   string  `json:"service_name"`
	Environment   string  `json:"environment"`
	Provider      string  `json:"provider"` // jaeger, zipkin, stdout
	Endpoint      string  `json:"endpoint"`
	SampleRate    float64 `json:"sample_rate"`
	EnableTracing bool    `json:"enable_tracing"`
}

// TracingManager manages distributed tracing
type TracingManager struct {
	config   *TracerConfig
	tracer   trace.Tracer
	provider *sdktrace.TracerProvider
	logger   *logger.Logger
}

// NewTracingManager creates a new tracing manager
func NewTracingManager(config *TracerConfig, logger *logger.Logger) (*TracingManager, error) {
	if config == nil {
		config = &TracerConfig{
			ServiceName:   "athena-service",
			Environment:   "development",
			Provider:      "stdout",
			SampleRate:    1.0,
			EnableTracing: true,
		}
	}

	tm := &TracingManager{
		config: config,
		logger: logger,
	}

	if config.EnableTracing {
		if err := tm.initializeTracing(); err != nil {
			return nil, fmt.Errorf("failed to initialize tracing: %w", err)
		}
	}

	return tm, nil
}

// initializeTracing initializes the tracing provider
func (tm *TracingManager) initializeTracing() error {
	var exporter sdktrace.SpanExporter
	var err error

	switch tm.config.Provider {
	case "jaeger":
		exporter, err = tm.createJaegerExporter()
	case "zipkin":
		exporter, err = tm.createZipkinExporter()
	case "stdout":
		exporter, err = tm.createStdoutExporter()
	default:
		return fmt.Errorf("unsupported tracing provider: %s", tm.config.Provider)
	}

	if err != nil {
		return fmt.Errorf("failed to create exporter: %w", err)
	}

	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(tm.config.ServiceName),
			semconv.ServiceVersionKey.String("1.0.0"),
			semconv.DeploymentEnvironmentKey.String(tm.config.Environment),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	tm.provider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(tm.config.SampleRate)),
	)

	// Register as global tracer provider
	otel.SetTracerProvider(tm.provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Create tracer
	tm.tracer = tm.provider.Tracer(tm.config.ServiceName)

	tm.logger.Infof("Tracing initialized with provider: %s", tm.config.Provider)

	return nil
}

// createJaegerExporter creates a Jaeger exporter
func (tm *TracingManager) createJaegerExporter() (sdktrace.SpanExporter, error) {
	endpoint := tm.config.Endpoint
	if endpoint == "" {
		endpoint = "http://localhost:14268/api/traces"
	}

	return jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)))
}

// createZipkinExporter creates a Zipkin exporter
func (tm *TracingManager) createZipkinExporter() (sdktrace.SpanExporter, error) {
	endpoint := tm.config.Endpoint
	if endpoint == "" {
		endpoint = "http://localhost:9411/api/v2/spans"
	}

	return zipkin.New(endpoint)
}

// createStdoutExporter creates a stdout exporter
func (tm *TracingManager) createStdoutExporter() (sdktrace.SpanExporter, error) {
	// For now, return a simple exporter that doesn't use stdouttrace
	// In production, you would implement a proper stdout exporter
	return nil, fmt.Errorf("stdout exporter not implemented - use jaeger or zipkin instead")
}

// GetTracer returns the tracer
func (tm *TracingManager) GetTracer() trace.Tracer {
	if tm.tracer == nil {
		// Return a no-op tracer when tracing is disabled
		return otel.Tracer("noop")
	}
	return tm.tracer
}

// StartSpan starts a new span
func (tm *TracingManager) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if tm.tracer == nil {
		// Tracing disabled, return no-op span
		return ctx, trace.SpanFromContext(ctx)
	}

	return tm.tracer.Start(ctx, name, opts...)
}

// AddSpanAttributes adds attributes to the current span
func (tm *TracingManager) AddSpanAttributes(ctx context.Context, attributes ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.SetAttributes(attributes...)
	}
}

// AddSpanEvent adds an event to the current span
func (tm *TracingManager) AddSpanEvent(ctx context.Context, name string, attributes ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.AddEvent(name, trace.WithAttributes(attributes...))
	}
}

// RecordError records an error in the current span
func (tm *TracingManager) RecordError(ctx context.Context, err error, attributes ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span != nil {
		attrs := append([]attribute.KeyValue{
			attribute.String("error.message", err.Error()),
			attribute.String("error.type", fmt.Sprintf("%T", err)),
		}, attributes...)

		span.SetAttributes(attrs...)
		span.SetStatus(codes.Error, err.Error())
	}
}

// GetTraceID returns the trace ID from the context
func (tm *TracingManager) GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span != nil && span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID returns the span ID from the context
func (tm *TracingManager) GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span != nil && span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// Shutdown shuts down the tracing manager
func (tm *TracingManager) Shutdown(ctx context.Context) error {
	if tm.provider != nil {
		return tm.provider.Shutdown(ctx)
	}
	return nil
}

// TraceFunction wraps a function with tracing
func (tm *TracingManager) TraceFunction(ctx context.Context, functionName string, fn func(context.Context) error) error {
	ctx, span := tm.StartSpan(ctx, functionName)
	defer span.End()

	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start)

	// Add duration attribute
	tm.AddSpanAttributes(ctx, attribute.Float64("duration_ms", float64(duration.Milliseconds())))

	if err != nil {
		tm.RecordError(ctx, err)
		return err
	}

	tm.AddSpanAttributes(ctx, attribute.String("status", "success"))
	return nil
}

// CorrelationID represents a correlation ID for request tracking
type CorrelationID string

// NewCorrelationID generates a new correlation ID
func NewCorrelationID() CorrelationID {
	return CorrelationID(fmt.Sprintf("%d", time.Now().UnixNano()))
}

// String returns the string representation
func (cid CorrelationID) String() string {
	return string(cid)
}

// WithCorrelationID adds correlation ID to context
func WithCorrelationID(ctx context.Context, cid CorrelationID) context.Context {
	return context.WithValue(ctx, correlationIDKey{}, cid)
}

// GetCorrelationID gets correlation ID from context
func GetCorrelationID(ctx context.Context) CorrelationID {
	if cid, ok := ctx.Value(correlationIDKey{}).(CorrelationID); ok {
		return cid
	}
	return ""
}

type correlationIDKey struct{}

// RequestTracer provides request-level tracing utilities
type RequestTracer struct {
	tracingManager *TracingManager
}

// NewRequestTracer creates a new request tracer
func NewRequestTracer(tm *TracingManager) *RequestTracer {
	return &RequestTracer{
		tracingManager: tm,
	}
}

// TraceRequest traces an HTTP request
func (rt *RequestTracer) TraceRequest(ctx context.Context, method, path string, headers map[string]string) (context.Context, trace.Span) {
	// Generate correlation ID if not present
	cid := GetCorrelationID(ctx)
	if cid == "" {
		cid = NewCorrelationID()
		ctx = WithCorrelationID(ctx, cid)
	}

	// Start span
	ctx, span := rt.tracingManager.StartSpan(ctx, fmt.Sprintf("%s %s", method, path))

	// Add request attributes
	attributes := []attribute.KeyValue{
		attribute.String("http.method", method),
		attribute.String("http.path", path),
		attribute.String("correlation.id", cid.String()),
	}

	// Add user agent if available
	if userAgent := headers["User-Agent"]; userAgent != "" {
		attributes = append(attributes, attribute.String("http.user_agent", userAgent))
	}

	// Add trace ID from headers if present
	if traceID := headers["X-Trace-Id"]; traceID != "" {
		attributes = append(attributes, attribute.String("incoming.trace_id", traceID))
	}

	rt.tracingManager.AddSpanAttributes(ctx, attributes...)

	return ctx, span
}

// TraceResponse traces an HTTP response
func (rt *RequestTracer) TraceResponse(ctx context.Context, statusCode int, contentLength int64) {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return
	}

	attributes := []attribute.KeyValue{
		attribute.Int("http.status_code", statusCode),
	}

	if contentLength > 0 {
		attributes = append(attributes, attribute.Int64("http.response_content_length", contentLength))
	}

	rt.tracingManager.AddSpanAttributes(ctx, attributes...)

	// Set span status based on status code
	if statusCode >= 400 {
		span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
	} else {
		span.SetStatus(codes.Ok, "HTTP request completed successfully")
	}
}

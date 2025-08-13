package otel

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/config/schema"
	"github.com/alfin-efendy/helper-go/logger"
	"github.com/alfin-efendy/helper-go/util"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Define custom type for context keys to avoid collisions
type contextKey string

const (
	spanIDContextKey contextKey = "spanID"
)

var (
	otelInstance Otel
	configs      *schema.Config
	serviceName  string
	Shutdown     = func(context.Context) error {
		return nil
	}
)

type Otel interface {
	Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, *SpanWrapper)
	AddCounter(ctx context.Context, counterName string, unit string) error
	Count(ctx context.Context, counterName string, incr int64, opts ...metric.AddOption)
}

type SpanWrapper struct {
	span trace.Span
}

type otelWrapper struct {
	tracer   trace.Tracer
	meter    metric.Meter
	counters sync.Map // Thread-safe map for counters
}

func NewOtel(tracer trace.Tracer, meter metric.Meter, counters map[string]metric.Int64Counter) Otel {
	wrapper := &otelWrapper{
		tracer: tracer,
		meter:  meter,
	}

	// Load existing counters into sync.Map
	for name, counter := range counters {
		wrapper.counters.Store(name, counter)
	}

	return wrapper
}

// validateConfig validates the OpenTelemetry configuration
func validateConfig(cfg *schema.Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration is nil")
	}

	if cfg.Otel.Trace || cfg.Otel.Metric {
		if cfg.Otel.Address == "" {
			return fmt.Errorf("OTLP endpoint address is required when tracing or metrics are enabled")
		}

		if cfg.Otel.Timeout <= 0 {
			return fmt.Errorf("timeout must be positive, got %d", cfg.Otel.Timeout)
		}
	}

	if cfg.App.Name == "" {
		return fmt.Errorf("service name is required")
	}

	return nil
}

func Init() {
	ctx := context.Background()
	configs = config.Data

	// Validate configuration
	if err := validateConfig(configs); err != nil {
		logger.Error(ctx, err, "Invalid OpenTelemetry configuration")
		return
	}

	// Check if OpenTelemetry is enabled
	if !configs.Otel.Trace && !configs.Otel.Metric {
		logger.Warn(ctx, "OpenTelemetry is disabled")
		return
	}

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	serviceName = configs.App.Name
	res, err := resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithTelemetrySDK(),
		resource.WithHost(),
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		logger.Fatal(ctx, err, "Failed to create resource")
		return
	}

	// Initialize grpc connection
	conn, err := initGrpcConn(configs.Otel.Address)
	if err != nil {
		logger.Fatal(ctx, err, "Failed to initialize gRPC connection")
		return
	}

	// Initialize shutdown hook
	var shutdownHooks []func(context.Context) error

	// Add connection cleanup
	shutdownHooks = append(shutdownHooks, func(ctx context.Context) error {
		return conn.Close()
	})

	Shutdown = func(ctx context.Context) error {
		var errs []error
		for _, hook := range shutdownHooks {
			if hookErr := hook(ctx); hookErr != nil {
				errs = append(errs, hookErr)
			}
		}
		shutdownHooks = nil
		return errors.Join(errs...)
	}

	if configs.Otel.Trace {
		// Initialize trace provider
		tracerProvider, err := initTracerProvider(ctx, res, conn)
		if err != nil {
			logger.Fatal(ctx, err, "Failed to initialize OpenTelemetry trace provider")
			return
		}

		// Add tracer provider shutdown hook
		shutdownHooks = append(shutdownHooks, func(ctx context.Context) error {
			return tracerProvider.Shutdown(ctx)
		})
	}

	if configs.Otel.Metric {
		// Initialize metric provider
		meterProvider, err := initMetricProvider(ctx, res, conn)
		if err != nil {
			logger.Fatal(ctx, err, "Failed to initialize OpenTelemetry metric provider")
			return
		}

		// Add meter provider shutdown hook
		shutdownHooks = append(shutdownHooks, func(ctx context.Context) error {
			return meterProvider.Shutdown(ctx)
		})
	}

	// Set default tracer
	tracer := otel.Tracer(serviceName)

	// Set default meter
	meter := otel.Meter(serviceName)

	// Init default counters
	counters := make(map[string]metric.Int64Counter)
	otelInstance = NewOtel(tracer, meter, counters)
}

func initGrpcConn(address string) (*grpc.ClientConn, error) {
	if address == "" {
		return nil, fmt.Errorf("gRPC address cannot be empty")
	}

	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to %s: %w", address, err)
	}

	return conn, nil
}

func initTracerProvider(ctx context.Context, res *resource.Resource, conn *grpc.ClientConn) (*sdktrace.TracerProvider, error) {
	conf := configs.Otel

	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithGRPCConn(conn),
		otlptracegrpc.WithTimeout(time.Duration(conf.Timeout)*time.Second),
	)
	if err != nil {
		return nil, err
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.1)), // 10% sampling instead of always sampling
	)

	otel.SetTracerProvider(tracerProvider)

	return tracerProvider, nil
}

func initMetricProvider(ctx context.Context, res *resource.Resource, conn *grpc.ClientConn) (*sdkmetric.MeterProvider, error) {
	conf := configs.Otel

	exporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithGRPCConn(conn),
		otlpmetricgrpc.WithTimeout(time.Duration(conf.Timeout)*time.Second),
	)
	if err != nil {
		return nil, err
	}

	// Make metric interval configurable, default to 30 seconds
	interval := 30 * time.Second
	if conf.Timeout > 0 {
		interval = time.Duration(conf.Timeout) * time.Second
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter,
				sdkmetric.WithInterval(interval),
			),
		),
		sdkmetric.WithResource(res),
	)

	otel.SetMeterProvider(meterProvider)

	return meterProvider, nil
}

func (o *otelWrapper) Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, *SpanWrapper) {
	// Get parent span if any
	sc := trace.SpanContextFromContext(ctx)
	ctx = context.WithValue(ctx, spanIDContextKey, sc.SpanID().String())

	var span trace.Span
	ctx, span = o.tracer.Start(ctx, spanName, opts...)

	return ctx, &SpanWrapper{span}
}

func Trace(ctx context.Context, opts ...trace.SpanStartOption) (context.Context, *SpanWrapper) {
	name := "unknown"

	pc, file, line, ok := runtime.Caller(1)
	if ok {
		opts = append(opts, trace.WithAttributes(
			semconv.CodeLineNumberKey.Int(line),
			semconv.CodeFilepathKey.String(file),
			semconv.CodeFunctionKey.String(runtime.FuncForPC(pc).Name()),
		))

		fullName := util.GetFrame(1).Function
		fullNames := strings.Split(fullName, "/")

		name = fullNames[len(fullNames)-1]
	}

	return otelInstance.Trace(ctx, name, opts...)
}

func (w *SpanWrapper) End(options ...trace.SpanEndOption) {
	w.span.End(options...)
}

// SetSpanStatus sets the status of the span
func (w *SpanWrapper) SetStatus(code codes.Code, description string) {
	w.span.SetStatus(code, description)
}

// AddEvent adds an event to the span
func (w *SpanWrapper) AddEvent(name string, options ...trace.EventOption) {
	w.span.AddEvent(name, options...)
}

// SetAttributes sets attributes on the span
func (w *SpanWrapper) SetAttributes(kv ...attribute.KeyValue) {
	w.span.SetAttributes(kv...)
}

// GetSpanID returns the span ID as a string
func GetSpanIDFromContext(ctx context.Context) string {
	if spanID, ok := ctx.Value(spanIDContextKey).(string); ok {
		return spanID
	}
	return ""
}

func (o *otelWrapper) AddCounter(_ context.Context, counterName string, unit string) error {
	counter, err := o.meter.Int64Counter(counterName, metric.WithUnit(unit))
	if err != nil {
		return err
	}

	o.counters.Store(counterName, counter)
	return nil
}

func AddCounter(ctx context.Context, counterName string, unit string) error {
	return otelInstance.AddCounter(ctx, counterName, unit)
}

func (o *otelWrapper) Count(ctx context.Context, counterName string, incr int64, opts ...metric.AddOption) {
	if counterInterface, ok := o.counters.Load(counterName); ok {
		if counter, ok := counterInterface.(metric.Int64Counter); ok {
			counter.Add(ctx, incr, opts...)
		}
	}
}

func Count(ctx context.Context, counterName string, incr int64, opts ...metric.AddOption) {
	otelInstance.Count(ctx, counterName, incr, opts...)
}

package otel

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"time"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/config/model"
	"github.com/alfin-efendy/helper-go/logger"
	"github.com/alfin-efendy/helper-go/utility"
	"go.opentelemetry.io/otel"
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

var (
	otelInstance Otel
	configs      *model.Config
	isEnabled    bool
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
	counters map[string]metric.Int64Counter
}

func NewOtel(tracer trace.Tracer, meter metric.Meter, counters map[string]metric.Int64Counter) Otel {
	return &otelWrapper{
		tracer:   tracer,
		meter:    meter,
		counters: counters,
	}
}

func Init() {
	ctx := context.Background()
	configs = config.Config

	// Check if OpenTelemetry is enabled
	if !configs.Otel.Trace && !configs.Otel.Metric {
		isEnabled = false
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
	conn, err := initGrpcConn(ctx, configs.Otel.Host)

	// Intialize shutdown hook
	var shutdownHooks []func(context.Context) error
	Shutdown = func(ctx context.Context) error {
		for _, hook := range shutdownHooks {
			err = errors.Join(err, hook(ctx))
		}
		shutdownHooks = nil
		return err
	}

	if configs.Otel.Trace {
		// Initialize trace provider
		err = initTracerProvider(ctx, res, conn)
		if err != nil {
			logger.Fatal(ctx, err, "Failed to initialize OpenTelemetry trace provider")
			return
		}
	}

	if configs.Otel.Metric {
		// Initialize metric provider
		err = initMetricProvider(ctx, res, conn)
		if err != nil {
			logger.Fatal(ctx, err, "Failed to initialize OpenTelemetry metric provider")
			return
		}
	}

	// Set default tracer
	tracer := otel.Tracer(serviceName)

	// Set default meter
	meter := otel.Meter(serviceName)

	// Init default counters
	counters := make(map[string]metric.Int64Counter)
	otelInstance = NewOtel(tracer, meter, counters)
}

func initGrpcConn(ctx context.Context, address string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Fatal(ctx, err, "Failed to create gRPC connection")
		return nil, err
	}

	return conn, nil
}

func initTracerProvider(ctx context.Context, res *resource.Resource, conn *grpc.ClientConn) error {
	conf := configs.Otel

	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithGRPCConn(conn),
		otlptracegrpc.WithTimeout(time.Duration(conf.Timeout)*time.Second),
	)
	if err != nil {
		logger.Fatal(ctx, err, "Failed to create trace exporter")
		return err
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tracerProvider)

	return nil
}

func initMetricProvider(ctx context.Context, res *resource.Resource, conn *grpc.ClientConn) error {
	conf := configs.Otel

	exporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithGRPCConn(conn),
		otlpmetricgrpc.WithTimeout(time.Duration(conf.Timeout)*time.Second),
	)
	if err != nil {
		logger.Fatal(ctx, err, "Failed to create metric exporter")
		return err
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter,
				sdkmetric.WithInterval(3*time.Second),
			),
		),
		sdkmetric.WithResource(res),
	)

	otel.SetMeterProvider(meterProvider)

	return nil
}

func (o *otelWrapper) Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, *SpanWrapper) {
	// Get parent span if any
	sc := trace.SpanContextFromContext(ctx)
	ctx = context.WithValue(ctx, logger.SpanParentIdKey, sc.SpanID().String())

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

		fullName := utility.GetFrame(1).Function
		fullNames := strings.Split(fullName, "/")

		name = fullNames[len(fullNames)-1]
	}

	return otelInstance.Trace(ctx, name, opts...)
}

func (w *SpanWrapper) End(options ...trace.SpanEndOption) {
	w.span.End(options...)
}

func (o *otelWrapper) AddCounter(_ context.Context, counterName string, unit string) error {
	counter, err := o.meter.Int64Counter(counterName, metric.WithUnit(unit))
	if err != nil {
		return err
	}

	o.counters[counterName] = counter
	return nil
}

func AddCounter(ctx context.Context, counterName string, unit string) error {
	return otelInstance.AddCounter(ctx, counterName, unit)
}

func (o *otelWrapper) Count(ctx context.Context, counterName string, incr int64, opts ...metric.AddOption) {
	o.counters[counterName].Add(ctx, incr, opts...)
}

func Count(ctx context.Context, counterName string, incr int64, opts ...metric.AddOption) {
	otelInstance.Count(ctx, counterName, incr, opts...)
}

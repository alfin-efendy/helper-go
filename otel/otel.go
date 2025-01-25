package otel

import (
	"context"
	"time"

	"github.com/alfin-efendy/helper-go/config"
	"github.com/alfin-efendy/helper-go/config/model"
	"github.com/alfin-efendy/helper-go/logger"
	"github.com/inhies/go-bytesize"
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
	tracer      trace.Tracer
	meter       metric.Meter
	counters    map[string]metric.Int64Counter
	configs     *model.Config
	isEnabled   bool
	serviceName string
)

type SpanWrapper struct {
	span trace.Span
}

func Init() {
	ctx := context.Background()
	configs = config.Config

	// Check if OpenTelemetry is enabled
	if configs.Otel.Trace == nil && configs.Otel.Metric == nil {
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

	if configs.Otel.Trace.Exporters.Enable {
		// Initialize trace provider
		err = initTracerProvider(ctx, res)
		if err != nil {
			logger.Fatal(ctx, err, "Failed to initialize OpenTelemetry trace provider")
			return
		}
	}

	if configs.Otel.Metric.Exporters.Enable {
		var metricExporter sdkmetric.Exporter

		conn, cancel, err := initGrpcConn(ctx, configs.Otel.Metric.Exporters.Otlp)
		if err != nil {
			logger.Fatal(ctx, err, "Failed to initialize OpenTelemetry metric exporter")
			return
		}
		defer cancel()

		metricExporter, err = otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
		if err != nil {
			logger.Fatal(ctx, err, "Failed to create OpenTelemetry metric exporter")
			return
		}

		meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)))
		otel.SetMeterProvider(meterProvider)
	}

	// Set default tracer
	tracer = otel.Tracer(serviceName)

	// Set default meter
	meter = otel.Meter(configs.Otel.Metric.InstrumentationName)

	// Init default counters
	counters = make(map[string]metric.Int64Counter)
}

func initGrpcConn(ctx context.Context, exporterConfig *model.OtelExportersOtlp) (*grpc.ClientConn, context.CancelFunc, error) {
	opts := make([]grpc.DialOption, 0)
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	clientMaxReceiveMessageSizeStr := exporterConfig.ClientMaxReceiveMessageSize
	if clientMaxReceiveMessageSizeStr != "" {
		clientMaxReceiveMessageSize, err := bytesize.Parse(clientMaxReceiveMessageSizeStr)
		if err != nil {
			logger.Fatal(ctx, err, "Failed to parse clientMaxReceiveMessageSize")
			return nil, nil, err
		}

		opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(int(clientMaxReceiveMessageSize))))
	}

	// Create GRPC connection with timeout
	ctxCancel, cancel := context.WithTimeout(ctx, time.Duration(exporterConfig.Timeout)*time.Second)
	conn, err := grpc.DialContext(
		ctxCancel,
		exporterConfig.Address,
		opts...,
	)
	return conn, cancel, err
}

func initTracerProvider(ctx context.Context, res *resource.Resource) error {
	conf := configs.Otel.Trace.Exporters.Otlp

	conn, err := grpc.NewClient(
		conf.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Fatal(ctx, err, "Failed to create gRPC connection")
		return err
	}

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

func Trace(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, *SpanWrapper) {
	// Get parent span if any
	sc := trace.SpanContextFromContext(ctx)
	ctx = context.WithValue(ctx, logger.SpanParentIdKey, sc.SpanID().String())

	var span trace.Span
	if isEnabled {
		ctx, span = tracer.Start(ctx, spanName, opts...)
	}

	return ctx, &SpanWrapper{span}
}

func AddCounter(_ context.Context, counterName string, unit string) error {
	counter, err := meter.Int64Counter(counterName, metric.WithUnit(unit))
	if err != nil {
		return err
	}

	counters[counterName] = counter
	return nil
}

func Count(ctx context.Context, counterName string, incr int64, opts ...metric.AddOption) {
	counters[counterName].Add(ctx, incr, opts...)
}

func GetTracer() trace.Tracer {
	return otel.Tracer(serviceName)
}

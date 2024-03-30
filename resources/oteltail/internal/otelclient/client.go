package otelclient

import (
	"context"

	"log/slog"
	"net/url"
	"oteltail/internal/config"
	"oteltail/internal/telemetry/sdklog"
	"oteltail/internal/telemetry/sdklog/otlploggrpc"
	"time"

	"go.opentelemetry.io/otel/log"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

type Client interface {
	SendToOtel(ctx context.Context, b *Batch) error
}

// Implements Client
type OtelClient struct {
	Config       *OtelClientConfig
	LogProcessor *sdklog.LoggerProvider
	Logger       log.Logger
}

type OtelClientConfig struct {
	Url *url.URL
}

const NearlyImmediate = 100 * time.Millisecond

func NewOtelClient(ctx context.Context, cfg *OtelClientConfig, log *slog.Logger) (*OtelClient, error) {

	resources := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(config.GetConfig(ctx).OtelServiceName),
	)

	opts := []otlploggrpc.Option{
		otlploggrpc.WithEndpointURL(cfg.Url.String()),
	}

	if config.GetConfig(ctx).OtelInsecure {
		opts = append(opts,
			otlploggrpc.WithInsecure())
	}

	client := otlploggrpc.NewClient(opts...)
	err := client.Start(ctx)

	lp := sdklog.NewLoggerProvider(resources)

	processor := sdklog.NewBatchLogProcessor(
		client,
		sdklog.WithBatchTimeout(NearlyImmediate),
	)

	lp.RegisterLogProcessor(processor)

	logger := lp.Logger("log/slog")

	return &OtelClient{
		Config:       cfg,
		LogProcessor: lp,
		Logger:       logger,
	}, err
}

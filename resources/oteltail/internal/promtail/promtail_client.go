package promtail

import (
	"context"
	"log/slog"
	"net/url"

	"github.com/grafana/dskit/backoff"
)

type Client interface {
	sendToOtel(ctx context.Context, b *batch) error
}

// Implements Client
type OtelClient struct {
	Config *OtelClientConfig
}

type OtelClientConfig struct {
	Backoff *backoff.Config
	Url     *url.URL
}

func NewOtelClient(cfg *OtelClientConfig, log *slog.Logger) *OtelClient {
	return &OtelClient{
		Config: cfg,
	}
}

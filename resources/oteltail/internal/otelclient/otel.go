package otelclient

import (
	"context"
	"time"

	"github.com/grafana/loki/pkg/logproto"
	"github.com/prometheus/common/model"
	"go.opentelemetry.io/otel/log"

	"oteltail/internal/config"
	"oteltail/internal/logger"
	"oteltail/internal/utils"
)

type LogEntry struct {
	Entry  logproto.Entry
	Labels model.LabelSet
}

type Batch struct {
	Streams   map[string]*Stream
	LineCount int
	Client    Client
}

type Stream struct {
	Labels  model.LabelSet
	Entries []LogEntry
}

type BatchIf interface {
	Add(ctx context.Context, e LogEntry) error
	FlushBatch(ctx context.Context) error
}

func NewBatch(ctx context.Context, oClient Client, entries ...LogEntry) (*Batch, error) {
	b := &Batch{
		Streams: map[string]*Stream{},
		Client:  oClient,
	}

	for _, entry := range entries {
		if err := b.Add(ctx, entry); err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (b *Batch) Add(ctx context.Context, e LogEntry) error {
	labels := utils.LabelsMapToString(e.Labels)
	stream, ok := b.Streams[labels]
	if !ok {
		b.Streams[labels] = &Stream{
			Labels:  e.Labels,
			Entries: []LogEntry{},
		}
		stream = b.Streams[labels]
	}

	stream.Entries = append(stream.Entries, e)
	b.LineCount += 1

	if b.LineCount > config.GetConfig(ctx).LogBatchSize {
		return b.FlushBatch(ctx)
	}

	return nil
}

func (b *Batch) FlushBatch(ctx context.Context) error {
	if b.Client != nil {
		err := b.Client.SendToOtel(ctx, b)
		if err != nil {
			return err
		}
	}
	b.ResetBatch()

	return nil
}

func (b *Batch) ResetBatch() {
	b.Streams = make(map[string]*Stream)
	b.LineCount = 0
}

func (c *OtelClient) SendToOtel(ctx context.Context, b *Batch) error {

	sendlog := logger.GetLogger(ctx)

	sendlog.DebugContext(ctx, "sending to otel")

	//lc, _ := lambdacontext.FromContext(ctx)

	for _, stream := range b.Streams {

		for _, logentry := range stream.Entries {

			var logRec log.Record

			logRec.SetTimestamp(logentry.Entry.Timestamp)
			logRec.SetObservedTimestamp(time.Now())
			logRec.SetBody(log.StringValue(string(logentry.Entry.Line)))
			logRec.AddAttributes(logKVs(logentry.Labels)...)

			c.Logger.Emit(ctx, logRec)
		}
	}

	return nil
}

func logKVs(ls model.LabelSet) []log.KeyValue {
	res := make([]log.KeyValue, 0, len(ls))
	for l, v := range ls {
		res = append(res, log.KeyValue{
			Key:   string(l),
			Value: log.StringValue(string(v)),
		})

	}
	return res
}

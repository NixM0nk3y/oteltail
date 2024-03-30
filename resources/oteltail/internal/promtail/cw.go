package promtail

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/grafana/loki/pkg/logproto"
	"github.com/prometheus/common/model"

	"oteltail/internal/config"
	"oteltail/internal/otelclient"
	"oteltail/internal/utils"
)

func parseCWEvent(ctx context.Context, b *otelclient.Batch, ev *events.CloudwatchLogsEvent) error {
	data, err := ev.AWSLogs.Parse()
	if err != nil {
		return err
	}

	labels := model.LabelSet{
		model.LabelName("__aws_log_type"):             model.LabelValue("cloudwatch"),
		model.LabelName("__aws_cloudwatch_log_group"): model.LabelValue(data.LogGroup),
		model.LabelName("__aws_cloudwatch_owner"):     model.LabelValue(data.Owner),
	}

	if config.GetConfig(ctx).KeepStream {
		labels[model.LabelName("__aws_cloudwatch_log_stream")] = model.LabelValue(data.LogStream)
	}

	labels = utils.ApplyResourceAttributes(ctx, labels)

	for _, event := range data.LogEvents {
		timestamp := time.UnixMilli(event.Timestamp)

		if err := b.Add(ctx, otelclient.LogEntry{Labels: labels, Entry: logproto.Entry{
			Line:      event.Message,
			Timestamp: timestamp,
		}}); err != nil {
			return err
		}
	}

	return nil
}

func ProcessCWEvent(ctx context.Context, ev *events.CloudwatchLogsEvent, oClient otelclient.Client) error {
	batch, err := otelclient.NewBatch(ctx, oClient)
	if err != nil {
		return err
	}

	err = parseCWEvent(ctx, batch, ev)
	if err != nil {
		return fmt.Errorf("error parsing log event: %s", err)
	}

	err = oClient.SendToOtel(ctx, batch)
	if err != nil {
		return err
	}

	return nil
}

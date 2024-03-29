package promtail

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/grafana/loki/pkg/logproto"
	"github.com/prometheus/common/model"

	"oteltail/internal/config"
	"oteltail/internal/utils"
)

func parseCWEvent(ctx context.Context, b *batch, ev *events.CloudwatchLogsEvent) error {
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

	labels = utils.ApplyLabels(ctx, labels)

	for _, event := range data.LogEvents {
		timestamp := time.UnixMilli(event.Timestamp)

		if err := b.add(ctx, entry{labels, logproto.Entry{
			Line:      event.Message,
			Timestamp: timestamp,
		}}); err != nil {
			return err
		}
	}

	return nil
}

func ProcessCWEvent(ctx context.Context, ev *events.CloudwatchLogsEvent, pClient Client) error {
	batch, err := newBatch(ctx, pClient)
	if err != nil {
		return err
	}

	err = parseCWEvent(ctx, batch, ev)
	if err != nil {
		return fmt.Errorf("error parsing log event: %s", err)
	}

	err = pClient.sendToOtel(ctx, batch)
	if err != nil {
		return err
	}

	return nil
}

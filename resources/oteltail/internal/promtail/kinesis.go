package promtail

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/grafana/loki/pkg/logproto"
	"github.com/prometheus/common/model"

	"oteltail/internal/config"
	"oteltail/internal/otelclient"
	"oteltail/internal/utils"
)

func parseKinesisEvent(ctx context.Context, b otelclient.BatchIf, ev *events.KinesisEvent) error {
	if ev == nil {
		return nil
	}

	for _, record := range ev.Records {
		timestamp := time.Unix(record.Kinesis.ApproximateArrivalTimestamp.Unix(), 0)

		labels := model.LabelSet{
			model.LabelName("__aws_log_type"):                 model.LabelValue("kinesis"),
			model.LabelName("__aws_kinesis_event_source_arn"): model.LabelValue(record.EventSourceArn),
		}

		labels = utils.ApplyResourceAttributes(ctx, labels)

		// Check if the data is gzipped by inspecting the 'data' field
		if isGzipped(record.Kinesis.Data) {
			uncompressedData, err := ungzipData(record.Kinesis.Data)
			if err != nil {
				return err
			}
			b.Add(ctx, otelclient.LogEntry{Labels: labels, Entry: logproto.Entry{
				Line:      string(uncompressedData),
				Timestamp: timestamp,
			}})
		} else {
			b.Add(ctx, otelclient.LogEntry{Labels: labels, Entry: logproto.Entry{
				Line:      string(record.Kinesis.Data),
				Timestamp: timestamp,
			}})
		}
	}

	return nil
}

func cwParse(rawData []byte, ev *events.CloudwatchLogsData) (err error) {

	rdata := bytes.NewReader(rawData)
	r, err := gzip.NewReader(rdata)
	if err != nil {
		return err
	}
	uncompressedData, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(uncompressedData, ev); err != nil {
		fmt.Printf("error: %v", err)
	}

	return err
}

func parseKinesisCwEvent(ctx context.Context, b otelclient.BatchIf, ev *events.KinesisEvent) error {
	if ev == nil {
		return nil
	}

	for _, record := range ev.Records {
		var cwEvents events.CloudwatchLogsData

		err := cwParse(record.Kinesis.Data, &cwEvents)
		if err != nil {
			return err
		}

		labels := model.LabelSet{
			model.LabelName("__aws_log_type"):             model.LabelValue("cloudwatch"),
			model.LabelName("__aws_cloudwatch_log_group"): model.LabelValue(cwEvents.LogGroup),
			model.LabelName("__aws_cloudwatch_owner"):     model.LabelValue(cwEvents.Owner),
		}

		if config.GetConfig(ctx).KeepStream {
			labels[model.LabelName("__aws_cloudwatch_log_stream")] = model.LabelValue(cwEvents.LogStream)
		}

		labels = utils.ApplyResourceAttributes(ctx, labels)

		for _, event := range cwEvents.LogEvents {
			timestamp := time.UnixMilli(event.Timestamp)

			if err := b.Add(ctx, otelclient.LogEntry{Labels: labels, Entry: logproto.Entry{
				Line:      event.Message,
				Timestamp: timestamp,
			}}); err != nil {
				return err
			}
		}
	}

	return nil
}

func ProcessKinesisEvent(ctx context.Context, ev *events.KinesisEvent, oClient otelclient.Client) error {
	batch, _ := otelclient.NewBatch(ctx, oClient)

	err := parseKinesisEvent(ctx, batch, ev)
	if err != nil {
		return err
	}

	err = oClient.SendToOtel(ctx, batch)
	if err != nil {
		return err
	}
	return nil
}

func ProcessKinesisCwEvent(ctx context.Context, ev *events.KinesisEvent, oClient otelclient.Client) error {
	batch, _ := otelclient.NewBatch(ctx, oClient)

	err := parseKinesisCwEvent(ctx, batch, ev)
	if err != nil {
		return err
	}

	err = oClient.SendToOtel(ctx, batch)
	if err != nil {
		return err
	}
	return nil
}

// isGzipped checks if the input data is gzipped
func isGzipped(data []byte) bool {
	return len(data) >= 2 && data[0] == 0x1F && data[1] == 0x8B
}

// unzipData decompress the gzipped data
func ungzipData(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

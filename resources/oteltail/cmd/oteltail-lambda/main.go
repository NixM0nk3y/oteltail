package main

import (
	"context"
	"log/slog"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/lambdacontext"

	"oteltail/internal/config"
	"oteltail/internal/logger"
	"oteltail/internal/otelclient"
	"oteltail/internal/promtail"
	"oteltail/internal/utils"
)

func handler(ctx context.Context, ev map[string]interface{}) error {

	log := logger.GetLogger(ctx)

	lc, _ := lambdacontext.FromContext(ctx)

	lctx := logger.AppendCtx(ctx, slog.String("request_id", lc.AwsRequestID))

	vctx := config.ReadEnvConfig(lctx, "OTELTAIL")

	oClient, err := otelclient.NewOtelClient(vctx, &otelclient.OtelClientConfig{
		Url: config.GetConfig(vctx).OtelExporterEndpoint.URL,
	}, log)
	if err != nil {
		log.ErrorContext(vctx, "error initiating otel client", "error", err)
		return err
	}

	event, err := utils.CheckEventType(ev)
	if err != nil {
		log.ErrorContext(vctx, "invalid event", "error", ev)
		return err
	}

	switch evt := event.(type) {
	case *events.CloudWatchEvent:
		err = promtail.ProcessEventBridgeEvent(vctx, evt, oClient, promtail.ProcessS3Event)
	case *events.S3Event:
		err = promtail.ProcessS3Event(vctx, evt, oClient)
	case *events.CloudwatchLogsEvent:
		err = promtail.ProcessCWEvent(vctx, evt, oClient)
	case *events.KinesisEvent:
		if config.GetConfig(vctx).ParseKinesisCwLogs {
			err = promtail.ProcessKinesisCwEvent(vctx, evt, oClient)
		} else {
			err = promtail.ProcessKinesisEvent(vctx, evt, oClient)
		}
	case *events.SQSEvent:
		err = promtail.ProcessSQSEvent(vctx, evt, handler)
	case *events.SNSEvent:
		err = promtail.ProcessSNSEvent(vctx, evt, handler)
	// When setting up S3 Notification on a bucket, a test event is first sent, see: https://docs.aws.amazon.com/AmazonS3/latest/userguide/notification-content-structure.html
	case *events.S3TestEvent:
		return nil
	}

	if err != nil {
		log.ErrorContext(vctx, "error processing event", "error", err)
		return err
	}

	oClient.LogProcessor.Shutdown(vctx)

	log.InfoContext(vctx, "processing complete")

	return err
}

func main() {
	lambda.Start(handler)
}

package sdklog

import (
	"context"
	"oteltail/internal/utils"

	"github.com/aws/aws-lambda-go/lambdacontext"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
)

var _ log.Logger = &logger{}

type logger struct {
	embedded.Logger

	provider             *LoggerProvider
	resource             *resource.Resource
	instrumentationScope instrumentation.Scope
}

func (l logger) Emit(ctx context.Context, r log.Record) {

	lc, _ := lambdacontext.FromContext(ctx)

	traceid, _ := utils.ParseTraceID(lc.AwsRequestID)

	log := &LogData{
		Record:   r,
		TraceID:  traceid,
		Resource: l.resource,
	}

	for _, proc := range l.provider.getLogProcessors() {
		proc.OnEmit(ctx, log)
	}
}

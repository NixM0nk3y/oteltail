package config

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"github.com/mdobak/go-xerrors"
	"github.com/prometheus/common/model"

	"oteltail/internal/logger"
)

const (
	invalidExtraLabelsError = "invalid value for environment variable EXTRA_LABELS. Expected a comma separated list with an even number of entries. "
)

type WriteAddress struct {
	URL *url.URL
}

func (w *WriteAddress) Decode(value string) error {
	url, err := url.Parse(value)
	if err != nil {
		panic(err)
	}
	*w = WriteAddress{
		URL: url,
	}
	return nil
}

// Configuration is
type Configuration struct {
	OtelExporterEndpoint  WriteAddress `envconfig:"OTEL_EXPORTER_OTLP_ENDPOINT" required:"true"`
	OtelInsecure          bool         `envconfig:"OTEL_EXPORTER_INSECURE"`
	OtelServiceName       string       `envconfig:"OTEL_SERVICE_NAME" required:"true"`
	ResourceAttributesRaw string       `envconfig:"RESOURCE_ATTRIBUTES"`
	DropAttributesRaw     string       `envconfig:"DROP_ATTRIBUTES"`
	KeepStream            bool         `envconfig:"KEEP_STREAM"`
	LogBatchSize          int          `envconfig:"LOG_BATCH_SIZE" default:"5"`
	PrintLogLine          bool         `envconfig:"PRINT_LOG_LINES"`
	ParseKinesisCwLogs    bool         `envconfig:"PARSE_KINESIS_CLOUDWATCH_LOGS"`
	ResourceAttributes    model.LabelSet
	DropAttributes        []model.LabelName
}

var lambdaConfig Configuration

// The key type is unexported to prevent collisions with context keys defined in
// other packages.
type contextKey string

func (c contextKey) String() string {
	return "context key " + string(c)
}

var (
	contextKeyConfig = contextKey("config")
)

// ReadEnvConfig is
func ReadEnvConfig(ctx context.Context, namespace string) context.Context {

	log := logger.GetLogger(ctx)

	err := envconfig.Process(namespace, &lambdaConfig)

	if err != nil {
		log.ErrorContext(ctx, "unable to process environment", "error", err)
		panic(err)
	}

	log.InfoContext(ctx, "config parse", "OtelExporterEndpoint", lambdaConfig.OtelExporterEndpoint.URL.String())

	lambdaConfig.ResourceAttributes, err = parseResourceAttributes(ctx, lambdaConfig.ResourceAttributesRaw)

	if err != nil {
		log.ErrorContext(ctx, "unable to parse extra labels", "error", err)
		panic(err)
	}

	lambdaConfig.DropAttributes, err = parseDropAttributes()
	if err != nil {
		log.ErrorContext(ctx, "unable to parse drop labels", "error", err)
		panic(err)
	}

	return context.WithValue(ctx, contextKeyConfig, &lambdaConfig)
}

// GetConfig is
func GetConfig(ctx context.Context) *Configuration {

	log := logger.GetLogger(ctx)

	econfig, ok := ctx.Value(contextKeyConfig).(*Configuration)

	if !ok {
		log.ErrorContext(ctx, "unable to retrieve config")
		panic(xerrors.New("unable to retrieve config"))
	}

	return econfig
}

func parseResourceAttributes(ctx context.Context, extraResourceAttributesRaw string) (model.LabelSet, error) {

	log := logger.GetLogger(ctx)

	var extractedResourceAttributes = model.LabelSet{}
	extraResourceAttributeSplit := strings.Split(extraResourceAttributesRaw, ",")

	if len(extraResourceAttributesRaw) < 1 {
		return extractedResourceAttributes, nil
	}

	if len(extraResourceAttributeSplit)%2 != 0 {
		return nil, fmt.Errorf(invalidExtraLabelsError)
	}
	for i := 0; i < len(extraResourceAttributeSplit); i += 2 {
		extractedResourceAttributes[model.LabelName(extraResourceAttributeSplit[i])] = model.LabelValue(extraResourceAttributeSplit[i+1])
	}
	err := extractedResourceAttributes.Validate()
	if err != nil {
		return nil, err
	}
	log.DebugContext(ctx, "extra resource attributes", "ResourceAttributes", extractedResourceAttributes)
	return extractedResourceAttributes, nil
}

func parseDropAttributes() ([]model.LabelName, error) {
	var result []model.LabelName

	if lambdaConfig.DropAttributesRaw != "" {
		dropAttributesRaw := strings.Split(lambdaConfig.DropAttributesRaw, ",")
		for _, dropAttributeRaw := range dropAttributesRaw {
			dropAttribute := model.LabelName(dropAttributeRaw)
			if !dropAttribute.IsValid() {
				return []model.LabelName{}, fmt.Errorf("invalid attribute name %s", dropAttributeRaw)
			}
			result = append(result, dropAttribute)
		}
	}

	return result, nil
}

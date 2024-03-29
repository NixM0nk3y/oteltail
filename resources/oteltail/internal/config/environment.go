package config

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/common/model"
	"golang.org/x/xerrors"

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
	WriteAddress          WriteAddress `envconfig:"WRITE_ADDRESS" required:"true"`
	Username              string       `envconfig:"USERNAME"`
	Password              string       `envconfig:"PASSWORD"`
	ExtraLabelsRaw        string       `envconfig:"EXTRA_LABELS"`
	OmitExtraLabelsPrefix bool         `envconfig:"OMIT_EXTRA_LABELS_PREFIX"`
	DropLabelsRaw         string       `envconfig:"DROP_LABELS"`
	Tenant                string       `envconfig:"TENANT"`
	Environment           string       `envconfig:"ENVIRONMENT"`
	Service               string       `envconfig:"SERVER"`
	BearerToken           string       `envconfig:"BEARER_TOKEN"`
	KeepStream            bool         `envconfig:"KEEP_STREAM"`
	BatchSize             int          `envconfig:"BATCH_SIZE" default:"131072"`
	SkipTlsVerify         bool         `envconfig:"SKIP_TLS_VERIFY"`
	PrintLogLine          bool         `envconfig:"PRINT_LOG_LINES"`
	ParseKinesisCwLogs    bool         `envconfig:"PARSE_KINESIS_CLOUDWATCH_LOGS"`
	ExtraLabels           model.LabelSet
	DropLabels            []model.LabelName
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

	log.InfoContext(ctx, "config parse", "write_address", lambdaConfig.WriteAddress.URL.String())

	lambdaConfig.ExtraLabels, err = parseExtraLabels(ctx, lambdaConfig.ExtraLabelsRaw, lambdaConfig.OmitExtraLabelsPrefix)

	if err != nil {
		log.ErrorContext(ctx, "unable to parse extra labels", "error", err)
		panic(err)
	}

	lambdaConfig.DropLabels, err = parseDropLabels()
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

func parseExtraLabels(ctx context.Context, extraLabelsRaw string, omitPrefix bool) (model.LabelSet, error) {

	log := logger.GetLogger(ctx)

	prefix := "__extra_"
	if omitPrefix {
		prefix = ""
	}
	var extractedLabels = model.LabelSet{}
	extraLabelsSplit := strings.Split(extraLabelsRaw, ",")

	if len(extraLabelsRaw) < 1 {
		return extractedLabels, nil
	}

	if len(extraLabelsSplit)%2 != 0 {
		return nil, fmt.Errorf(invalidExtraLabelsError)
	}
	for i := 0; i < len(extraLabelsSplit); i += 2 {
		extractedLabels[model.LabelName(prefix+extraLabelsSplit[i])] = model.LabelValue(extraLabelsSplit[i+1])
	}
	err := extractedLabels.Validate()
	if err != nil {
		return nil, err
	}
	log.DebugContext(ctx, "extra labels", "labels", extractedLabels)
	return extractedLabels, nil
}

func parseDropLabels() ([]model.LabelName, error) {
	var result []model.LabelName

	if lambdaConfig.DropLabelsRaw != "" {
		dropLabelsRawSplit := strings.Split(lambdaConfig.DropLabelsRaw, ",")
		for _, dropLabelRaw := range dropLabelsRawSplit {
			dropLabel := model.LabelName(dropLabelRaw)
			if !dropLabel.IsValid() {
				return []model.LabelName{}, fmt.Errorf("invalid label name %s", dropLabelRaw)
			}
			result = append(result, dropLabel)
		}
	}

	return result, nil
}

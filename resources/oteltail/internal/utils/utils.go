package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/prometheus/common/model"

	"oteltail/internal/config"
)

func ApplyLabels(ctx context.Context, labels model.LabelSet) model.LabelSet {
	finalLabels := labels.Merge(config.GetConfig(ctx).ExtraLabels)

	for _, dropLabel := range config.GetConfig(ctx).DropLabels {
		delete(finalLabels, dropLabel)
	}

	return finalLabels
}

func CheckEventType(ev map[string]interface{}) (interface{}, error) {
	var s3Event events.S3Event
	var s3TestEvent events.S3TestEvent
	var cwEvent events.CloudwatchLogsEvent
	var kinesisEvent events.KinesisEvent
	var sqsEvent events.SQSEvent
	var snsEvent events.SNSEvent
	var eventBridgeEvent events.CloudWatchEvent

	types := [...]interface{}{&s3Event, &s3TestEvent, &cwEvent, &kinesisEvent, &sqsEvent, &snsEvent, &eventBridgeEvent}

	j, _ := json.Marshal(ev)
	reader := strings.NewReader(string(j))
	d := json.NewDecoder(reader)
	d.DisallowUnknownFields()

	for _, t := range types {
		err := d.Decode(t)

		if err == nil {
			return t, nil
		}

		reader.Seek(0, 0)
	}

	return nil, fmt.Errorf("unknown event type!")
}

// getUnixSecNsec returns the Unix time seconds and nanoseconds in the string s.
// It assumes that the first 10 digits of the parsed int is the Unix time in seconds and the rest is the nanoseconds part.
// This assumption will hold until 2286-11-20 17:46:40 UTC, so it's a safe assumption.
// It also makes use of the fact that the log10 of a number in base 10 is its number of digits - 1.
// It returns early if the fractional seconds is 0 because getting the log10 of 0 results in -Inf.
// For example, given a string 1234567890123:
//
//	iLog10 = 12  // the parsed int is 13 digits long
//	multiplier = 0.001  // to get the seconds part it must be divided by 1000
//	sec = 1234567890123 * 0.001 = 1234567890  // this is the seconds part of the Unix time
//	fractionalSec = 123  // the rest of the parsed int
//	fractionalSecLog10 = 2  // it is 3 digits long
//	multiplier = 1000000  // nano is 10^-9, so the nanoseconds part is 9 digits long
//	nsec = 123000000  // this is the nanoseconds part of the Unix time
func GetUnixSecNsec(s string) (sec int64, nsec int64, err error) {
	const (
		UNIX_SEC_LOG10     = 9
		UNIX_NANOSEC_LOG10 = 8
	)

	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return sec, nsec, err
	}

	iLog10 := int(math.Log10(float64(i)))
	multiplier := math.Pow10(UNIX_SEC_LOG10 - iLog10)
	sec = int64(float64(i) * multiplier)

	fractionalSec := float64(i % sec)
	if fractionalSec == 0 {
		return sec, 0, err
	}

	fractionalSecLog10 := int(math.Log10(fractionalSec))
	multiplier = math.Pow10(UNIX_NANOSEC_LOG10 - fractionalSecLog10)
	nsec = int64(fractionalSec * multiplier)

	return sec, nsec, err
}

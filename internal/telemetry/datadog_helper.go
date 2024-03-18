package telemetry

import (
	"fmt"
	"go-workers/internal/util"
	"strings"

	"github.com/DataDog/datadog-go/v5/statsd"
)

var _dataDogClient *statsd.Client
var ddEnabled string

func init() {
	var err error

	ddEnabled = util.ReadParameter("DATADOG_ENABLED")

	if strings.ToLower(ddEnabled) == "true" {
		_dataDogClient, err = statsd.New(
			util.ReadParameter("DATADOG_HOST_AGENT"),
			statsd.WithTags([]string{
				fmt.Sprintf("env:%s", util.ReadParameter("DATADOG_ENVIRONMENT")),
				fmt.Sprintf("service:%s", util.ReadParameter("DATADOG_SERVICE_NAME")),
			}))
	}

	if err != nil {
		util.Sugar.Infof(err.Error())
	}
}

func AddTickHistogram(
	metricName string,
	value float64,
	tags []string,
) {
	ddEnabled = util.ReadParameter("DATADOG_ENABLED")
	if strings.ToLower(ddEnabled) == "true" {
		err := _dataDogClient.Histogram(metricName, value, tags, 1)
		if err != nil {
			util.Sugar.Infof(err.Error())
		}
	}
}

func TickErrorCount(
	metricName string,
	value int64,
	tags []string,
) {

	ddEnabled = util.ReadParameter("DATADOG_ENABLED")
	if strings.ToLower(ddEnabled) == "true" {
		err := _dataDogClient.Count(metricName, value, tags, 1)
		if err != nil {
			util.Sugar.Infof(err.Error())
		}
	}
}

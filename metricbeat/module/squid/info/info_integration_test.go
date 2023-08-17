package info

import (
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"
)

func TestFetch(t *testing.T) {
	// service := compose.EnsureUp(t, "squid")

	f := mbtest.NewReportingMetricSetV2Error(t, map[string]interface{}{"module": "squid", "metricsets": []string{"info"}, "hosts": []string{"localhost"}})
	events, errs := mbtest.ReportingFetchV2Error(f)

	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}

	t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
		events[0].BeatEvent("squid", "info").Fields.StringToPrint())

}

package node_stats

import (
	"net/url"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/opendistro"
)

func init() {
	mb.Registry.MustAddMetricSet(opendistro.ModuleName, "node_stats", New,
		mb.WithHostParser(opendistro.HostParser),
		mb.DefaultMetricSet(),
		mb.WithNamespace("opendistro.node.stats"),
	)

}

const (
	nodeLocalStatsPath = "/_nodes/_local/stats"
)

type MetricSet struct {
	*opendistro.MetricSet
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	cfgwarn.Beta("The opendistro node_stats metricset is beta.")

	ms, err := opendistro.NewMetricSet(base, "") // servicePath will be set in Fetch()
	if err != nil {
		return nil, err
	}
	return &MetricSet{MetricSet: ms}, nil
}

func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	if err := m.updateServiceURI(); err != nil {
		return err
	}

	content, err := m.HTTP.FetchContent()
	if err != nil {
		return err
	}

	info, err := opendistro.GetInfo(m.HTTP, m.GetServiceURI())
	if err != nil {
		return err
	}
	err = eventsMapping(report, m, *info, content)
	if err != nil {
		m.Logger().Error(err)
		return nil
	}
	return nil

}

func (m *MetricSet) updateServiceURI() error {
	u, err := getServiceURI(m.GetURI())
	if err != nil {
		return err
	}

	m.HTTP.SetURI(u)
	return nil

}

func getServiceURI(currURI string) (string, error) {
	u, err := url.Parse(currURI)
	if err != nil {
		return "", err
	}

	u.Path = nodeLocalStatsPath

	return u.String(), nil
}

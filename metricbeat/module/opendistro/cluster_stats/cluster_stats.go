package cluster_stats

import (
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/opendistro"
)

func init() {
	mb.Registry.MustAddMetricSet(opendistro.ModuleName, "cluster_stats", New,
		mb.WithHostParser(opendistro.HostParser),
		mb.WithNamespace("opendistro.cluster.stats"),
	)
}

const (
	clusterStatsPath = "/_cluster/stats"
)

type MetricSet struct {
	*opendistro.MetricSet
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The opendistro cluster_stats metricset is beta.")
	ms, err := opendistro.NewMetricSet(base, clusterStatsPath)
	if err != nil {
		return nil, err
	}
	return &MetricSet{MetricSet: ms}, nil
}

func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	shouldSkip, err := m.ShouldSkipFetch()
	if err != nil {
		return err
	}
	if shouldSkip {
		return nil
	}

	content, err := m.HTTP.FetchContent()
	if err != nil {
		return err
	}
	info, err := opendistro.GetInfo(m.HTTP, m.HostData().SanitizedURI+clusterStatsPath)
	if err != nil {
		return err
	}
	err = eventMapping(report, m, *info, content)
	if err != nil {
		m.Logger().Error(err)
		return nil
	}
	return nil
}

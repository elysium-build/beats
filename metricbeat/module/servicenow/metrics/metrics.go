package metrics

import (
	"encoding/json"
	"encoding/xml"
	"net/url"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/metricbeat/helper"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"

	defaultPath = "/xmlstats.do"
)

var (
	debugf = logp.MakeDebug("ServiceNow-metrics")

	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
		PathConfigKey: "path",
	}.Build()
)

func init() {
	mb.Registry.MustAddMetricSet("servicenow", "metrics", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

type MetricSet struct {
	mb.BaseMetricSet
	http *helper.HTTP
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The servicenow metrics metricset is beta.")

	config := struct{}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		http:          http,
	}, nil
}

func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	content, err := fetchPath(m.http, m.HostData().SanitizedURI, "/xmlstats.do", "include=servlet,database,semaphores,memory")
	if err != nil {
		return err
	}

	type ServletMetric struct {
		Count                    int    `xml:"count,attr"`
		Max                      string `xml:"max,attr"`
		Mean                     string `xml:"mean,attr"`
		Median                   string `xml:"median,attr"`
		Ninetypercent            string `xml:"ninetypercent,attr"`
		NinetypercentTrimmedMean string `xml:"ninetypercentTrimmedMean,attr"`
	}

	type Semaphore struct {
		Debugger_active  bool   `xml:"debugger_active,attr"`
		Debugger_enabled bool   `xml:"debugger_enabled,attr"`
		Processor        string `xml:"processor,attr"`
		Age              int    `xml:"age,attr"`
	}

	type Semaphores struct {
		Available int         `xml:"available,attr"`
		Semaphore []Semaphore `xml:"semaphore"`
	}

	type DatabasePool struct {
		Name      string `xml:"name,attr"`
		Status    string `xml:"status"`
		Available int    `xml:"available"`
	}

	type SNCStats struct {
		XMLName                 xml.Name       `xml:"xmlstats"`
		Servlet_Started         string         `xml:"servlet.started"`
		Threads_CPU_One         ServletMetric  `xml:"servlet.metrics>threads_cpu>one"`
		Threads_DB_One          ServletMetric  `xml:"servlet.metrics>threads_db>one"`
		Threads_Network_One     ServletMetric  `xml:"servlet.metrics>threads_network>one"`
		Threads_Concurrency_One ServletMetric  `xml:"servlet.metrics>threads_concurrency>one"`
		Client_Transactions_One ServletMetric  `xml:"servlet.metrics>client_transactions>one"`
		SQL_Response_One        ServletMetric  `xml:"servlet.metrics>sql_response>one"`
		Garbage_Collection_One  ServletMetric  `xml:"servlet.metrics>garbage_collection>one"`
		Semaphore_Waiters_One   ServletMetric  `xml:"servlet.metrics>semaphore_waiters>one"`
		Semaphores              []Semaphores   `xml:"semaphores"`
		DBPools                 []DatabasePool `xml:"db.pools>pool"`
		System_Memory_Pct_Free  float64        `xml:"system.memory.pct.free"`
	}

	var snowStats SNCStats
	if err := xml.Unmarshal(content, &snowStats); err != nil {
		return err
	}

	var glidPool DatabasePool
	for i := range snowStats.DBPools {
		if snowStats.DBPools[i].Name == "glide" {
			glidPool = snowStats.DBPools[i]
			break
		}
	}

	content, err = fetchPath(m.http, m.HostData().SanitizedURI, "/api/now/table/v_user_session", "sysparm_query=user%21%3Dadmin&sysparm_query=user%21%3Dintegration")
	if err != nil {
		return err
	}

	// var data map[string]interface{}

	type SNCUser struct {
		User   string `json:"user"`
		Active string `json:"active"`
	}
	type SNCUsers struct {
		User []SNCUser `json:"result"`
	}

	var sncUsers SNCUsers
	err = json.Unmarshal(content, &sncUsers)
	if err != nil {
		return err
	}

	var numOfConcurrentTransactions int = 0

	for _, user := range sncUsers.User {
		if user.Active == "true" {
			numOfConcurrentTransactions += 1
		}
	}

	// m.Logger().Debugf("trying to fetch %v", sncUsers)
	// m.Logger().Debugf("trying to fetch len %d", len(sncUsers.User))

	report.Event(mb.Event{
		MetricSetFields: common.MapStr{
			"threadsCpu":                  snowStats.Threads_CPU_One.Mean,
			"threadsDb":                   snowStats.Threads_DB_One.Mean,
			"threadsNetwork":              snowStats.Threads_Network_One.Mean,
			"threadsConcurrency":          snowStats.Threads_Concurrency_One.Mean,
			"transactionCount":            snowStats.Client_Transactions_One.Count,
			"dbResponseTime":              snowStats.SQL_Response_One.Mean,
			"garbageCollection":           snowStats.Garbage_Collection_One.Mean,
			"semaphoreWait":               snowStats.Semaphore_Waiters_One.Max,
			"avaliablePools":              glidPool.Available,
			"avaliableSemaphores":         snowStats.Semaphores[0].Available,
			"heapSpace":                   100 - snowStats.System_Memory_Pct_Free,
			"numOfUsers":                  len(sncUsers.User),
			"numOfConcurrentTransactions": numOfConcurrentTransactions,
		},
	})

	return nil
}

func fetchPath(http *helper.HTTP, uri, path string, query string) ([]byte, error) {
	defer http.SetURI(uri)

	// Parses the uri to replace the path
	u, _ := url.Parse(uri)
	u.Path = path
	u.RawQuery = query

	// Http helper includes the HostData with username and password
	http.SetURI(u.String())
	return http.FetchContent()
}

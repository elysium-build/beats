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

	type UnitMetric struct {
		Count                    int    `xml:"count,attr"`
		Max                      string `xml:"max,attr"`
		Mean                     string `xml:"mean,attr"`
		Median                   string `xml:"median,attr"`
		Min                      string `xml:"min,attr"`
		Ninetypercent            string `xml:"ninetypercent,attr"`
		NinetypercentTrimmedMean string `xml:"ninetypercentTrimmedMean,attr"`
	}

	type ServletMetric struct {
		XMLName xml.Name `xml:"servlet.metrics"`

		Transactions_One     []UnitMetric `xml:"transactions>one"`
		Transactions_Five    []UnitMetric `xml:"transactions>five"`
		Transactions_Fifteen []UnitMetric `xml:"transactions>fifteen"`
		Transactions_Hour    []UnitMetric `xml:"transactions>hour"`
		Transactions_Daily   []UnitMetric `xml:"transactions>daily"`

		Client_transactions_One     []UnitMetric `xml:"client_transactions>one"`
		Client_transactions_Five    []UnitMetric `xml:"client_transactions>five"`
		Client_transactions_Fifteen []UnitMetric `xml:"client_transactions>fifteen"`
		Client_transactions_Hour    []UnitMetric `xml:"client_transactions>hour"`
		Client_transactions_Daily   []UnitMetric `xml:"client_transactions>daily"`

		Semaphore_amb_send_response_time_One     []UnitMetric `xml:"semaphore_amb_send_response_time>one"`
		Semaphore_amb_send_response_time_Five    []UnitMetric `xml:"semaphore_amb_send_response_time>five"`
		Semaphore_amb_send_response_time_Fifteen []UnitMetric `xml:"semaphore_amb_send_response_time>fifteen"`
		Semaphore_amb_send_response_time_Hour    []UnitMetric `xml:"semaphore_amb_send_response_time>hour"`
		Semaphore_amb_send_response_time_Daily   []UnitMetric `xml:"semaphore_amb_send_response_time>daily"`

		Semaphore_amb_receive_response_time_One     []UnitMetric `xml:"semaphore_amb_receive_response_time>one"`
		Semaphore_amb_receive_response_time_Five    []UnitMetric `xml:"semaphore_amb_receive_response_time>five"`
		Semaphore_amb_receive_response_time_Fifteen []UnitMetric `xml:"semaphore_amb_receive_response_time>fifteen"`
		Semaphore_amb_receive_response_time_Hour    []UnitMetric `xml:"semaphore_amb_receive_response_time>hour"`
		Semaphore_amb_receive_response_time_Daily   []UnitMetric `xml:"semaphore_amb_receive_response_time>daily"`

		Semaphore_api_int_response_time_One     []UnitMetric `xml:"semaphore_api_int_response_time>one"`
		Semaphore_api_int_response_time_Five    []UnitMetric `xml:"semaphore_api_int_response_time>five"`
		Semaphore_api_int_response_time_Fifteen []UnitMetric `xml:"semaphore_api_int_response_time>fifteen"`
		Semaphore_api_int_response_time_Hour    []UnitMetric `xml:"semaphore_api_int_response_time>hour"`
		Semaphore_api_int_response_time_Daily   []UnitMetric `xml:"semaphore_api_int_response_time>daily"`

		Semaphore_default_response_time_One     []UnitMetric `xml:"semaphore_default_response_time>one"`
		Semaphore_default_response_time_Five    []UnitMetric `xml:"semaphore_default_response_time>five"`
		Semaphore_default_response_time_Fifteen []UnitMetric `xml:"semaphore_default_response_time>fifteen"`
		Semaphore_default_response_time_Hour    []UnitMetric `xml:"semaphore_default_response_time>hour"`
		Semaphore_default_response_time_Daily   []UnitMetric `xml:"semaphore_default_response_time>daily"`

		Semaphore_debug_response_time_One     []UnitMetric `xml:"semaphore_debug_response_time>one"`
		Semaphore_debug_response_time_Five    []UnitMetric `xml:"semaphore_debug_response_time>five"`
		Semaphore_debug_response_time_Fifteen []UnitMetric `xml:"semaphore_debug_response_time>fifteen"`
		Semaphore_debug_response_time_Hour    []UnitMetric `xml:"semaphore_debug_response_time>hour"`
		Semaphore_debug_response_time_Daily   []UnitMetric `xml:"semaphore_debug_response_time>daily"`

		Semaphore_presence_response_time_One     []UnitMetric `xml:"semaphore_presence_response_time>one"`
		Semaphore_presence_response_time_Five    []UnitMetric `xml:"semaphore_presence_response_time>five"`
		Semaphore_presence_response_time_Fifteen []UnitMetric `xml:"semaphore_presence_response_time>fifteen"`
		Semaphore_presence_response_time_Hour    []UnitMetric `xml:"semaphore_presence_response_time>hour"`
		Semaphore_presence_response_time_Daily   []UnitMetric `xml:"semaphore_presence_response_time>daily"`

		Memory_One     []UnitMetric `xml:"memory>one"`
		Memory_Five    []UnitMetric `xml:"memory>five"`
		Memory_Fifteen []UnitMetric `xml:"memory>fifteen"`

		Memory_total_One     []UnitMetric `xml:"memory_total>one"`
		Memory_total_Five    []UnitMetric `xml:"memory_total>five"`
		Memory_total_Fifteen []UnitMetric `xml:"memory_total>fifteen"`

		Memory_max_One     []UnitMetric `xml:"memory_max>one"`
		Memory_max_Five    []UnitMetric `xml:"memory_max>five"`
		Memory_max_Fifteen []UnitMetric `xml:"memory_max>fifteen"`

		Sql_response_One     []UnitMetric `xml:"sql_response>one"`
		Sql_response_Five    []UnitMetric `xml:"sql_response>five"`
		Sql_response_Fifteen []UnitMetric `xml:"sql_response>fifteen"`

		Sql_inserts_One     []UnitMetric `xml:"sql_inserts>one"`
		Sql_inserts_Five    []UnitMetric `xml:"sql_inserts>five"`
		Sql_inserts_Fifteen []UnitMetric `xml:"sql_inserts>fifteen"`

		Sql_updates_One     []UnitMetric `xml:"sql_updates>one"`
		Sql_updates_Five    []UnitMetric `xml:"sql_updates>five"`
		Sql_updates_Fifteen []UnitMetric `xml:"sql_updates>fifteen"`

		Sql_deletes_One     []UnitMetric `xml:"sql_deletes>one"`
		Sql_deletes_Five    []UnitMetric `xml:"sql_deletes>five"`
		Sql_deletes_Fifteen []UnitMetric `xml:"sql_deletes>fifteen"`

		Sql_selects_One     []UnitMetric `xml:"sql_selects>one"`
		Sql_selects_Five    []UnitMetric `xml:"sql_selects>five"`
		Sql_selects_Fifteen []UnitMetric `xml:"sql_selects>fifteen"`

		Threads_cpu_One     []UnitMetric `xml:"threads_cpu>one"`
		Threads_cpu_Five    []UnitMetric `xml:"threads_cpu>five"`
		Threads_cpu_Fifteen []UnitMetric `xml:"threads_cpu>fifteen"`

		Threads_db_One     []UnitMetric `xml:"threads_db>one"`
		Threads_db_Five    []UnitMetric `xml:"threads_db>five"`
		Threads_db_Fifteen []UnitMetric `xml:"threads_db>fifteen"`

		Threads_network_One     []UnitMetric `xml:"threads_network>one"`
		Threads_network_Five    []UnitMetric `xml:"threads_network>five"`
		Threads_network_Fifteen []UnitMetric `xml:"threads_network>fifteen"`

		Threads_concurrency_One     []UnitMetric `xml:"threads_concurrency>one"`
		Threads_concurrency_Five    []UnitMetric `xml:"threads_concurrency>five"`
		Threads_concurrency_Fifteen []UnitMetric `xml:"threads_concurrency>fifteen"`

		Garbage_collection_One     []UnitMetric `xml:"garbage_collection>one"`
		Garbage_collection_Five    []UnitMetric `xml:"garbage_collection>five"`
		Garbage_collection_Fifteen []UnitMetric `xml:"garbage_collection>fifteen"`

		Semaphore_waiters_One     []UnitMetric `xml:"semaphore_waiters>one"`
		Semaphore_waiters_Five    []UnitMetric `xml:"semaphore_waiters>five"`
		Semaphore_waiters_Fifteen []UnitMetric `xml:"semaphore_waiters>fifteen"`
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
		XMLName                xml.Name       `xml:"xmlstats"`
		Servlet_Started        string         `xml:"servlet.started"`
		Semaphores             []Semaphores   `xml:"semaphores"`
		DBPools                []DatabasePool `xml:"db.pools>pool"`
		Servlet_Metrics        ServletMetric  `xml:"servlet.metrics"`
		System_Memory_Pct_Free float64        `xml:"system.memory.pct.free"`
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
			// "threadsCpu":                  snowStats.Threads_CPU_One.Mean,
			// "threadsDb":                   snowStats.Threads_DB_One.Mean,
			// "threadsNetwork":              snowStats.Threads_Network_One.Mean,
			// "threadsConcurrency":          snowStats.Threads_Concurrency_One.Mean,
			// "transactionCount":            snowStats.Client_Transactions_One.Count,
			// "dbResponseTime":              snowStats.SQL_Response_One.Mean,
			// "garbageCollection":           snowStats.Garbage_Collection_One.Mean,
			// "semaphoreWait":               snowStats.Semaphore_Waiters_One.Max,
			"avaliablePools":              glidPool.Available,
			"avaliableSemaphores":         snowStats.Semaphores[0].Available,
			"heapSpace":                   100 - snowStats.System_Memory_Pct_Free,
			"numOfUsers":                  len(sncUsers.User),
			"servletMetric":               snowStats.Servlet_Metrics,
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

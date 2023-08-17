package squid

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/mitchellh/mapstructure"
)

const (
	requestProtocol = "GET cache_object://localhost/%s HTTP/1.0"
)

// Info represents the show info response from Squid
type Info struct {
	Version       string `mapstructure:"Squid Object Cache"`
	NClientAccess string `mapstructure:"Number of clients accessing cache"`
	NHTTPRecv     string `mapstructure:"Number of HTTP requests received"`
	NICPRecv      string `mapstructure:"Number of ICP messages received"`
	NICPSent      string `mapstructure:"Number of ICP messages sent"`
	NQueuedICP    string `mapstructure:"Number of queued ICP replies"`
	NHTCPRecv     string `mapstructure:"Number of HTCP messages received"`
	NHTCPSent     string `mapstructure:"Number of HTCP messages sent"`
	ReqFailRatio  string `mapstructure:"Request failure ratio"`
	AvHTTPperMin  string `mapstructure:"Average HTTP requests per minute since start"`
	AvICPperMin   string `mapstructure:"Average ICP messages per minute since start"`
	LoopCalled    string `mapstructure:"Select loop called"`

	HTTPReqTime        string `mapstructure:"HTTP Requests (All)"`
	CacheMisses        string `mapstructure:"Cache Misses"`
	CacheHits          string `mapstructure:"Cache Hits"`
	NearHits           string `mapstructure:"Near Hits"`
	NotModifiedReplies string `mapstructure:"Not-Modified Replies"`
	DNSLookup          string `mapstructure:"DNS Lookups"`
	ICPQueries         string `mapstructure:"ICP Queries"`

	Uptime      string `mapstructure:"UP Time"`
	CPUTime     string `mapstructure:"CPU Time"`
	CPUUsage    string `mapstructure:"CPU Usage"`
	MaxResident string `mapstructure:"Maximum Resident Size"`
	PageFaults  string `mapstructure:"Page faults with physical i/o"`

	MemTotal     string `mapstructure:"Total accounted"`
	MemAllocCall string `mapstructure:"memPoolAlloc calls"`
	MemFreeCall  string `mapstructure:"memPoolFree calls"`

	MaxFD         string `mapstructure:"Maximum number of file descriptors"`
	LargestFDUsed string `mapstructure:"Largest file desc currently in use"`
	NFDUsed       string `mapstructure:"Number of file desc currently in use"`
	FDQueued      string `mapstructure:"Files queued for open"`
	AvailableFD   string `mapstructure:"Available number of file descriptors"`
	ReservedFD    string `mapstructure:"Reserved number of file descriptors"`
	SDiskFiles    string `mapstructure:"Store Disk files open"`
}

type Client struct {
	host string
	port string
}

func NewSquidClient(address string) (*Client, error) {
	u, err := url.Parse(address)
	if err != nil {
		return nil, fmt.Errorf("invalid url")
	}

	if u.Scheme != "tcp" {
		return nil, fmt.Errorf("invalid scheme. must be tcp")
	}

	return &Client{host: u.Hostname(), port: u.Port()}, nil
}

func (c *Client) run(cmd string) (*bytes.Buffer, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", c.host, c.port))
	response := bytes.NewBuffer(nil)

	if err != nil {
		return nil, fmt.Errorf("error connecting to %s:%s", c.host, c.port)
	}
	defer conn.Close()

	rBody := []string{
		fmt.Sprintf(requestProtocol, cmd),
		"Host: localhost",
		"User-Agent: squidclient/3.5.12",
	}

	rBody = append(rBody, "Accept: */*", "\r\n")
	request := strings.Join(rBody, "\r\n")

	_, err = conn.Write([]byte(request))
	if err != nil {
		return nil, fmt.Errorf("error writing to connection")
	}

	recv, err := io.Copy(response, conn)
	if err != nil {
		return nil, fmt.Errorf("error reading response")
	}

	if recv == 0 {
		return nil, fmt.Errorf("got empty response from Squid")
	}

	return response, nil
}

func (c *Client) GetInfo() (*Info, error) {
	res, err := c.run("info")
	if err != nil {
		return nil, err
	}

	if b, err := io.ReadAll(res); err == nil {

		resultMap := map[string]interface{}{}

		for _, ln := range strings.Split(string(b), "\n") {

			ln := strings.TrimSpace(ln)
			if ln == "" {
				continue
			}

			parts := strings.SplitN(ln, ":", 2)
			if len(parts) != 2 {
				continue
			}

			resultMap[parts[0]] = strings.TrimSpace(parts[1])
		}

		reg := regexp.MustCompile("5min: (.*?),(.*?)")

		resultMap["CPU Time"] = strings.Split(resultMap["CPU Time"].(string), " ")[0]
		resultMap["UP Time"] = strings.Split(resultMap["UP Time"].(string), " ")[0]
		resultMap["CPU Usage"] = strings.Replace(resultMap["CPU Usage"].(string), "%", "", -1)

		resultMap["HTTP Requests (All)"] = strings.Split(resultMap["HTTP Requests (All)"].(string), " ")[0]
		resultMap["Cache Hits"] = strings.Split(resultMap["Cache Hits"].(string), " ")[0]
		resultMap["Cache Misses"] = strings.Split(resultMap["Cache Misses"].(string), " ")[0]
		resultMap["Near Hits"] = strings.Split(resultMap["Near Hits"].(string), " ")[0]
		resultMap["Not-Modified Replies"] = strings.Split(resultMap["Not-Modified Replies"].(string), " ")[0]
		resultMap["DNS Lookups"] = strings.Split(resultMap["DNS Lookups"].(string), " ")[0]
		resultMap["ICP Queries"] = strings.Split(resultMap["ICP Queries"].(string), " ")[0]

		resultMap["Hits as % of all requests"] = reg.FindStringSubmatch(resultMap["Hits as % of all requests"].(string))[1]
		resultMap["Hits as % of bytes sent"] = reg.FindStringSubmatch(resultMap["Hits as % of bytes sent"].(string))[1]

		resultMap["Maximum Resident Size"] = strings.Split(resultMap["Maximum Resident Size"].(string), " ")[0]
		resultMap["Total accounted"] = strings.Split(resultMap["Total accounted"].(string), " ")[0]

		var result *Info

		if err := mapstructure.Decode(resultMap, &result); err != nil {
			return nil, err
		}
		return result, nil
	}

	return nil, err
}

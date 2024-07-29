package mbeaninfo

import (
	"bytes"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/pkg/errors"
	"golang.org/x/net/html"

	s "github.com/elastic/beats/v7/libbeat/common/schema"
	c "github.com/elastic/beats/v7/libbeat/common/schema/mapstrstr"
)

var (
	schema = s.Schema{
		"HostName":            c.Str("HostName"),
		"State":               c.Str("State"),
		"AcceptNewConnection": c.Bool("AcceptNewConnection"),
		"QuiescedStatusCode":  c.Str("QuiescedStatusCode"),
		"MaxLoadStatusCode":   c.Str("MaxLoadStatusCode"),
		"PercentLoaded":       c.Int("PercentLoaded"),
		"IsQuiesced":          c.Bool("IsQuiesced"),
		"CurrentLoad":         c.Int("CurrentLoad"),
		"OKStatusCode":        c.Str("OKStatusCode"),
		"MaxLoad":             c.Int("MaxLoad"),
	}
)

func eventMapping(content *[]byte, hostname string) (common.MapStr, error) {
	row := []string{}
	z := html.NewTokenizer(bytes.NewReader(*content))

	for z.Token().Data != "html" {
		tt := z.Next()
		if tt == html.StartTagToken {
			t := z.Token()
			if t.Data == "td" {
				inner := z.Next()
				if inner == html.TextToken {
					text := (string)(z.Text())
					t := strings.TrimSpace(text)
					row = append(row, t)
				}
			}
		}
	}

	if len(row) == 0 {
		return nil, errors.New("error parse html")
	}

	fullEvent := map[string]interface{}{}
	for i, s := range row {

		if s == "State" || s == "AcceptNewConnection" || s == "QuiescedStatusCode" || s == "MaxLoadStatusCode" || s == "PercentLoaded" || s == "CurrentLoad" || s == "IsQuiesced" || s == "OKStatusCode" || s == "MaxLoad" {
			fullEvent[s] = row[i+4]
		}
	}

	fullEvent["HostName"] = hostname

	fields, err := schema.Apply(fullEvent)
	if err != nil {
		return nil, errors.Wrap(err, "error applying schema")
	}

	return fields, nil

}

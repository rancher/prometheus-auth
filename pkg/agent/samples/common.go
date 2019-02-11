// +build test

package samples

import (
	"net/url"

	"github.com/prometheus/prometheus/prompb"
)

type Scenario struct {
	Endpoint      string
	Queries       url.Values
	Params        map[string]string
	PrompbQueries []*prompb.Query

	RespCode int
	RespBody interface{}
}

type jsonResponseData struct {
	Status    string      `json:"status"`
	Data      interface{} `json:"data,omitempty"`
	ErrorType string      `json:"errorType,omitempty"`
	Error     string      `json:"error,omitempty"`
}

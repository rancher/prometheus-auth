package agent

import (
	"bytes"
	"fmt"
	"github.com/golang/snappy"
	"github.com/juju/errors"
	prommodel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
	"github.com/prometheus/prometheus/storage/remote"
	"github.com/rancher/prometheus-auth/pkg/utils"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type queryValue struct {
	rawReq *http.Request
	value  string
}

type queryValueHijackFn func(rawValue string) (hjkValue string, err error)

func (qv *queryValue) hijack(fn queryValueHijackFn) error {
	req := qv.rawReq
	req.Form.Del("query")

	rawValue := qv.value
	utils.LogTrace(func() string {
		return fmt.Sprintf("raw => %s", rawValue)
	})

	hjkValue, err := fn(rawValue)
	if err != nil {
		return err
	}

	if len(hjkValue) != 0 {
		req.Form.Set("query", hjkValue)
		utils.LogTrace(func() string {
			return fmt.Sprintf("hjk => %s", hjkValue)
		})
	}

	req.URL.RawQuery = req.Form.Encode()

	return nil
}

type matchValues struct {
	rawReq *http.Request
	values []string
}

type matchValuesHijackFn func(rawValue string) (hjkValue string, err error)

func (mv *matchValues) hijack(fn matchValuesHijackFn) error {
	req := mv.rawReq
	req.Form.Del("match[]")

	for _, rawValue := range mv.values {
		utils.LogTrace(func() string {
			return fmt.Sprintf("raw => %s", rawValue)
		})

		hjkValue, err := fn(rawValue)
		if err != nil {
			return err
		}

		if len(hjkValue) == 0 {
			continue
		}

		req.Form.Add("match[]", hjkValue)
		utils.LogTrace(func() string {
			return fmt.Sprintf("hjk => %s", hjkValue)
		})
	}

	req.URL.RawQuery = req.Form.Encode()

	return nil
}

type queryValues struct {
	rawReq *http.Request
	value  *prompb.ReadRequest
}

type queryValuesHijackFn func(rawQuery *prompb.Query) (hjkQuery *prompb.Query, err error)

func (qv *queryValues) hijack(fn queryValuesHijackFn) error {
	req := qv.rawReq
	pbreq := qv.value
	rawQueries := pbreq.Queries
	hjkQueries := make([]*prompb.Query, 0, len(rawQueries))

	for _, rawValue := range rawQueries {
		utils.LogTrace(func() string {
			return fmt.Sprintf("raw => %s", rawValue)
		})

		hjkValue, err := fn(rawValue)
		if err != nil {
			return err
		}

		if hjkValue == nil {
			continue
		}

		hjkQueries = append(hjkQueries, hjkValue)
		utils.LogTrace(func() string {
			return fmt.Sprintf("hjk => %s", hjkValue)
		})
	}

	pbreq.Queries = hjkQueries
	marshaledData, err := pbreq.Marshal()
	if err != nil {
		return errors.Annotate(err, "unable to marshal to protobuf")
	}

	compressedData := snappy.Encode(nil, marshaledData)
	req.Body = ioutil.NopCloser(bytes.NewBuffer(compressedData))
	req.ContentLength = int64(len(compressedData))

	return nil
}

type apiRequest struct {
	authenticationCheck *sync.Once
	authorizationToken  *string
	rawReq              *http.Request
}

func (a *apiRequest) getMethod() string {
	return a.rawReq.Method
}

func (a *apiRequest) getURL() *url.URL {
	return a.rawReq.URL
}

func (a *apiRequest) getPath() string {
	return a.rawReq.URL.Path
}

func (a *apiRequest) getQueryValue() (*queryValue, error) {
	queryFormValue := a.rawReq.FormValue("query")
	if len(queryFormValue) == 0 {
		return nil, errors.New("unable to get query value from request")
	}

	return &queryValue{
		rawReq: a.rawReq,
		value:  queryFormValue,
	}, nil
}

func (a *apiRequest) getMatchValues() (*matchValues, error) {
	req := a.rawReq

	if err := req.ParseForm(); err != nil {
		return nil, errors.New("unable to parse form from request")
	}
	matchFormValues := req.Form["match[]"]
	if len(matchFormValues) == 0 {
		return nil, errors.New("unable to get match[] value from request")
	}

	return &matchValues{
		rawReq: a.rawReq,
		values: matchFormValues,
	}, nil
}

func (a *apiRequest) getQueryValues() (*queryValues, error) {
	req := a.rawReq

	pbreq, err := remote.DecodeReadRequest(req)
	if err != nil {
		return nil, errors.Annotate(err, "unable to decode protobuf from request")
	}

	return &queryValues{
		rawReq: req,
		value:  pbreq,
	}, nil
}

func (a *apiRequest) getLabelValue() (string, error) {
	endpoint := a.v1ApiEndpoint()
	labelPathValue := func() (matched string) {
		eps := strings.SplitN(endpoint, "/", 4)
		if len(eps) == 4 {
			matched = eps[2]
		}

		return
	}()
	if !prommodel.LabelNameRE.MatchString(labelPathValue) {
		return "", errors.New("unable to parse :name value from request")
	}

	return labelPathValue, nil
}

func (a *apiRequest) is(matches ...string) (ret bool) {
	current := a.getPath()
	for _, match := range matches {
		ret = strings.HasPrefix(current, match)
		if ret {
			return
		}
	}

	return
}

func (a *apiRequest) whoami() string {
	a.authenticationCheck.Do(func() {
		authorization := a.rawReq.Header.Get(authorizationHeaderKey)
		token := ""

		if len(authorization) != 0 {
			token = strings.TrimPrefix(a.rawReq.Header.Get(authorizationHeaderKey), "Bearer ")
		}

		a.authorizationToken = &token
	})

	return *a.authorizationToken
}

func (a *apiRequest) v1ApiEndpoint() string {
	return strings.TrimPrefix(a.getPath(), v1Api)
}

func (a *apiRequest) String() string {
	return fmt.Sprintf("%s - %s", a.getMethod(), a.getURL())
}

func createApi(req *http.Request) *apiRequest {
	return &apiRequest{
		authenticationCheck: &sync.Once{},
		rawReq:              req,
	}
}

package agent

import (
	"bytes"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/caas-team/prometheus-auth/pkg/data"
	"github.com/caas-team/prometheus-auth/pkg/prom"
	"github.com/golang/snappy"
	"github.com/juju/errors"
	prommodel "github.com/prometheus/common/model"
	promlb "github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/prompb"
	"github.com/prometheus/prometheus/promql"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/prometheus/prometheus/storage/remote"
	"github.com/prometheus/prometheus/util/stats"
	log "github.com/sirupsen/logrus"
)

func hijackFederate(apiCtx *apiContext) error {
	// pre check
	queries, err := url.ParseQuery(apiCtx.request.URL.RawQuery)
	if err != nil {
		return errors.Wrap(err, badRequestErr)
	}

	matchFormValues := queries["match[]"]
	for _, rawValue := range matchFormValues {
		_, err := parser.ParseMetricSelector(rawValue)
		if err != nil {
			return errors.Wrap(err, badRequestErr)
		}
	}

	// quick response
	if len(matchFormValues) == 0 || len(apiCtx.namespaceSet) == 0 {
		return apiCtx.responseMetrics(nil)
	}

	// hijack
	queries.Del("match[]")
	for idx, rawValue := range matchFormValues {
		expr, err := parser.ParseExpr(rawValue)
		if err != nil {
			return errors.Wrap(err, badRequestErr)
		}

		log.Debugf("raw federate[%s - %d] => %s", apiCtx.tag, idx, rawValue)
		hjkValue := modifyExpression(expr, apiCtx.namespaceSet)
		log.Debugf("hjk federate[%s - %d] => %s", apiCtx.tag, idx, hjkValue)

		queries.Add("match[]", hjkValue)
	}

	// inject
	reqURL := *apiCtx.request.URL
	reqURL.RawQuery = queries.Encode()

	// proxy
	newReq, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return errors.Wrap(err, internalErr)
	}

	return apiCtx.proxyWith(newReq)
}

func hijackQuery(apiCtx *apiContext) error {
	req := apiCtx.request
	apiCtx.response.Header().Set("Content-Type", "application/json")

	// pre check
	if to := req.FormValue("timeout"); len(to) != 0 {
		if _, err := parseDuration(to); err != nil {
			return errors.Wrap(err, badRequestErr)
		}
	}

	queryFormValue := req.FormValue("query")
	if len(queryFormValue) == 0 {
		return errors.Wrap(errors.New("unable to get 'query' value from request"), badRequestErr)
	}

	rawValue := queryFormValue
	queryExpr, err := parser.ParseExpr(rawValue)
	if err != nil {
		return errors.Wrap(err, badRequestErr)
	}

	// quick response
	if len(apiCtx.namespaceSet) == 0 {
		var qs *stats.QueryStats

		if queryExpr.Type() != parser.ValueTypeScalar {
			var val parser.Value
			switch queryExpr.Type() {
			case parser.ValueTypeVector:
				val = make(promql.Vector, 0)
			case parser.ValueTypeMatrix:
				val = promql.Matrix{}
			default:
				return errors.Wrap(errors.Errorf("unexpected expression type %q", queryExpr.Type()), badRequestErr)
			}

			emptyRespData := struct {
				ResultType parser.ValueType  `json:"resultType"`
				Result     parser.Value      `json:"result"`
				Stats      *stats.QueryStats `json:"stats,omitempty"`
			}{
				ResultType: val.Type(),
				Result:     val,
				Stats:      qs,
			}

			return apiCtx.responseJSON(emptyRespData)
		}
	}

	// hijack
	req.Form.Del("query")
	log.Debugf("raw query[%s - 0] => %s", apiCtx.tag, rawValue)
	hjkValue := modifyExpression(queryExpr, apiCtx.namespaceSet)
	log.Debugf("hjk query[%s - 0] => %s", apiCtx.tag, hjkValue)
	req.Form.Set("query", hjkValue)

	// inject
	reqURL := *req.URL
	reqURL.RawQuery = req.Form.Encode()

	// proxy
	newReq, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return errors.Wrap(err, internalErr)
	}

	return apiCtx.proxyWith(newReq)
}

func hijackQueryRange(apiCtx *apiContext) error {
	req := apiCtx.request
	apiCtx.response.Header().Set("Content-Type", "application/json")

	// pre check
	if to := req.FormValue("timeout"); len(to) != 0 {
		if _, err := parseDuration(to); err != nil {
			return errors.Wrap(err, badRequestErr)
		}
	}

	start, err := parseTime(req.FormValue("start"))
	if err != nil {
		return errors.Wrap(err, badRequestErr)
	}

	end, err := parseTime(req.FormValue("end"))
	if err != nil {
		return errors.Wrap(err, badRequestErr)
	}

	if end.Before(start) {
		return errors.Wrap(errors.New("end timestamp must not be before start time"), badRequestErr)
	}

	step, err := parseDuration(req.FormValue("step"))
	if err != nil {
		return errors.Wrap(err, badRequestErr)
	}

	if step <= 0 {
		return errors.Wrap(errors.New("zero or negative query resolution step widths are not accepted. Try a positive integer"), badRequestErr)
	}

	if end.Sub(start)/step > 11000 {
		return errors.Wrap(errors.New("exceeded maximum resolution of 11,000 points per timeseries. Try decreasing the query resolution (?step=XX)"), badRequestErr)
	}

	queryFormValue := req.FormValue("query")
	if len(queryFormValue) == 0 {
		return errors.Wrap(errors.New("unable to get 'query' value from request"), badRequestErr)
	}

	rawValue := queryFormValue
	queryExpr, err := parser.ParseExpr(rawValue)
	if err != nil {
		return errors.Wrap(err, badRequestErr)
	}

	// quick response
	if len(apiCtx.namespaceSet) == 0 {
		var qs *stats.QueryStats

		if queryExpr.Type() != parser.ValueTypeScalar {
			var val parser.Value
			switch queryExpr.Type() {
			case parser.ValueTypeVector:
				val = promql.Matrix{}
			case parser.ValueTypeMatrix:
				val = promql.Matrix{}
			default:
				return errors.Wrap(errors.Errorf("unexpected expression type %q", queryExpr.Type()), badRequestErr)
			}

			emptyRespData := struct {
				ResultType parser.ValueType  `json:"resultType"`
				Result     parser.Value      `json:"result"`
				Stats      *stats.QueryStats `json:"stats,omitempty"`
			}{
				ResultType: val.Type(),
				Result:     val,
				Stats:      qs,
			}

			return apiCtx.responseJSON(emptyRespData)
		}
	}

	// hijack
	req.Form.Del("query")
	log.Debugf("raw query[%s - 0] => %s", apiCtx.tag, rawValue)
	hjkValue := modifyExpression(queryExpr, apiCtx.namespaceSet)
	log.Debugf("hjk query[%s - 0] => %s", apiCtx.tag, hjkValue)
	req.Form.Set("query", hjkValue)

	// inject
	reqURL := *req.URL
	reqURL.RawQuery = req.Form.Encode()

	// proxy
	newReq, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return errors.Wrap(err, internalErr)
	}

	return apiCtx.proxyWith(newReq)
}

func hijackSeries(apiCtx *apiContext) error {
	apiCtx.response.Header().Set("Content-Type", "application/json")

	// pre check
	queries, err := url.ParseQuery(apiCtx.request.URL.RawQuery)
	if err != nil {
		return errors.Wrap(err, badRequestErr)
	}

	if t := queries.Get("start"); t != "" {
		if _, err := parseTime(t); err != nil {
			return errors.Wrap(err, badRequestErr)
		}
	}

	if t := queries.Get("end"); t != "" {
		if _, err := parseTime(t); err != nil {
			return errors.Wrap(err, badRequestErr)
		}
	}

	matchFormValues := queries["match[]"]
	if len(matchFormValues) == 0 {
		return errors.Wrap(errors.New("no match[] parameter provided"), badRequestErr)
	}

	for _, rawValue := range matchFormValues {
		_, err := parser.ParseMetricSelector(rawValue)
		if err != nil {
			return errors.Wrap(err, badRequestErr)
		}
	}

	// quick response
	if len(apiCtx.namespaceSet) == 0 {
		emptyRespData := make([]promlb.Labels, 0)

		return apiCtx.responseJSON(emptyRespData)
	}

	// hijack
	queries.Del("match[]")
	for idx, rawValue := range matchFormValues {
		expr, err := parser.ParseExpr(rawValue)
		if err != nil {
			return errors.Wrap(err, badRequestErr)
		}

		log.Debugf("raw series[%s - %d] => %s", apiCtx.tag, idx, rawValue)
		hjkValue := modifyExpression(expr, apiCtx.namespaceSet)
		log.Debugf("hjk series[%s - %d] => %s", apiCtx.tag, idx, hjkValue)

		queries.Add("match[]", hjkValue)
	}

	// inject
	reqURL := *apiCtx.request.URL
	reqURL.RawQuery = queries.Encode()

	// proxy
	newReq, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return errors.Wrap(err, internalErr)
	}

	return apiCtx.proxyWith(newReq)
}

func hijackRead(apiCtx *apiContext) error {
	req := apiCtx.request

	// pre check
	pbreq, err := remote.DecodeReadRequest(req)
	if err != nil {
		return errors.Wrap(err, badRequestErr)
	}

	rawQueries := pbreq.Queries

	// quick response
	if len(apiCtx.namespaceSet) == 0 {
		size := len(rawQueries)

		results := make([]*prompb.QueryResult, 0, size)
		for i := 0; i < size; i++ {
			results = append(results, &prompb.QueryResult{})
		}

		emptyRespData := &prompb.ReadResponse{
			Results: results,
		}

		return apiCtx.responseProto(emptyRespData)
	}

	// hijack
	hjkQueries := make([]*prompb.Query, 0, len(rawQueries))
	for idx, rawValue := range rawQueries {
		log.Debugf("raw read[%s - %d] => %s", apiCtx.tag, idx, rawValue)
		hjkValue := modifyQuery(rawValue, apiCtx.namespaceSet, apiCtx.filterReaderLabelSet)
		log.Debugf("hjk read[%s - %d] => %s", apiCtx.tag, idx, hjkValue)

		hjkQueries = append(hjkQueries, hjkValue)
	}
	pbreq.Queries = hjkQueries

	// inject
	marshaledData, err := pbreq.Marshal()
	if err != nil {
		return errors.Wrap(err, badRequestErr)
	}
	compressedData := snappy.Encode(nil, marshaledData)

	// proxy
	newReq, err := http.NewRequest(http.MethodPost, req.URL.String(), bytes.NewBuffer(compressedData))
	if err != nil {
		return errors.Wrap(err, internalErr)
	}

	return apiCtx.proxyWith(newReq)
}

func hijackLabelNamespaces(apiCtx *apiContext) error {
	// quick response
	if len(apiCtx.namespaceSet) == 0 {
		emptyRespData := make([]string, 0)

		return apiCtx.responseJSON(emptyRespData)
	}

	// hijack
	// for performance considerations, just return all owned namespaces
	hjkValue := make(prommodel.LabelValues, 0, len(apiCtx.namespaceSet))
	for _, v := range apiCtx.namespaceSet.Values() {
		hjkValue = append(hjkValue, prommodel.LabelValue(v))
	}

	return apiCtx.responseJSON(hjkValue)
}

func hijackLabelName(apiCtx *apiContext) error {
	apiCtx.response.Header().Set("Content-Type", "application/json")

	// quick response
	if len(apiCtx.namespaceSet) == 0 {
		emptyRespData := make([]string, 0)

		return apiCtx.responseJSON(emptyRespData)
	}

	// hijack
	expr := prom.NewExprForCountAllLabels(apiCtx.namespaceSet.Values())
	vals, warns, err := apiCtx.remoteAPI.Query(apiCtx.request.Context(), expr, time.Time{})
	for _, warn := range warns {
		log.Debugf("received warning on query: %s", warn)
	}
	if err != nil {
		return errors.Wrap(err, notProvisionedErr)
	}

	vectorVals, ok := vals.(prommodel.Vector)
	if !ok {
		return errors.Wrap(err, notProvisionedErr)
	}

	hjkValues := make(prommodel.LabelValues, 0, len(vectorVals))
	for _, vectorVal := range vectorVals {
		valLabelSet := prommodel.LabelSet(vectorVal.Metric)
		hjkValues = append(hjkValues, valLabelSet["__name__"])
	}

	return apiCtx.responseJSON(hjkValues)
}

func parseTime(s string) (time.Time, error) {
	if t, err := strconv.ParseFloat(s, 64); err == nil {
		s, ns := math.Modf(t)
		return time.Unix(int64(s), int64(ns*float64(time.Second))), nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}

	return time.Time{}, errors.Errorf("cannot parse %q to a valid timestamp", s)
}

func parseDuration(s string) (time.Duration, error) {
	if d, err := strconv.ParseFloat(s, 64); err == nil {
		ts := d * float64(time.Second)
		if ts > float64(math.MaxInt64) || ts < float64(math.MinInt64) {
			return 0, errors.Errorf("cannot parse %q to a valid duration. It overflows int64", s)
		}
		return time.Duration(ts), nil
	}
	if d, err := prommodel.ParseDuration(s); err == nil {
		return time.Duration(d), nil
	}
	return 0, errors.Errorf("cannot parse %q to a valid duration", s)
}

func modifyExpression(originalExpr parser.Expr, namespaceSet data.Set) (modifiedExpr string) {
	parser.Inspect(originalExpr, func(node parser.Node, _ []parser.Node) error {
		switch n := node.(type) {
		case *parser.VectorSelector:
			n.LabelMatchers = prom.FilterMatchers(namespaceSet, n.LabelMatchers)
		case *parser.MatrixSelector:
			vs, ok := n.VectorSelector.(*parser.VectorSelector)
			if !ok {
				// If it is not a vector selector, we don't need to modify the labelMatchers
				//
				// However, this is unexpected since we always expect to be able to extract
				// the VectorSelector from the MatrixSelector.
				//
				// If this is not the case, we may be encountering an unexpected error here.
				log.Errorf("unable to extract vector selector from matrix selector")
				return nil
			}
			vs.LabelMatchers = prom.FilterMatchers(namespaceSet, vs.LabelMatchers)
			n.VectorSelector = vs
		}
		return nil
	})

	return originalExpr.String()
}

func modifyQuery(originalQuery *prompb.Query, namespaceSet, filterReaderLabelSet data.Set) (modifiedQuery *prompb.Query) {
	rawMatchers := originalQuery.GetMatchers()
	filteredMatchers := make([]*prompb.LabelMatcher, 0, len(rawMatchers))
	for _, rawMatcher := range rawMatchers {
		if _, exist := filterReaderLabelSet[rawMatcher.GetName()]; !exist {
			filteredMatchers = append(filteredMatchers, rawMatcher)
		}
	}

	originalQuery.Matchers = prom.FilterLabelMatchers(namespaceSet, filteredMatchers)
	return originalQuery
}

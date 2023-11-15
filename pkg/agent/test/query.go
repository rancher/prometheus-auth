package test

import (
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/timestamp"
	"github.com/prometheus/prometheus/promql"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/prometheus/prometheus/util/stats"
)

type queryData struct {
	ResultType parser.ValueType  `json:"resultType"`
	Result     parser.Value      `json:"result"`
	Stats      *stats.QueryStats `json:"stats,omitempty"`
}

var start = time.Unix(0, 0)

var NoneNamespacesTokenQueryScenarios = map[string]Scenario{
	"query - none expression with time 1": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"2"},
			"time":  []string{"123.4"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeScalar,
				Result: promql.Scalar{
					V: 2,
					T: timestamp.FromTime(start.Add(123*time.Second + 400*time.Millisecond)),
				},
			},
		},
	},
	"query - none expression with time 2": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"0.333"},
			"time":  []string{"1970-01-01T00:02:03Z"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeScalar,
				Result: promql.Scalar{
					V: 0.333,
					T: timestamp.FromTime(start.Add(123 * time.Second)),
				},
			},
		},
	},
	"query - bad query `invalid][query`": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"invalid][query"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `1:8: parse error: unexpected right bracket ']'`,
		},
	},
	"query - test_metric1": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"test_metric1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeVector,
				Result:     promql.Vector{},
			},
		},
	},
	"query - test_metric1{namespace='ns-c'}": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"test_metric1{namespace='ns-c'}"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeVector,
				Result:     promql.Vector{},
			},
		},
	},
	"query - test_metric2{foo='boo'}": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"test_metric2{foo='boo'}"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeVector,
				Result:     promql.Vector{},
			},
		},
	},
	"query - test_metric_without_labels": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"test_metric_without_labels"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeVector,
				Result:     promql.Vector{},
			},
		},
	},
	"query - does_not_match_anything": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"does_not_match_anything"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeVector,
				Result:     promql.Vector{},
			},
		},
	},
	"query_range - query=time()&start=0&end=2&step=1": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"time()"},
			"start": []string{"0"},
			"end":   []string{"2"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeMatrix,
				Result: promql.Matrix{
					promql.Series{
						Floats: []promql.FPoint{
							{F: 0, T: timestamp.FromTime(start)},
							{F: 1, T: timestamp.FromTime(start.Add(1 * time.Second))},
							{F: 2, T: timestamp.FromTime(start.Add(2 * time.Second))},
						},
						Metric: nil,
					},
				},
			},
		},
	},
	"query_range - query=time()&end=2&step=1": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"time()"},
			"end":   []string{"2"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `cannot parse "" to a valid timestamp`,
		},
	},
	"query_range - bad query `invalid][query`": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"invalid][query"},
			"start": []string{"0"},
			"end":   []string{"100"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `1:8: parse error: unexpected right bracket ']'`,
		},
	},
	"query_range - invalid step": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"time()"},
			"start": []string{"1"},
			"end":   []string{"2"},
			"step":  []string{"0"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `zero or negative query resolution step widths are not accepted. Try a positive integer`,
		},
	},
	"query_range - start after end": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"time()"},
			"start": []string{"2"},
			"end":   []string{"1"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `end timestamp must not be before start time`,
		},
	},
	"query_range - start overflows int64 internally": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"time()"},
			"start": []string{"148966367200.372"},
			"end":   []string{"1489667272.372"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     "end timestamp must not be before start time",
		},
	},
	"query_range - test_metric1": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"test_metric1"},
			"start": []string{"0"},
			"end":   []string{"2"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeMatrix,
				Result:     promql.Matrix{},
			},
		},
	},
	"query_range - test_metric1{namespace='ns-c'}": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"test_metric1{namespace='ns-c'}"},
			"start": []string{"0"},
			"end":   []string{"2"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeMatrix,
				Result:     promql.Matrix{},
			},
		},
	},
	"query_range - test_metric2{foo='boo'}": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"test_metric2{foo='boo'}"},
			"start": []string{"0"},
			"end":   []string{"2"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeMatrix,
				Result:     promql.Matrix{},
			},
		},
	},
	"query_range - test_metric_without_labels": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"test_metric_without_labels"},
			"start": []string{"0"},
			"end":   []string{"2"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeMatrix,
				Result:     promql.Matrix{},
			},
		},
	},
	"query_range - does_not_match_anything": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"does_not_match_anything"},
			"start": []string{"0"},
			"end":   []string{"2"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeVector,
				Result:     promql.Vector{},
			},
		},
	},
}

var SomeNamespacesTokenQueryScenarios = map[string]Scenario{
	"query - none expression with time 1": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"2"},
			"time":  []string{"123.4"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeScalar,
				Result: promql.Scalar{
					V: 2,
					T: timestamp.FromTime(start.Add(123*time.Second + 400*time.Millisecond)),
				},
			},
		},
	},
	"query - none expression with time 2": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"0.333"},
			"time":  []string{"1970-01-01T00:02:03Z"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeScalar,
				Result: promql.Scalar{
					V: 0.333,
					T: timestamp.FromTime(start.Add(123 * time.Second)),
				},
			},
		},
	},
	"query - bad query `invalid][query`": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"invalid][query"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `1:8: parse error: unexpected right bracket ']'`,
		},
	},
	"query - test_metric1": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"test_metric1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeVector,
				Result: promql.Vector{
					promql.Sample{
						Metric: []labels.Label{
							{
								Name:  "__name__",
								Value: "test_metric1",
							},
							{
								Name:  "foo",
								Value: "bar",
							},
							{
								Name:  "namespace",
								Value: "ns-a",
							},
						},
						F: 0,
						T: timestamp.FromTime(start),
					},
				},
			},
		},
	},
	"query - test_metric1{namespace='ns-c'}": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"test_metric1{namespace='ns-c'}"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeVector,
				Result:     promql.Vector{},
			},
		},
	},
	"query - test_metric2{foo='boo'}": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"test_metric2{foo='boo'}"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeVector,
				Result:     promql.Vector{},
			},
		},
	},
	"query - test_metric1[5m]": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"test_metric1[5m]"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeMatrix,
				Result: promql.Matrix{
					promql.Series{
						Floats: []promql.FPoint{{
							T: timestamp.FromTime(start),
							F: 0,
						},
						},
						Metric: []labels.Label{
							{
								Name:  "__name__",
								Value: "test_metric1",
							},
							{
								Name:  "foo",
								Value: "bar",
							},
							{
								Name:  "namespace",
								Value: "ns-a",
							},
						},
					},
				},
			},
		},
	},
	"query - test_metric_without_labels": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"test_metric_without_labels"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeVector,
				Result:     promql.Vector{},
			},
		},
	},
	"query - does_not_match_anything": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"does_not_match_anything"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeVector,
				Result:     promql.Vector{},
			},
		},
	},
	"query_range - query=time()&start=0&end=2&step=1": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"time()"},
			"start": []string{"0"},
			"end":   []string{"2"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeMatrix,
				Result: promql.Matrix{
					promql.Series{
						Floats: []promql.FPoint{
							{F: 0, T: timestamp.FromTime(start)},
							{F: 1, T: timestamp.FromTime(start.Add(1 * time.Second))},
							{F: 2, T: timestamp.FromTime(start.Add(2 * time.Second))},
						},
						Metric: nil,
					},
				},
			},
		},
	},
	"query_range - query=time()&end=2&step=1": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"time()"},
			"end":   []string{"2"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `cannot parse "" to a valid timestamp`,
		},
	},
	"query_range - bad query `invalid][query`": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"invalid][query"},
			"start": []string{"0"},
			"end":   []string{"100"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `1:8: parse error: unexpected right bracket ']'`,
		},
	},
	"query_range - invalid step": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"time()"},
			"start": []string{"1"},
			"end":   []string{"2"},
			"step":  []string{"0"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `zero or negative query resolution step widths are not accepted. Try a positive integer`,
		},
	},
	"query_range - start after end": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"time()"},
			"start": []string{"2"},
			"end":   []string{"1"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `end timestamp must not be before start time`,
		},
	},
	"query_range - start overflows int64 internally": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"time()"},
			"start": []string{"148966367200.372"},
			"end":   []string{"1489667272.372"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     "end timestamp must not be before start time",
		},
	},
	"query_range - test_metric1": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"test_metric1"},
			"start": []string{"0"},
			"end":   []string{"2"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeMatrix,
				Result: promql.Matrix{
					promql.Series{
						Metric: []labels.Label{
							{
								Name:  "__name__",
								Value: "test_metric1",
							},
							{
								Name:  "foo",
								Value: "bar",
							},
							{
								Name:  "namespace",
								Value: "ns-a",
							},
						},
						Floats: []promql.FPoint{
							{F: 0, T: timestamp.FromTime(start)},
							{F: 0, T: timestamp.FromTime(start.Add(1 * time.Second))},
							{F: 0, T: timestamp.FromTime(start.Add(2 * time.Second))},
						},
					},
				},
			},
		},
	},
	"query_range - test_metric1{namespace='ns-c'}": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"test_metric1{namespace='ns-c'}"},
			"start": []string{"0"},
			"end":   []string{"2"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeMatrix,
				Result:     promql.Matrix{},
			},
		},
	},
	"query_range - test_metric2{foo='boo'}": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"test_metric2{foo='boo'}"},
			"start": []string{"0"},
			"end":   []string{"2"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeMatrix,
				Result:     promql.Matrix{},
			},
		},
	},
	"query_range - test_metric_without_labels": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"test_metric_without_labels"},
			"start": []string{"0"},
			"end":   []string{"2"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeMatrix,
				Result:     promql.Matrix{},
			},
		},
	},
	"query_range - does_not_match_anything": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"does_not_match_anything"},
			"start": []string{"0"},
			"end":   []string{"2"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeVector,
				Result:     promql.Vector{},
			},
		},
	},
}

var MyTokenQueryScenarios = map[string]Scenario{
	"query - none expression with time 1": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"2"},
			"time":  []string{"123.4"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeScalar,
				Result: promql.Scalar{
					V: 2,
					T: timestamp.FromTime(start.Add(123*time.Second + 400*time.Millisecond)),
				},
			},
		},
	},
	"query - none expression with time 2": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"0.333"},
			"time":  []string{"1970-01-01T00:02:03Z"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeScalar,
				Result: promql.Scalar{
					V: 0.333,
					T: timestamp.FromTime(start.Add(123 * time.Second)),
				},
			},
		},
	},
	"query - bad query `invalid][query`": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"invalid][query"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `invalid parameter "query": 1:8: parse error: unexpected right bracket ']'`,
		},
	},
	"query - test_metric1": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"test_metric1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeVector,
				Result: promql.Vector{
					{
						Metric: []labels.Label{
							{
								Name:  "__name__",
								Value: "test_metric1",
							},
							{
								Name:  "foo",
								Value: "bar",
							},
							{
								Name:  "namespace",
								Value: "ns-a",
							},
						},
						F: 0,
						T: timestamp.FromTime(start),
					},
					{
						Metric: []labels.Label{
							{
								Name:  "__name__",
								Value: "test_metric1",
							},
							{
								Name:  "foo",
								Value: "boo",
							},
							{
								Name:  "namespace",
								Value: "ns-c",
							},
						},
						F: 1,
						T: timestamp.FromTime(start),
					},
				},
			},
		},
	},
	"query - test_metric1{namespace='ns-c'}": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"test_metric1{namespace='ns-c'}"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeVector,
				Result: promql.Vector{
					promql.Sample{
						Metric: []labels.Label{
							{
								Name:  "__name__",
								Value: "test_metric1",
							},
							{
								Name:  "foo",
								Value: "boo",
							},
							{
								Name:  "namespace",
								Value: "ns-c",
							},
						},

						F: 1,
						T: timestamp.FromTime(start),
					},
				},
			},
		},
	},
	"query - test_metric2{foo='boo'}": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"test_metric2{foo='boo'}"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeVector,
				Result: promql.Vector{
					promql.Sample{
						Metric: []labels.Label{
							{
								Name:  "__name__",
								Value: "test_metric2",
							},
							{
								Name:  "foo",
								Value: "boo",
							},
						},

						F: 1,
						T: timestamp.FromTime(start),
					},
				},
			},
		},
	},
	"query - test_metric1[5m]": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"test_metric1[5m]"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeMatrix,
				Result: promql.Matrix{
					promql.Series{
						Floats: []promql.FPoint{
							{F: 0, T: timestamp.FromTime(start)},
						},
						Metric: []labels.Label{
							{
								Name:  "__name__",
								Value: "test_metric1",
							},
							{
								Name:  "foo",
								Value: "bar",
							},
							{
								Name:  "namespace",
								Value: "ns-a",
							},
						},
					},
					promql.Series{
						Floats: []promql.FPoint{
							{F: 1, T: timestamp.FromTime(start)},
						},
						Metric: []labels.Label{
							{
								Name:  "__name__",
								Value: "test_metric1",
							},
							{
								Name:  "foo",
								Value: "boo",
							},
							{
								Name:  "namespace",
								Value: "ns-c",
							},
						},
					},
				},
			},
		},
	},
	"query - test_metric_without_labels": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"test_metric_without_labels"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeVector,
				Result: promql.Vector{promql.Sample{
					Metric: []labels.Label{
						{
							Name:  "__name__",
							Value: "test_metric_without_labels",
						},
					},

					F: 1,
					T: timestamp.FromTime(start),
				}},
			},
		},
	},
	"query - does_not_match_anything": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"does_not_match_anything"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeVector,
				Result:     promql.Vector{},
			},
		},
	},
	"query_range - query=time()&start=0&end=2&step=1": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"time()"},
			"start": []string{"0"},
			"end":   []string{"2"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeMatrix,
				Result: promql.Matrix{
					promql.Series{
						Floats: []promql.FPoint{
							{F: 0, T: timestamp.FromTime(start)},
							{F: 1, T: timestamp.FromTime(start.Add(1 * time.Second))},
							{F: 2, T: timestamp.FromTime(start.Add(2 * time.Second))},
						},
						Metric: nil,
					},
				},
			},
		},
	},
	"query_range - query=time()&end=2&step=1": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"time()"},
			"end":   []string{"2"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `invalid parameter "start": cannot parse "" to a valid timestamp`,
		},
	},
	"query_range - bad query `invalid][query`": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"invalid][query"},
			"start": []string{"0"},
			"end":   []string{"100"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `invalid parameter "query": 1:8: parse error: unexpected right bracket ']'`,
		},
	},
	"query_range - invalid step": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"time()"},
			"start": []string{"1"},
			"end":   []string{"2"},
			"step":  []string{"0"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `invalid parameter "step": zero or negative query resolution step widths are not accepted. Try a positive integer`,
		},
	},
	"query_range - start after end": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"time()"},
			"start": []string{"2"},
			"end":   []string{"1"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `invalid parameter "end": end timestamp must not be before start time`,
		},
	},
	"query_range - start overflows int64 internally": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"time()"},
			"start": []string{"148966367200.372"},
			"end":   []string{"1489667272.372"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `invalid parameter "end": end timestamp must not be before start time`,
		},
	},
	"query_range - test_metric1": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"test_metric1"},
			"start": []string{"0"},
			"end":   []string{"2"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeMatrix,
				Result: promql.Matrix{
					promql.Series{
						Metric: []labels.Label{
							{
								Name:  "__name__",
								Value: "test_metric1",
							},
							{
								Name:  "foo",
								Value: "bar",
							},
							{
								Name:  "namespace",
								Value: "ns-a",
							},
						},
						Floats: []promql.FPoint{
							{F: 0, T: timestamp.FromTime(start)},
							{F: 0, T: timestamp.FromTime(start.Add(1 * time.Second))},
							{F: 0, T: timestamp.FromTime(start.Add(2 * time.Second))},
						},
					},
					promql.Series{
						Metric: []labels.Label{
							{
								Name:  "__name__",
								Value: "test_metric1",
							},
							{
								Name:  "foo",
								Value: "boo",
							},
							{
								Name:  "namespace",
								Value: "ns-c",
							},
						},
						Floats: []promql.FPoint{
							{F: 1, T: timestamp.FromTime(start)},
							{F: 1, T: timestamp.FromTime(start.Add(1 * time.Second))},
							{F: 1, T: timestamp.FromTime(start.Add(2 * time.Second))},
						},
					},
				},
			},
		},
	},
	"query_range - test_metric1{namespace='ns-c'}": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"test_metric1{namespace='ns-c'}"},
			"start": []string{"0"},
			"end":   []string{"2"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeMatrix,
				Result: promql.Matrix{
					promql.Series{
						Metric: []labels.Label{
							{
								Name:  "__name__",
								Value: "test_metric1",
							},
							{
								Name:  "foo",
								Value: "boo",
							},
							{
								Name:  "namespace",
								Value: "ns-c",
							},
						},
						Floats: []promql.FPoint{
							{F: 1, T: timestamp.FromTime(start)},
							{F: 1, T: timestamp.FromTime(start.Add(1 * time.Second))},
							{F: 1, T: timestamp.FromTime(start.Add(2 * time.Second))},
						},
					},
				},
			},
		},
	},
	"query_range - test_metric2{foo='boo'}": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"test_metric2{foo='boo'}"},
			"start": []string{"0"},
			"end":   []string{"2"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeMatrix,
				Result: promql.Matrix{
					promql.Series{
						Metric: []labels.Label{
							{
								Name:  "__name__",
								Value: "test_metric2",
							},
							{
								Name:  "foo",
								Value: "boo",
							},
						},
						Floats: []promql.FPoint{
							{F: 1, T: timestamp.FromTime(start)},
							{F: 1, T: timestamp.FromTime(start.Add(1 * time.Second))},
							{F: 1, T: timestamp.FromTime(start.Add(2 * time.Second))},
						},
					},
				},
			},
		},
	},
	"query_range - test_metric_without_labels": {
		Endpoint: "/query_range",
		Queries: url.Values{
			"query": []string{"test_metric_without_labels"},
			"start": []string{"0"},
			"end":   []string{"2"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeMatrix,
				Result: promql.Matrix{
					promql.Series{
						Metric: []labels.Label{
							{
								Name:  "__name__",
								Value: "test_metric_without_labels",
							},
						},
						Floats: []promql.FPoint{
							{F: 1, T: timestamp.FromTime(start)},
							{F: 1, T: timestamp.FromTime(start.Add(1 * time.Second))},
							{F: 1, T: timestamp.FromTime(start.Add(2 * time.Second))},
						},
					},
				},
			},
		},
	},
	"query_range - does_not_match_anything": {
		Endpoint: "/query",
		Queries: url.Values{
			"query": []string{"does_not_match_anything"},
			"start": []string{"0"},
			"end":   []string{"2"},
			"step":  []string{"1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: &queryData{
				ResultType: parser.ValueTypeVector,
				Result:     promql.Vector{},
			},
		},
	},
}

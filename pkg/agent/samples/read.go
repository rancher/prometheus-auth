// +build test

package samples

import (
	"net/http"
	"testing"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/prompb"
	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/prometheus/storage/remote"
)

func mockQueries(t *testing.T, externalLabelMatchers []*labels.Matcher) [][]*prompb.Query {
	// avg(test_metric1)
	metricNameMatcher, err := labels.NewMatcher(labels.MatchEqual, "__name__", "test_metric1")
	if err != nil {
		t.Fatal(err)
	}
	avgQuery, err := remote.ToQuery(0, 1, append(externalLabelMatchers, metricNameMatcher), &storage.SelectParams{Step: 0, Func: "avg"})
	if err != nil {
		t.Fatal(err)
	}

	// count(test_metric1{namespace="ns-c"})
	countNamespaceMatcher, err := labels.NewMatcher(labels.MatchEqual, "namespace", "ns-c")
	if err != nil {
		t.Fatal(err)
	}
	countQuery, err := remote.ToQuery(0, 1, append(externalLabelMatchers, metricNameMatcher, countNamespaceMatcher), &storage.SelectParams{Step: 0, Func: "count"})
	if err != nil {
		t.Fatal(err)
	}

	// sum({foo="boo"})
	sumLabelMatcher, err := labels.NewMatcher(labels.MatchEqual, "foo", "boo")
	if err != nil {
		t.Fatal(err)
	}
	sumQuery, err := remote.ToQuery(0, 1, append(externalLabelMatchers, sumLabelMatcher), &storage.SelectParams{Step: 0, Func: "sum"})
	if err != nil {
		t.Fatal(err)
	}

	// test_metric1[5m]
	query, err := remote.ToQuery(0, 1, append(externalLabelMatchers, metricNameMatcher), &storage.SelectParams{Step: 5})
	if err != nil {
		t.Fatal(err)
	}

	return [][]*prompb.Query{
		{
			avgQuery,
		},
		{
			countQuery,
		},
		{
			sumQuery,
		},
		{
			query,
		},
	}
}

func NoneNamespacesTokenReadScenarios(t *testing.T) map[string]Scenario {
	// clusterPrometheusLabel
	clientLabelMatchers := func() []*labels.Matcher {
		prometheusLabelMatcher, err := labels.NewMatcher(labels.MatchEqual, "prometheus", "project-level/test")
		if err != nil {
			t.Fatal(err)
		}

		return []*labels.Matcher{
			prometheusLabelMatcher,
		}
	}()

	queries := mockQueries(t, clientLabelMatchers)

	return map[string]Scenario{
		"avg(test_metric1)": {
			PrompbQueries: queries[0],
			RespCode:      http.StatusOK,
			RespBody: []*prompb.QueryResult{
				{},
			},
		},
		`count(test_metric1{namespace="ns-c"})`: {
			PrompbQueries: queries[1],
			RespCode:      http.StatusOK,
			RespBody: []*prompb.QueryResult{
				{},
			},
		},
		`sum({foo="boo"})`: {
			PrompbQueries: queries[2],
			RespCode:      http.StatusOK,
			RespBody: []*prompb.QueryResult{
				{},
			},
		},
		"test_metric1[5m]": {
			PrompbQueries: queries[3],
			RespCode:      http.StatusOK,
			RespBody: []*prompb.QueryResult{
				{},
			},
		},
	}
}

func SomeNamespacesTokenReadScenarios(t *testing.T) map[string]Scenario {
	// clusterPrometheusLabel
	clientLabelMatchers := func() []*labels.Matcher {
		prometheusLabelMatcher, err := labels.NewMatcher(labels.MatchEqual, "prometheus", "project-level/test")
		if err != nil {
			t.Fatal(err)
		}

		return []*labels.Matcher{
			prometheusLabelMatcher,
		}
	}()

	queries := mockQueries(t, clientLabelMatchers)

	return map[string]Scenario{
		"avg(test_metric1)": {
			PrompbQueries: queries[0],
			RespCode:      http.StatusOK,
			RespBody: []*prompb.QueryResult{
				{
					Timeseries: []*prompb.TimeSeries{
						{
							Labels: []*prompb.Label{
								{Name: "__name__", Value: "test_metric1"},
								{Name: "foo", Value: "bar"},
								{Name: "namespace", Value: "ns-a"},
								{Name: "prometheus", Value: "cluster-level/test"},
							},
							Samples: []*prompb.Sample{
								{Value: 0, Timestamp: 0},
							},
						},
					},
				},
			},
		},
		`count(test_metric1{namespace="ns-c"})`: {
			PrompbQueries: queries[1],
			RespCode:      http.StatusOK,
			RespBody: []*prompb.QueryResult{
				{},
			},
		},
		`sum({foo="boo"})`: {
			PrompbQueries: queries[2],
			RespCode:      http.StatusOK,
			RespBody: []*prompb.QueryResult{
				{},
			},
		},
		"test_metric1[5m]": {
			PrompbQueries: queries[3],
			RespCode:      http.StatusOK,
			RespBody: []*prompb.QueryResult{
				{
					Timeseries: []*prompb.TimeSeries{
						{
							Labels: []*prompb.Label{
								{Name: "__name__", Value: "test_metric1"},
								{Name: "foo", Value: "bar"},
								{Name: "namespace", Value: "ns-a"},
								{Name: "prometheus", Value: "cluster-level/test"},
							},
							Samples: []*prompb.Sample{
								{Value: 0, Timestamp: 0},
							},
						},
					},
				},
			},
		},
	}
}

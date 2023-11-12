package test

import (
	"net/http"
	"net/url"

	"github.com/prometheus/prometheus/model/labels"
)

var NoneNamespacesTokenSeriesScenarios = map[string]Scenario{
	"missing match[] query Params in series requests": {
		Queries:  url.Values{},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     "no match[] parameter provided",
		},
	},
	"bad match[] `invalid][query`": {
		Queries: url.Values{
			"match[]": []string{"invalid][query"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `1:8: parse error: unexpected right bracket ']'`,
		},
	},
	"test_metric1": {
		Queries: url.Values{
			"match[]": []string{"test_metric1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"test_metric1{namespace='ns-c'}": {
		Queries: url.Values{
			"match[]": []string{"test_metric1{namespace='ns-c'}"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"test_metric2{foo='boo'}": {
		Queries: url.Values{
			"match[]": []string{"test_metric2{foo='boo'}"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"{foo='boo'}": {
		Queries: url.Values{
			"match[]": []string{"{foo='boo'}"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"two matches": {
		Queries: url.Values{
			"match[]": []string{`test_metric1{foo=~".+o$"}`, `test_metric1{foo=~".+o"}`},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"two matches, but one is `none`": {
		Queries: url.Values{
			"match[]": []string{`test_metric2{foo=~".+o"}`, `none`},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"test_metric_without_labels": {
		Queries: url.Values{
			"match[]": []string{"test_metric_without_labels"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"does_not_match_anything": {
		Queries: url.Values{
			"match[]": []string{"does_not_match_anything"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"start and end before series starts": {
		Queries: url.Values{
			"match[]": []string{`test_metric1`},
			"start":   []string{"-2"},
			"end":     []string{"-1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"start and end after series ends": {
		Queries: url.Values{
			"match[]": []string{`test_metric1`},
			"start":   []string{"100000"},
			"end":     []string{"100001"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"start and end within series": {
		Queries: url.Values{
			"match[]": []string{`test_metric1`},
			"start":   []string{"1"},
			"end":     []string{"100"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"start within series, end after": {
		Queries: url.Values{
			"match[]": []string{`test_metric1`},
			"start":   []string{"1"},
			"end":     []string{"100000"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"start before series, end within series": {
		Queries: url.Values{
			"match[]": []string{`test_metric1`},
			"start":   []string{"-1"},
			"end":     []string{"1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
}

var SomeNamespacesTokenSeriesScenarios = map[string]Scenario{
	"missing match[] query Params in series requests": {
		Queries:  url.Values{},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     "no match[] parameter provided",
		},
	},
	"bad match[] `invalid][query`": {
		Queries: url.Values{
			"match[]": []string{"invalid][query"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `1:8: parse error: unexpected right bracket ']'`,
		},
	},
	"test_metric1": {
		Queries: url.Values{
			"match[]": []string{"test_metric1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: []labels.Labels{
				labels.FromStrings("__name__", "test_metric1", "foo", "bar", "namespace", "ns-a"),
			},
		},
	},
	"test_metric1{namespace='ns-c'}": {
		Queries: url.Values{
			"match[]": []string{"test_metric1{namespace='ns-c'}"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"test_metric2{foo='boo'}": {
		Queries: url.Values{
			"match[]": []string{"test_metric2{foo='boo'}"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"{foo='boo'}": {
		Queries: url.Values{
			"match[]": []string{"{foo='boo'}"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"two matches": {
		Queries: url.Values{
			"match[]": []string{`test_metric1{foo=~".+o$"}`, `test_metric1{foo=~".+o"}`},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"two matches, but one is `none`": {
		Queries: url.Values{
			"match[]": []string{`test_metric2{foo=~".+o"}`, `none`},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"test_metric_without_labels": {
		Queries: url.Values{
			"match[]": []string{"test_metric_without_labels"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"does_not_match_anything": {
		Queries: url.Values{
			"match[]": []string{"does_not_match_anything"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"start and end before series starts": {
		Queries: url.Values{
			"match[]": []string{`test_metric1`},
			"start":   []string{"-2"},
			"end":     []string{"-1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"start and end after series ends": {
		Queries: url.Values{
			"match[]": []string{`test_metric1`},
			"start":   []string{"100000"},
			"end":     []string{"100001"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"start and end within series": {
		Queries: url.Values{
			"match[]": []string{`test_metric1`},
			"start":   []string{"1"},
			"end":     []string{"100"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: []labels.Labels{
				labels.FromStrings("__name__", "test_metric1", "foo", "bar", "namespace", "ns-a"),
			},
		},
	},
	"start within series, end after": {
		Queries: url.Values{
			"match[]": []string{`test_metric1`},
			"start":   []string{"1"},
			"end":     []string{"100000"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: []labels.Labels{
				labels.FromStrings("__name__", "test_metric1", "foo", "bar", "namespace", "ns-a"),
			},
		},
	},
	"start before series, end within series": {
		Queries: url.Values{
			"match[]": []string{`test_metric1`},
			"start":   []string{"-1"},
			"end":     []string{"1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: []labels.Labels{
				labels.FromStrings("__name__", "test_metric1", "foo", "bar", "namespace", "ns-a"),
			},
		},
	},
}

var MyTokenSeriesScenarios = map[string]Scenario{
	"missing match[] query Params in series requests": {
		Queries:  url.Values{},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     "no match[] parameter provided",
		},
	},
	"bad match[] `invalid][query`": {
		Queries: url.Values{
			"match[]": []string{"invalid][query"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `invalid parameter "match[]": 1:8: parse error: unexpected right bracket ']'`,
		},
	},
	"test_metric1": {
		Queries: url.Values{
			"match[]": []string{"test_metric1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: []labels.Labels{
				labels.FromStrings("__name__", "test_metric1", "foo", "bar", "namespace", "ns-a"),
				labels.FromStrings("__name__", "test_metric1", "foo", "boo", "namespace", "ns-c"),
			},
		},
	},
	"test_metric1{namespace='ns-c'}": {
		Queries: url.Values{
			"match[]": []string{"test_metric1{namespace='ns-c'}"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: []labels.Labels{
				labels.FromStrings("__name__", "test_metric1", "foo", "boo", "namespace", "ns-c"),
			},
		},
	},
	"test_metric2{foo='boo'}": {
		Queries: url.Values{
			"match[]": []string{"test_metric2{foo='boo'}"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: []labels.Labels{
				labels.FromStrings("__name__", "test_metric2", "foo", "boo"),
			},
		},
	},
	"{foo='boo'}": {
		Queries: url.Values{
			"match[]": []string{"{foo='boo'}"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: []labels.Labels{
				labels.FromStrings("__name__", "test_metric1", "foo", "boo", "namespace", "ns-c"),
				labels.FromStrings("__name__", "test_metric2", "foo", "boo"),
			},
		},
	},
	"two matches": {
		Queries: url.Values{
			"match[]": []string{`test_metric1{foo=~".+o$"}`, `test_metric1{foo=~".+o"}`},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: []labels.Labels{
				labels.FromStrings("__name__", "test_metric1", "foo", "boo", "namespace", "ns-c"),
			},
		},
	},
	"two matches, but one is `none`": {
		Queries: url.Values{
			"match[]": []string{`test_metric2{foo=~".+o"}`, `none`},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: []labels.Labels{
				labels.FromStrings("__name__", "test_metric2", "foo", "boo"),
			},
		},
	},
	"test_metric_without_labels": {
		Queries: url.Values{
			"match[]": []string{"test_metric_without_labels"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: []labels.Labels{
				labels.FromStrings("__name__", "test_metric_without_labels"),
			},
		},
	},
	"does_not_match_anything": {
		Queries: url.Values{
			"match[]": []string{"does_not_match_anything"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"start and end before series starts": {
		Queries: url.Values{
			"match[]": []string{`test_metric1`},
			"start":   []string{"-2"},
			"end":     []string{"-1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"start and end after series ends": {
		Queries: url.Values{
			"match[]": []string{`test_metric1`},
			"start":   []string{"100000"},
			"end":     []string{"100001"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []labels.Labels{},
		},
	},
	"start and end within series": {
		Queries: url.Values{
			"match[]": []string{`test_metric1`},
			"start":   []string{"1"},
			"end":     []string{"100"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: []labels.Labels{
				labels.FromStrings("__name__", "test_metric1", "foo", "bar", "namespace", "ns-a"),
				labels.FromStrings("__name__", "test_metric1", "foo", "boo", "namespace", "ns-c"),
			},
		},
	},
	"start within series, end after": {
		Queries: url.Values{
			"match[]": []string{`test_metric1`},
			"start":   []string{"1"},
			"end":     []string{"100000"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: []labels.Labels{
				labels.FromStrings("__name__", "test_metric1", "foo", "bar", "namespace", "ns-a"),
				labels.FromStrings("__name__", "test_metric1", "foo", "boo", "namespace", "ns-c"),
			},
		},
	},
	"start before series, end within series": {
		Queries: url.Values{
			"match[]": []string{`test_metric1`},
			"start":   []string{"-1"},
			"end":     []string{"1"},
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: []labels.Labels{
				labels.FromStrings("__name__", "test_metric1", "foo", "bar", "namespace", "ns-a"),
				labels.FromStrings("__name__", "test_metric1", "foo", "boo", "namespace", "ns-c"),
			},
		},
	},
}

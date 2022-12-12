//go:build test

package samples

import (
	"net/http"
	"net/url"
)

var NoneNamespacesTokenFederateScenarios = map[string]Scenario{
	"empty": {
		Queries:  url.Values{},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"match nothing": {
		Queries: url.Values{
			"match[]": []string{"does_not_match_anything"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"invalid Params from the beginning": {
		Queries: url.Values{
			"match[]": []string{"-not-a-valid-metric-name"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: `parse error at char 1: vector selector must contain label matchers or metric name
`,
	},
	"invalid Params somewhere in the middle": {
		Queries: url.Values{
			"match[]": []string{"not-a-valid-metric-name"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: `parse error at char 4: could not parse remaining input "-a-valid-metric"...
`,
	},
	"test_metric1": {
		Queries: url.Values{
			"match[]": []string{"test_metric1"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"test_metric2": {
		Queries: url.Values{
			"match[]": []string{"test_metric2"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"test_metric_without_labels": {
		Queries: url.Values{
			"match[]": []string{"test_metric_without_labels"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"test_stale_metric": {
		Queries: url.Values{
			"match[]": []string{"test_metric_stale"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"test_old_metric": {
		Queries: url.Values{
			"match[]": []string{"test_metric_old"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"{foo='boo'}": {
		Queries: url.Values{
			"match[]": []string{"{foo='boo'}"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"{namespace='ns-c'}": {
		Queries: url.Values{
			"match[]": []string{"{namespace='ns-c'}"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"two matchers": {
		Queries: url.Values{
			"match[]": []string{"test_metric1", "test_metric2"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"everything": {
		Queries: url.Values{
			"match[]": []string{"{__name__=~'.+'}"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"empty existing label value matches everything that doesn't have that label": {
		Queries: url.Values{
			"match[]": []string{"{foo='',__name__=~'.+'}"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"empty none-existing label value matches everything": {
		Queries: url.Values{
			"match[]": []string{"{bar='',__name__=~'.+'}"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"empty `namespace` label value matches everything": {
		Queries: url.Values{
			"match[]": []string{"{namespace='',__name__=~'.+'}"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
}

var SomeNamespacesTokenFederateScenarios = map[string]Scenario{
	"empty": {
		Queries:  url.Values{},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"match nothing": {
		Queries: url.Values{
			"match[]": []string{"does_not_match_anything"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"invalid Params from the beginning": {
		Queries: url.Values{
			"match[]": []string{"-not-a-valid-metric-name"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: `parse error at char 1: vector selector must contain label matchers or metric name
`,
	},
	"invalid Params somewhere in the middle": {
		Queries: url.Values{
			"match[]": []string{"not-a-valid-metric-name"},
		},
		RespCode: http.StatusBadRequest,
		RespBody: `parse error at char 4: could not parse remaining input "-a-valid-metric"...
`,
	},
	"test_metric1": {
		Queries: url.Values{
			"match[]": []string{"test_metric1"},
		},
		RespCode: http.StatusOK,
		RespBody: `# TYPE test_metric1 untyped
test_metric1{foo="bar",namespace="ns-a",instance="",prometheus="cluster-level/test"} 10000 6000000
`,
	},
	"test_metric2": {
		Queries: url.Values{
			"match[]": []string{"test_metric2"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"test_metric_without_labels": {
		Queries: url.Values{
			"match[]": []string{"test_metric_without_labels"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"test_stale_metric": {
		Queries: url.Values{
			"match[]": []string{"test_metric_stale"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"test_old_metric": {
		Queries: url.Values{
			"match[]": []string{"test_metric_old"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"{foo='boo'}": {
		Queries: url.Values{
			"match[]": []string{"{foo='boo'}"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"{namespace='ns-c'}": {
		Queries: url.Values{
			"match[]": []string{"{namespace='ns-c'}"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"two matchers": {
		Queries: url.Values{
			"match[]": []string{"test_metric1", "test_metric2"},
		},
		RespCode: http.StatusOK,
		RespBody: `# TYPE test_metric1 untyped
test_metric1{foo="bar",namespace="ns-a",instance="",prometheus="cluster-level/test"} 10000 6000000
`,
	},
	"everything": {
		Queries: url.Values{
			"match[]": []string{"{__name__=~'.+'}"},
		},
		RespCode: http.StatusOK,
		RespBody: `# TYPE test_metric1 untyped
test_metric1{foo="bar",namespace="ns-a",instance="",prometheus="cluster-level/test"} 10000 6000000
`,
	},
	"empty existing label value matches everything that doesn't have that label": {
		Queries: url.Values{
			"match[]": []string{"{foo='',__name__=~'.+'}"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
	"empty none-existing label value matches everything": {
		Queries: url.Values{
			"match[]": []string{"{bar='',__name__=~'.+'}"},
		},
		RespCode: http.StatusOK,
		RespBody: `# TYPE test_metric1 untyped
test_metric1{foo="bar",namespace="ns-a",instance="",prometheus="cluster-level/test"} 10000 6000000
`,
	},
	"empty `namespace` label value matches everything that doesn't have `namespace` label": {
		Queries: url.Values{
			"match[]": []string{"{namespace='',__name__=~'.+'}"},
		},
		RespCode: http.StatusOK,
		RespBody: ``,
	},
}

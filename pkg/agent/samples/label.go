//go:build test

package samples

import (
	"net/http"
)

var NoneNamespacesTokenLabelScenarios = map[string]Scenario{
	"bad value `invalid][query`": {
		Params: map[string]string{
			"name": "invalid][query",
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `invalid label name: "invalid][query"`,
		},
	},
	"__name__": {
		Params: map[string]string{
			"name": "__name__",
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []string{},
		},
	},
	"namespace": {
		Params: map[string]string{
			"name": "namespace",
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []string{},
		},
	},
	"foo": {
		Params: map[string]string{
			"name": "foo",
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: []string{
				"bar",
				"boo",
			},
		},
	},
	"does_not_match_anything": {
		Params: map[string]string{
			"name": "does_not_match_anything",
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []string{},
		},
	},
	"test_metric_without_labels": {
		Params: map[string]string{
			"name": "test_metric_without_labels",
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []string{},
		},
	},
}

var SomeNamespacesTokenLabelScenarios = map[string]Scenario{
	"bad value `invalid][query`": {
		Params: map[string]string{
			"name": "invalid][query",
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `invalid label name: "invalid][query"`,
		},
	},
	"__name__": {
		Params: map[string]string{
			"name": "__name__",
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: []string{
				"test_metric1",
			},
		},
	},
	"namespace": {
		Params: map[string]string{
			"name": "namespace",
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: []string{
				"ns-a",
				"ns-b",
			},
		},
	},
	"foo": {
		Params: map[string]string{
			"name": "foo",
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: []string{
				"bar",
				"boo",
			},
		},
	},
	"does_not_match_anything": {
		Params: map[string]string{
			"name": "does_not_match_anything",
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []string{},
		},
	},
	"test_metric_without_labels": {
		Params: map[string]string{
			"name": "test_metric_without_labels",
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []string{},
		},
	},
}

var MyTokenLabelScenarios = map[string]Scenario{
	"bad value `invalid][query`": {
		Params: map[string]string{
			"name": "invalid][query",
		},
		RespCode: http.StatusBadRequest,
		RespBody: &jsonResponseData{
			Status:    "error",
			ErrorType: "bad_data",
			Error:     `invalid label name: "invalid][query"`,
		},
	},
	"__name__": {
		Params: map[string]string{
			"name": "__name__",
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: []string{
				"test_metric1",
				"test_metric2",
				"test_metric_old",
				"test_metric_stale",
				"test_metric_without_labels",
			},
		},
	},
	"namespace": {
		Params: map[string]string{
			"name": "namespace",
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: []string{
				"ns-a",
				"ns-c",
			},
		},
	},
	"foo": {
		Params: map[string]string{
			"name": "foo",
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data: []string{
				"bar",
				"boo",
			},
		},
	},
	"does_not_match_anything": {
		Params: map[string]string{
			"name": "does_not_match_anything",
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []string{},
		},
	},
	"test_metric_without_labels": {
		Params: map[string]string{
			"name": "test_metric_without_labels",
		},
		RespCode: http.StatusOK,
		RespBody: &jsonResponseData{
			Status: "success",
			Data:   []string{},
		},
	},
}

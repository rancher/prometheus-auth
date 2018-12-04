package agent

import (
	"context"
	"fmt"
	"github.com/json-iterator/go"
	"github.com/juju/errors"
	promapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prommodel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
	"github.com/prometheus/prometheus/promql"
	"github.com/rancher/prometheus-auth/pkg/kubeauth"
	"github.com/rancher/prometheus-auth/pkg/prommetric"
	"github.com/rancher/prometheus-auth/pkg/utils"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	namespaceMatchName = "namespace"
	allLabelsMatchName = "__name__"

	queryEndpoint  = v1Api + "/query"
	seriesEndpoint = v1Api + "/series"
	readEndpoint   = v1Api + "/read"
	adminEndpoint  = v1Api + "/admin"
	labelEndpoint  = v1Api + "/label"
)

func (a *agent) wrapBackend(backend http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		api := createApi(req)
		reqMethod := api.getMethod()
		utils.LogTrace(func() string {
			return api.String()
		})

		if api.is(v1Api) {
			me := a.cfg.hostAccessToken

			if who := api.whoami(); who != me {
				if api.is(adminEndpoint) || http.MethodDelete == reqMethod {
					http.Error(resp, "unable to access", http.StatusUnauthorized)
					return
				}

				on := a.globalNamespacesOwnedView.Own(who)
				utils.LogTrace(func() string {
					return on.String()
				})

				if !on.HasAll() {
					var (
						value interface{}
						err   error
					)

					switch reqMethod {
					case http.MethodGet:
						if api.is(queryEndpoint) {
							value, err = api.getQueryValue()
							if err == nil {
								err = hijackQueryApi(value.(*queryValue), on)
							}
						} else if api.is(seriesEndpoint) {
							value, err = api.getMatchValues()
							if err == nil {
								err = hijackSeriesApi(value.(*matchValues), on)
							}
						} else if api.is(labelEndpoint) {
							// only hijack /label/namespace/values, /label/__name__/values
							if api.is(labelEndpoint+"/"+namespaceMatchName, labelEndpoint+"/"+allLabelsMatchName) {
								value, err = api.getLabelValue()
								if err == nil {
									err = hijackLabelApi(value.(string), on, a.cfg.ctx, a.backendHTTPApi, resp)
								}

								if err == nil {
									return
								}
							}
						}
					case http.MethodPost:
						if api.is(queryEndpoint) {
							value, err = api.getQueryValue()
							if err == nil {
								err = hijackQueryApi(value.(*queryValue), on)
							}
						} else if api.is(readEndpoint) {
							value, err = api.getQueryValues()
							if err == nil {
								err = hijackReadApi(value.(*queryValues), on)
							}
						}
					}

					if err != nil {
						log.Error(errors.ErrorStack(err))

						http.Error(resp, "failed to access", http.StatusInternalServerError)
						return
					}
				}
			}
		}

		backend.ServeHTTP(resp, req)
	})
}

func hijackQueryApi(queryValue *queryValue, on *kubeauth.OwnedNamespaces) error {
	labelFilter := prommetric.CreateLabelMatchersNameFilter(nil, map[string]prommetric.LabelMatcherTranslator{
		namespaceMatchName: prommetric.CreateLabelMatcherTranslator(namespaceMatchName, on.ToSetView()),
	})

	return queryValue.hijack(func(queryFormValue string) (string, error) {
		queryExpr, err := promql.ParseExpr(queryFormValue)
		if err != nil {
			return "", errors.Annotatef(err, "failed to parse expr from %s", queryFormValue)
		}
		promql.Inspect(queryExpr, func(node promql.Node, _ []promql.Node) error {
			switch n := node.(type) {
			case *promql.VectorSelector:
				n.LabelMatchers = labelFilter.Filter(n.LabelMatchers)
			case *promql.MatrixSelector:
				n.LabelMatchers = labelFilter.Filter(n.LabelMatchers)
			}
			return nil
		})

		return queryExpr.String(), nil
	})
}

func hijackSeriesApi(matchValues *matchValues, on *kubeauth.OwnedNamespaces) error {
	labelFilter := prommetric.CreateLabelMatchersNameFilter(nil, map[string]prommetric.LabelMatcherTranslator{
		namespaceMatchName: prommetric.CreateLabelMatcherTranslator(namespaceMatchName, on.ToSetView()),
	})

	return matchValues.hijack(func(matchFormValue string) (string, error) {
		matchExpr, err := promql.ParseExpr(matchFormValue)
		if err != nil {
			return "", errors.Annotatef(err, "failed to parse expr from %s", matchFormValue)
		}
		promql.Inspect(matchExpr, func(node promql.Node, _ []promql.Node) error {
			switch n := node.(type) {
			case *promql.VectorSelector:
				n.LabelMatchers = labelFilter.Filter(n.LabelMatchers)
			case *promql.MatrixSelector:
				n.LabelMatchers = labelFilter.Filter(n.LabelMatchers)
			}
			return nil
		})

		return matchExpr.String(), nil
	})
}

func hijackReadApi(queryValues *queryValues, on *kubeauth.OwnedNamespaces) error {
	labelFilter := prommetric.CreatePromPbLabelMatchersNameFilter(nil, map[string]prommetric.PromPbLabelMatcherTranslator{
		namespaceMatchName: prommetric.CreatePromPbLabelMatcherTranslator(namespaceMatchName, on.ToSetView()),
	})

	return queryValues.hijack(func(query *prompb.Query) (*prompb.Query, error) {
		query.Matchers = labelFilter.Filter(query.GetMatchers())

		return query, nil
	})
}

func hijackLabelApi(labelValue string, on *kubeauth.OwnedNamespaces, ctx context.Context, backendHTTPApi promapiv1.API, resp http.ResponseWriter) error {
	hjkValue := make(prommodel.LabelValues, 0)

	allOwnedNamespaces := on.Values()
	if len(allOwnedNamespaces) != 0 {
		switch labelValue {
		case namespaceMatchName:
			hjkValue = make(prommodel.LabelValues, 0, len(allOwnedNamespaces))
			for _, v := range allOwnedNamespaces {
				hjkValue = append(hjkValue, prommodel.LabelValue(v))
			}

		case allLabelsMatchName:
			expr := fmt.Sprintf(`count by (%s) ({namespace=~"%s"})`, allLabelsMatchName, strings.Join(allOwnedNamespaces, "|"))
			vals, err := backendHTTPApi.Query(ctx, expr, time.Now())
			if err != nil {
				return errors.Annotatef(err, "unable to query %s", expr)
			}

			vectorVals, ok := vals.(prommodel.Vector)
			if !ok {
				return errors.Annotatef(err, "%s returns none vector type", expr)
			}

			hjkValue = make(prommodel.LabelValues, 0, len(vectorVals))
			for _, vectorVal := range vectorVals {
				valLabelSet := prommodel.LabelSet(vectorVal.Metric)
				hjkValue = append(hjkValue, valLabelSet[allLabelsMatchName])
			}
		}

		sort.Sort(hjkValue)
	}

	json := jsoniter.ConfigCompatibleWithStandardLibrary
	hjkValueJSONBytes, err := json.Marshal(struct {
		Status string      `json:"status"`
		Data   interface{} `json:"data,omitempty"`
	}{
		Status: "success",
		Data:   hjkValue,
	})
	if err != nil {
		return errors.Annotate(err, "unable to marshal to json")
	}

	resp.Header().Set("Content-Type", "application/json")
	resp.WriteHeader(http.StatusOK)
	if n, err := resp.Write(hjkValueJSONBytes); err != nil {
		return errors.Annotatef(err, "failed to write response, bytesWritten %d", n)
	}

	return nil
}

// +build test

package agent

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/cockroachdb/cockroach/pkg/util/httputil"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/json-iterator/go"
	promapi "github.com/prometheus/client_golang/api"
	promapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/prompb"
	"github.com/prometheus/prometheus/promql"
	"github.com/prometheus/prometheus/util/testutil"
	promweb "github.com/prometheus/prometheus/web"
	promtsdb "github.com/prometheus/tsdb"
	"github.com/rancher/prometheus-auth/pkg/agent/samples"
	"github.com/rancher/prometheus-auth/pkg/data"
	"github.com/rancher/prometheus-auth/pkg/kube"
)

func Test_accessControl(t *testing.T) {
	// all namespaceSet : ns-a, ns-b, ns-c
	suite, err := promql.NewTest(t, `
		load 1m
			test_metric1{namespace="ns-a",foo="bar"}    	0+100x100
			test_metric1{namespace="ns-c",foo="boo"}    	1+0x100
			test_metric2{foo="boo"}    						1+0x100
			test_metric_without_labels 						1+10x100
			test_metric_stale                      	 		1+10x99 stale
			test_metric_old                         		1+10x98
	`)
	if err != nil {
		t.Fatal(err)
	}
	defer suite.Close()

	if err := suite.Run(); err != nil {
		t.Fatal(err)
	}

	dbDir, err := ioutil.TempDir("", "tsdb-ready")
	defer os.RemoveAll(dbDir)

	testutil.Ok(t, err)

	db, err := promtsdb.Open(dbDir, nil, nil, nil)

	testutil.Ok(t, err)

	webHandler := promweb.New(nil, &promweb.Options{
		Context:        context.Background(),
		ListenAddress:  ":9090",
		ReadTimeout:    30 * time.Second,
		MaxConnections: 512,
		Storage:        suite.Storage(),
		QueryEngine:    suite.QueryEngine(),
		ScrapeManager:  nil,
		RuleManager:    nil,
		Notifier:       nil,
		RoutePrefix:    "/",
		EnableAdminAPI: true,
		ExternalURL: &url.URL{
			Scheme: "http",
			Host:   "localhost:9090",
			Path:   "/",
		},
		TSDB:    func() *promtsdb.DB { return db },
		Version: &promweb.PrometheusVersion{},
		Flags:   map[string]string{},
	})
	defer webHandler.Quit()

	err = webHandler.ApplyConfig(&config.Config{
		GlobalConfig: config.GlobalConfig{
			ExternalLabels: model.LabelSet{
				"prometheus": "cluster-level/test",
			},
		},
	})
	if err != nil {
		t.Error(err)
	}

	// modify the `now` field
	refVal := reflect.ValueOf(webHandler).Elem()
	nowMemberVal := refVal.FieldByName("now")
	ptrToNow := unsafe.Pointer(nowMemberVal.UnsafeAddr())
	realPtrToNow := (*func() model.Time)(ptrToNow)
	*realPtrToNow = func() model.Time { return model.Time(101 * 60 * 1000) } // 101min, federation has 5 min `LookbackDelta`
	apiV1MemberVal := refVal.FieldByName("apiV1").Elem()
	nowMemberVal2 := apiV1MemberVal.FieldByName("now")
	ptrToNow2 := unsafe.Pointer(nowMemberVal2.UnsafeAddr())
	realPtrToNow2 := (*func() time.Time)(ptrToNow2)
	*realPtrToNow2 = func() time.Time { return model.Time(0).Time() }

	startPrometheusWebHandler(t, webHandler)

	agt := mockAgent(t)
	httpBackend := agt.httpBackend()

	func() {
		t.Log("federate testing begin ...")

		tokenScenariosMap := map[string]map[string]samples.Scenario{
			"noneNamespacesToken": samples.NoneNamespacesTokenFederateScenarios,
			"someNamespacesToken": samples.SomeNamespacesTokenFederateScenarios,
		}

		for token, tokenScenarios := range tokenScenariosMap {
			for name, tokenScenario := range tokenScenarios {
				req := httptest.NewRequest("GET", "http://example.org/federate?"+tokenScenario.Queries.Encode(), nil)
				req.Header.Set(authorizationHeaderKey, fmt.Sprintf("Bearer %s", token))
				res := httptest.NewRecorder()
				httpBackend.ServeHTTP(res, req)
				if got, want := res.Code, tokenScenario.RespCode; got != want {
					t.Errorf("[federate] [GET ] token %q scenario %q: got code %d, want %d", token, name, got, want)
				}
				if got, want := normalizeResponseBody(res.Body), tokenScenario.RespBody; got != want {
					t.Errorf("[federate] [GET ] token %q scenario %q: got body\n%s\n, want\n%s\n", token, name, got, want)
				}
			}

		}

		t.Log("...federate testing end")
	}()

	func() {
		t.Log("label testing begin ...")

		tokenScenariosMap := map[string]map[string]samples.Scenario{
			"noneNamespacesToken": samples.NoneNamespacesTokenLabelScenarios,
			"someNamespacesToken": samples.SomeNamespacesTokenLabelScenarios,
		}

		for token, tokenScenarios := range tokenScenariosMap {
			for name, tokenScenario := range tokenScenarios {
				req := httptest.NewRequest("GET", "http://example.org/api/v1/label/"+tokenScenario.Params["name"]+"/values", nil)
				req.Header.Set(authorizationHeaderKey, fmt.Sprintf("Bearer %s", token))
				res := httptest.NewRecorder()
				httpBackend.ServeHTTP(res, req)
				if got, want := res.Code, tokenScenario.RespCode; got != want {
					t.Errorf("[label] [GET ] token %q scenario %q: got code %d, want %d", token, name, got, want)
				}
				if got, want := string(res.Body.Bytes()), jsonResponseBody(tokenScenario.RespBody); got != want {
					t.Errorf("[label] [GET ] token %q scenario %q: got body\n%s\n, want\n%s\n", token, name, got, want)
				}
			}

		}

		t.Log("...label testing end")
	}()

	func() {
		t.Log("query testing begin ...")

		tokenScenariosMap := map[string]map[string]samples.Scenario{
			"noneNamespacesToken": samples.NoneNamespacesTokenQueryScenarios,
			"someNamespacesToken": samples.SomeNamespacesTokenQueryScenarios,
		}

		for token, tokenScenarios := range tokenScenariosMap {
			for name, tokenScenario := range tokenScenarios {
				// GET
				func(tokenScenario *samples.Scenario, token string, name string) {
					req := httptest.NewRequest("GET", "http://example.org/api/v1"+tokenScenario.Endpoint+"?"+tokenScenario.Queries.Encode(), nil)
					req.Header.Set(authorizationHeaderKey, fmt.Sprintf("Bearer %s", token))
					res := httptest.NewRecorder()
					httpBackend.ServeHTTP(res, req)
					if got, want := res.Code, tokenScenario.RespCode; got != want {
						t.Errorf("[query] [GET ] token %q scenario %q: got code %d, want %d", token, name, got, want)
					}
					if got, want := string(res.Body.Bytes()), jsonResponseBody(tokenScenario.RespBody); got != want {
						t.Errorf("[query] [GET ] token %q scenario %q: got body\n%s\n, want\n%s\n", token, name, got, want)
					}
				}(&tokenScenario, token, name)

				// POST
				func(tokenScenario *samples.Scenario, token string, name string) {
					req := httptest.NewRequest("POST", "http://example.org/api/v1"+tokenScenario.Endpoint, strings.NewReader(tokenScenario.Queries.Encode()))
					req.Header.Set(httputil.ContentTypeHeader, "application/x-www-form-urlencoded")
					req.Header.Set(authorizationHeaderKey, fmt.Sprintf("Bearer %s", token))
					res := httptest.NewRecorder()
					httpBackend.ServeHTTP(res, req)
					if got, want := res.Code, tokenScenario.RespCode; got != want {
						t.Errorf("[query] [POST] token %q scenario %q: got code %d, want %d", token, name, got, want)
					}
					if got, want := string(res.Body.Bytes()), jsonResponseBody(tokenScenario.RespBody); got != want {
						t.Errorf("[query] [POST] token %q scenario %q: got body\n%s\n, want\n%s\n", token, name, got, want)
					}
				}(&tokenScenario, token, name)
			}

		}

		t.Log("...query testing end")
	}()

	func() {
		t.Log("read testing begin ...")

		tokenScenariosMap := map[string]map[string]samples.Scenario{
			"noneNamespacesToken": samples.NoneNamespacesTokenReadScenarios(t),
			"someNamespacesToken": samples.SomeNamespacesTokenReadScenarios(t),
		}

		for token, tokenScenarios := range tokenScenariosMap {
			for name, tokenScenario := range tokenScenarios {
				// raw -> proto request
				protoReq := &prompb.ReadRequest{Queries: tokenScenario.PrompbQueries}
				protoReqData, err := proto.Marshal(protoReq)
				if err != nil {
					t.Fatal(err)
				}
				compressedProtoReqData := snappy.Encode(nil, protoReqData)

				req := httptest.NewRequest("POST", "http://example.org/api/v1/read", bytes.NewBuffer(compressedProtoReqData))
				req.Header.Set(authorizationHeaderKey, fmt.Sprintf("Bearer %s", token))
				res := httptest.NewRecorder()
				httpBackend.ServeHTTP(res, req)

				if got, want := res.Code, tokenScenario.RespCode; got != want {
					t.Errorf("[read] [POST] token %q scenario %q: got code %d, want %d", token, name, got, want)
				}

				// proto response -> raw
				compressedProtoResData, err := ioutil.ReadAll(res.Body)
				if err != nil {
					t.Fatal(err)
				}
				protoResData, err := snappy.Decode(nil, compressedProtoResData)
				if err != nil {
					t.Fatal(err)
				}
				var protoRes prompb.ReadResponse
				err = proto.Unmarshal(protoResData, &protoRes)
				if err != nil {
					t.Fatal(err)
				}

				if got, want := protoRes.Results, tokenScenario.RespBody; !reflect.DeepEqual(got, want) {
					t.Errorf("[read] [POST] token %q scenario %q: got body\n%v\n, want\n%v\n", token, name, got, want)
				}
			}

		}

		t.Log("...read testing end")
	}()

	func() {
		t.Log("series testing begin ...")

		tokenScenariosMap := map[string]map[string]samples.Scenario{
			"noneNamespacesToken": samples.NoneNamespacesTokenSeriesScenarios,
			"someNamespacesToken": samples.SomeNamespacesTokenSeriesScenarios,
		}

		for token, tokenScenarios := range tokenScenariosMap {
			for name, tokenScenario := range tokenScenarios {
				req := httptest.NewRequest("GET", "http://example.org/api/v1/series?"+tokenScenario.Queries.Encode(), nil)
				req.Header.Set(authorizationHeaderKey, fmt.Sprintf("Bearer %s", token))
				res := httptest.NewRecorder()
				httpBackend.ServeHTTP(res, req)
				if got, want := res.Code, tokenScenario.RespCode; got != want {
					t.Errorf("[series] [GET] token %q scenario %q: got code %d, want %d", token, name, got, want)
				}
				if got, want := string(res.Body.Bytes()), jsonResponseBody(tokenScenario.RespBody); got != want {
					t.Errorf("[series] [GET] token %q scenario %q: got body\n%s\n, want\n%s\n", token, name, got, want)
				}
			}

		}

		t.Log("...series testing end")
	}()
}

func startPrometheusWebHandler(t *testing.T, webHandler *promweb.Handler) {
	go func() {
		err := webHandler.Run(context.Background())
		if err != nil {
			panic(fmt.Sprintf("Can't start web handler:%s", err))
		}
	}()

	time.Sleep(5 * time.Second)

	resp, err := http.Get("http://localhost:9090/-/healthy")

	testutil.Ok(t, err)
	testutil.Equals(t, http.StatusOK, resp.StatusCode)

	resp, err = http.Get("http://localhost:9090/-/ready")

	testutil.Ok(t, err)
	testutil.Equals(t, http.StatusServiceUnavailable, resp.StatusCode)

	resp, err = http.Get("http://localhost:9090/version")

	testutil.Ok(t, err)
	testutil.Equals(t, http.StatusServiceUnavailable, resp.StatusCode)

	resp, err = http.Get("http://localhost:9090/graph")

	testutil.Ok(t, err)
	testutil.Equals(t, http.StatusServiceUnavailable, resp.StatusCode)

	resp, err = http.Post("http://localhost:9090/api/v2/admin/tsdb/snapshot", "", strings.NewReader(""))

	testutil.Ok(t, err)
	testutil.Equals(t, http.StatusServiceUnavailable, resp.StatusCode)

	resp, err = http.Post("http://localhost:9090/api/v2/admin/tsdb/delete_series", "", strings.NewReader("{}"))

	testutil.Ok(t, err)
	testutil.Equals(t, http.StatusServiceUnavailable, resp.StatusCode)

	// Set to ready.
	webHandler.Ready()

	resp, err = http.Get("http://localhost:9090/-/healthy")

	testutil.Ok(t, err)
	testutil.Equals(t, http.StatusOK, resp.StatusCode)

	resp, err = http.Get("http://localhost:9090/-/ready")

	testutil.Ok(t, err)
	testutil.Equals(t, http.StatusOK, resp.StatusCode)

	resp, err = http.Get("http://localhost:9090/version")

	testutil.Ok(t, err)
	testutil.Equals(t, http.StatusOK, resp.StatusCode)

	resp, err = http.Get("http://localhost:9090/graph")

	testutil.Ok(t, err)
	testutil.Equals(t, http.StatusOK, resp.StatusCode)

	resp, err = http.Post("http://localhost:9090/api/v2/admin/tsdb/snapshot", "", strings.NewReader(""))

	testutil.Ok(t, err)
	testutil.Equals(t, http.StatusOK, resp.StatusCode)

	resp, err = http.Post("http://localhost:9090/api/v2/admin/tsdb/delete_series", "", strings.NewReader("{}"))

	testutil.Ok(t, err)
	testutil.Equals(t, http.StatusOK, resp.StatusCode)
}

func mockAgent(t *testing.T) *agent {
	proxyURL, err := url.Parse("http://localhost:9090")
	if err != nil {
		t.Error(err)
	}

	agtCfg := &agentConfig{
		ctx:      context.Background(),
		myToken:  "myToken",
		proxyURL: proxyURL,
		filterReaderLabelSet: data.NewSet(
			"prometheus",
			"prometheus_replica",
		),
	}

	// create Prometheus client
	promClient, err := promapi.NewClient(promapi.Config{
		Address: proxyURL.String(),
	})
	if err != nil {
		t.Error(err)
	}

	return &agent{
		cfg:        agtCfg,
		namespaces: mockOwnedNamespaces(),
		remoteAPI:  promapiv1.NewAPI(promClient),
	}
}

func normalizeResponseBody(body *bytes.Buffer) string {
	var (
		lines    []string
		lastHash int
	)
	for line, err := body.ReadString('\n'); err == nil; line, err = body.ReadString('\n') {
		if line[0] == '#' && len(lines) > 0 {
			sort.Strings(lines[lastHash+1:])
			lastHash = len(lines)
		}
		lines = append(lines, line)
	}
	if len(lines) > 0 {
		sort.Strings(lines[lastHash+1:])
	}
	return strings.Join(lines, "")
}

func jsonResponseBody(body interface{}) string {
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	respBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error())
	}

	return string(respBytes)
}

type fakeOwnedNamespaces struct {
	token2Namespaces map[string]data.Set
}

func (f *fakeOwnedNamespaces) Query(token string) data.Set {
	return f.token2Namespaces[token]
}

func mockOwnedNamespaces() kube.Namespaces {
	return &fakeOwnedNamespaces{
		token2Namespaces: map[string]data.Set{
			"noneNamespacesToken": {},
			"someNamespacesToken": data.NewSet("ns-a", "ns-b"),
		},
	}
}

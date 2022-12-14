package agent

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/cockroachdb/cockroach/pkg/util/httputil"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/juju/errors"
	promapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	promgo "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/rancher/prometheus-auth/pkg/data"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/runtime"
)

const (
	apiContextKey = "_apiContext_"
)

var (
	badRequestErr     = errors.BadRequestf("bad_data")
	notProvisionedErr = errors.NotProvisionedf("execution")
	internalErr       = errors.New("internal")
)

type apiContext struct {
	sync.Once
	tag                  string
	response             http.ResponseWriter
	request              *http.Request
	proxyHandler         http.Handler
	filterReaderLabelSet data.Set
	namespaceSet         data.Set
	remoteAPI            promapiv1.API
}

type jsonResponseData struct {
	Status    string      `json:"status"`
	Data      interface{} `json:"data,omitempty"`
	ErrorType string      `json:"errorType,omitempty"`
	Error     string      `json:"error,omitempty"`
}

func (c *apiContext) responseJSON(data interface{}) (err error) {
	c.Do(func() {
		resp := c.response
		resp.Header().Set(httputil.ContentTypeHeader, httputil.JSONContentType)

		responseData := &jsonResponseData{
			Status: "success",
			Data:   data,
		}

		respBytes, marshalErr := json.Marshal(responseData)
		if marshalErr != nil {
			err = errors.Wrap(marshalErr, internalErr)
			return
		}

		if _, writeErr := resp.Write(respBytes); writeErr != nil {
			err = errors.Wrap(writeErr, internalErr)
		}
	})

	return
}

func (c *apiContext) responseProto(data proto.Message) (err error) {
	c.Do(func() {
		resp := c.response
		resp.Header().Set(httputil.ContentTypeHeader, httputil.ProtoContentType)
		resp.Header().Set(httputil.ContentEncodingHeader, "snappy")

		if data == nil {
			resp.WriteHeader(http.StatusNoContent)
			return
		}

		responseData, marshalErr := proto.Marshal(data)
		if marshalErr != nil {
			err = errors.Wrap(marshalErr, internalErr)
			return
		}

		respBytes := snappy.Encode(nil, responseData)
		if _, writeErr := resp.Write(respBytes); writeErr != nil {
			err = errors.Wrap(writeErr, internalErr)
		}
	})

	return
}

func (c *apiContext) responseMetrics(data *promgo.MetricFamily) (err error) {
	c.Do(func() {
		req, resp := c.request, c.response

		respFormat := expfmt.Negotiate(req.Header)
		respEncoder := expfmt.NewEncoder(resp, respFormat)
		resp.Header().Set(httputil.ContentTypeHeader, string(respFormat))

		if data == nil {
			return
		}

		if encodeErr := respEncoder.Encode(data); encodeErr != nil {
			err = errors.Wrap(encodeErr, internalErr)
		}
	})

	return
}

func (c *apiContext) proxy() error {
	c.Do(func() {
		c.proxyHandler.ServeHTTP(c.response, c.request)
	})

	return nil
}

func (c *apiContext) proxyWith(request *http.Request) error {
	c.Do(func() {
		c.proxyHandler.ServeHTTP(c.response, request)
	})

	return nil
}

type apiContextHandler func(*apiContext) error

func (f apiContextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer runtime.HandleCrash(func(_ interface{}) {
		http.Error(w, "unknown internal error", http.StatusInternalServerError)
	})

	apiCtx := r.Context().Value(apiContextKey).(*apiContext)

	err := f(apiCtx)
	if err == nil {
		return
	}

	log.Debug(errors.ErrorStack(err))

	// response error msg
	causeErrMsg := ""
	switch e := err.(type) {
	case *errors.Err:
		causeErrMsg = e.Underlying().Error()
	default:
		causeErrMsg = err.Error()
	}

	responseErrType := ""
	responseCode := http.StatusInternalServerError
	if errors.IsBadRequest(err) {
		responseCode = http.StatusBadRequest
		responseErrType = "bad_data"
	} else if errors.IsNotProvisioned(err) {
		responseCode = http.StatusUnprocessableEntity
		responseErrType = "execution"
	}

	acceptHeaderValue := r.Header.Get(httputil.AcceptHeader)
	contentTypeHeaderValue := w.Header().Get(httputil.ContentTypeHeader)
	if !strings.Contains(acceptHeaderValue, httputil.JSONContentType) &&
		!strings.EqualFold(contentTypeHeaderValue, httputil.JSONContentType) {

		http.Error(w, causeErrMsg, responseCode)
		return
	}

	responseData := &jsonResponseData{
		Status:    "error",
		ErrorType: responseErrType,
		Error:     causeErrMsg,
	}

	respBytes, marshalErr := json.Marshal(responseData)
	if marshalErr != nil {
		log.WithError(err).Errorf("unable to marshal responseData %#v", responseData)
		http.Error(w, "internal error", http.StatusInternalServerError)

		return
	}

	w.Header().Set(httputil.ContentTypeHeader, httputil.JSONContentType)
	w.WriteHeader(responseCode)
	if _, writeErr := w.Write(respBytes); writeErr != nil {
		log.WithError(err).Errorf("failed to write %q into http response", string(respBytes))
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

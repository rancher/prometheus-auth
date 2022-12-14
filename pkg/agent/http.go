package agent

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/juju/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rancher/prometheus-auth/pkg/kube"
	log "github.com/sirupsen/logrus"
	authentication "k8s.io/api/authentication/v1"
)

func (a *agent) httpBackend() http.Handler {
	proxy := httputil.NewSingleHostReverseProxy(a.cfg.proxyURL)
	router := mux.NewRouter()

	if log.GetLevel() == log.DebugLevel {
		router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
				log.Debugf("%s - %s", req.Method, req.URL.Path)

				next.ServeHTTP(resp, req)
			})
		})
	}

	// enable metrics
	router.Path("/_/metrics").Methods("GET").Handler(promhttp.Handler())

	// proxy white list
	router.Path("/alerts").Methods("GET").Handler(proxy)
	router.Path("/graph").Methods("GET").Handler(proxy)
	router.Path("/status").Methods("GET").Handler(proxy)
	router.Path("/flags").Methods("GET").Handler(proxy)
	router.Path("/config").Methods("GET").Handler(proxy)
	router.Path("/rules").Methods("GET").Handler(proxy)
	router.Path("/targets").Methods("GET").Handler(proxy)
	router.Path("/version").Methods("GET").Handler(proxy)
	router.Path("/service-discovery").Methods("GET").Handler(proxy)
	router.PathPrefix("/consoles/").Methods("GET").Handler(proxy)
	router.PathPrefix("/static/").Methods("GET").Handler(proxy)
	router.PathPrefix("/user/").Methods("GET").Handler(proxy)
	router.Path("/metrics").Methods("GET").Handler(proxy)
	router.Path("/-/healthy").Methods("GET").Handler(proxy)
	router.Path("/-/ready").Methods("GET").Handler(proxy)
	router.PathPrefix("/debug/").Methods("GET").Handler(proxy)

	// access control
	router.PathPrefix("/").Handler(accessControl(a, proxy))

	return router
}

func accessControl(agt *agent, proxyHandler http.Handler) http.Handler {
	router := mux.NewRouter()

	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var userInfo authentication.UserInfo
			var err error
			accessToken := strings.TrimPrefix(r.Header.Get(authorizationHeaderKey), "Bearer ")

			// try to authenticate the access token
			if len(accessToken) == 0 {
				err = errors.New("no access token provided")
			} else {
				userInfo, err = agt.tokens.Authenticate(accessToken)
			}

			if err != nil {
				// either not token was provided or user is unauthenticated with k8s API
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			// direct proxy
			if kube.MatchingUsers(agt.userInfo, userInfo) {
				proxyHandler.ServeHTTP(w, r)
				return
			}

			apiCtx := &apiContext{
				tag:                  fmt.Sprintf("%016x", time.Now().Unix()),
				response:             w,
				request:              r,
				proxyHandler:         proxyHandler,
				filterReaderLabelSet: agt.cfg.filterReaderLabelSet,
				namespaceSet:         agt.namespaces.Query(accessToken),
				remoteAPI:            agt.remoteAPI,
			}

			newReqCtx := context.WithValue(r.Context(), apiContextKey, apiCtx)
			next.ServeHTTP(w, r.WithContext(newReqCtx))
		})
	})

	router.Path("/api/v1/query").Methods("GET", "POST").Handler(apiContextHandler(hijackQuery))
	router.Path("/api/v1/query_range").Methods("GET", "POST").Handler(apiContextHandler(hijackQueryRange))
	router.Path("/api/v1/series").Methods("GET").Handler(apiContextHandler(hijackSeries))
	router.Path("/api/v1/read").Methods("POST").Handler(apiContextHandler(hijackRead))
	router.Path("/api/v1/label/__name__/values").Methods("GET").Handler(apiContextHandler(hijackLabelName))
	router.Path("/api/v1/label/namespace/values").Methods("GET").Handler(apiContextHandler(hijackLabelNamespaces))
	router.Path("/api/v1/label/{name}/values").Methods("GET").Handler(proxyHandler)
	router.Path("/federate").Methods("GET").Handler(apiContextHandler(hijackFederate))

	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})

	return router
}

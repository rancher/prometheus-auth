package remote

import (
	"context"
	"fmt"
	"github.com/juju/errors"
	"github.com/prometheus/common/config"
	promconfig "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/prompb"
	promremote "github.com/prometheus/prometheus/storage/remote"
	"github.com/rancher/prometheus-auth/pkg/kubeauth"
	"github.com/rancher/prometheus-auth/pkg/prommetric"
	"github.com/rancher/prometheus-auth/pkg/utils"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	rancherProjectIDLabelKey = "field.cattle.io/projectId"
	namespaceMatchName       = "namespace"
)

func Start() cli.Command {
	cmd := cli.Command{}
	cmd.Name = "start"
	cmd.Usage = "Start a Prometheus remote reader"
	cmd.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "listen.address",
			Usage: "[optional] Address to listen",
			Value: ":9201",
		},
		cli.StringFlag{
			Name:  "remote.read-url",
			Usage: "[optional] URL to read from remote",
			Value: "http://prometheus-operated.cattle-prometheus:9090/api/v1/read",
		},
		cli.DurationFlag{
			Name:  "remote.read-timeout",
			Usage: "[optional] Timeout to read from remote",
			Value: 5 * time.Second,
		},
		cli.StringSliceFlag{
			Name:  "filter.external-labels-key",
			Usage: "[optional] Filter out the keys of configured 'externalLabels' before reading remote",
			Value: &cli.StringSlice{"prometheus", "prometheus_replica"},
		},
		cli.BoolFlag{
			Name:  "filter.only",
			Usage: "[optional] Only filter out the setting keys, but not authorization",
		},
		cli.DurationFlag{
			Name:  "watch.resync-period",
			Usage: "[optional] Resync period of Kubernetes watching",
			Value: 15 * time.Minute,
		},
	}

	cmd.Action = func(cliContext *cli.Context) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cfg := &readerConfig{
			ctx:                  ctx,
			listenAddress:        cliContext.String("listen.address"),
			remoteReadTimeout:    cliContext.Duration("remote.read-timeout"),
			withoutAuthorization: cliContext.Bool("filter.only"),
			watchResyncPeriod:    cliContext.Duration("watch.resync-period"),
		}

		externalLabelsKey := cliContext.StringSlice("filter.external-labels-key")
		cfg.ignoreLabelsKeyMap = make(map[string]struct{}, len(externalLabelsKey))
		for _, key := range externalLabelsKey {
			cfg.ignoreLabelsKeyMap[key] = struct{}{}
		}

		remoteReadURLString := cliContext.String("remote.read-url")
		if len(remoteReadURLString) == 0 {
			log.Fatal("--remote.read-url is blank")
		}
		remoteReadURL, err := url.Parse(remoteReadURLString)
		if err != nil {
			log.Fatal("Unable to parse remote.read-url")
		}
		cfg.remoteURL = remoteReadURL

		reader, err := createReader(cfg)
		if err != nil {
			log.WithError(err).Fatal("Failed to create remote reader")
		}

		if err = reader.server(); err != nil {
			log.WithError(err).Fatal("Failed to serve")
		}
	}

	return cmd
}

type readerConfig struct {
	ctx                  context.Context
	listenAddress        string
	ignoreLabelsKeyMap   map[string]struct{}
	withoutAuthorization bool
	remoteURL            *url.URL
	remoteReadTimeout    time.Duration
	watchResyncPeriod    time.Duration
}

func createReader(cfg *readerConfig) (*reader, error) {
	// create Prometheus client
	var promClientHTTPConfig promconfig.HTTPClientConfig
	accessTokenPath := "/var/run/secrets/kubernetes.io/serviceaccount/token"
	accessTokenBytes, err := ioutil.ReadFile(accessTokenPath)
	if err == nil {
		accessToken := strings.TrimSpace(string(accessTokenBytes))
		if len(accessToken) != 0 {
			promClientHTTPConfig = promconfig.HTTPClientConfig{
				BearerTokenFile: accessTokenPath,
			}
		}
	}
	promClient, err := promremote.NewClient(0, &promremote.ClientConfig{
		URL:              &config.URL{URL: cfg.remoteURL},
		Timeout:          model.Duration(cfg.remoteReadTimeout),
		HTTPClientConfig: promClientHTTPConfig,
	})
	if err != nil {
		return nil, errors.Annotatef(err, "unable to create prometheus remote client")
	}

	// create Kubernetes client
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Annotatef(err, "unable to create Kubernetes config by InClusterConfig()")
	}
	k8sClient, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, errors.Annotatef(err, "unable to new Kubernetes clientSet")
	}

	// query ProjectID
	ownedNamespaceBytes, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return nil, errors.Annotate(err, "unable to query Namespace file mounted by Kubernetes")
	}
	ownedNamespace, err := k8sClient.CoreV1().Namespaces().Get(string(ownedNamespaceBytes), metav1.GetOptions{})
	if err != nil {
		return nil, errors.Annotatef(err, "unable to query Namespace %s", string(ownedNamespaceBytes))
	}
	projectID := ownedNamespace.Labels[rancherProjectIDLabelKey]
	if len(projectID) == 0 {
		return nil, errors.Annotatef(err, "unable to query ProjectID from Namespace %s", string(ownedNamespaceBytes))
	}

	var (
		projectNamespaceOwnedView kubeauth.ProjectNamespacesOwnedView
		labelTranslator           prommetric.PromPbLabelMatcherNameTranslator
	)

	if !cfg.withoutAuthorization {
		projectNamespaceOwnedView = kubeauth.NewProjectNamespacesOwnedView(informers.NewSharedInformerFactory(k8sClient, cfg.watchResyncPeriod), func(obj interface{}) (match bool) {
			if objMeta, err := meta.Accessor(obj); err == nil {
				labels := objMeta.GetLabels()
				if pID, exist := labels[rancherProjectIDLabelKey]; exist {
					match = pID == projectID
				}
			}

			return
		})

		labelTranslator = map[string]prommetric.PromPbLabelMatcherTranslator{
			namespaceMatchName: prommetric.CreatePromPbLabelMatcherTranslator(namespaceMatchName, projectNamespaceOwnedView),
		}
	}

	return &reader{
		cfg:                        cfg,
		matchersFilter:             prommetric.CreatePromPbLabelMatchersNameFilter(cfg.ignoreLabelsKeyMap, labelTranslator),
		projectNamespacesOwnedView: projectNamespaceOwnedView,
		promClient:                 promClient,
	}, nil
}

type reader struct {
	cfg                        *readerConfig
	matchersFilter             prommetric.PromPbLabelMatchersFilter
	promClient                 *promremote.Client
	projectNamespacesOwnedView kubeauth.ProjectNamespacesOwnedView
}

func (r *reader) hijackRemoteRead(ctx context.Context, queries []*prompb.Query) ([]*prompb.Query, error) {
	for _, query := range queries {
		utils.LogTrace(func() string {
			return fmt.Sprintf("raw => %s", utils.JSON(query))
		})

		query.Matchers = r.matchersFilter.Filter(query.GetMatchers())

		utils.LogTrace(func() string {
			return fmt.Sprintf("hjk => %s", utils.JSON(query))
		})
	}

	return queries, nil
}

func (r *reader) read(ctx context.Context, req *prompb.ReadRequest) (*prompb.ReadResponse, error) {
	authQueries, err := r.hijackRemoteRead(ctx, req.Queries)
	if err != nil {
		return nil, errors.Annotate(err, "failed to hijack remote read")
	}

	rawResults := make([]*prompb.QueryResult, 0, len(authQueries))
	for _, query := range authQueries {
		result, err := r.promClient.Read(ctx, query)
		if err != nil {
			return nil, errors.Annotate(err, "unable call remote")
		}

		rawResults = append(rawResults, result)
	}

	return &prompb.ReadResponse{
		Results: rawResults,
	}, nil
}

func (r *reader) startSync() {
	if r.cfg.withoutAuthorization {
		return
	}

	go r.projectNamespacesOwnedView.Run(r.cfg.ctx.Done())
}

func (r *reader) server() error {
	r.startSync()

	http.HandleFunc("/read", func(w http.ResponseWriter, rdr *http.Request) {
		req, err := promremote.DecodeReadRequest(rdr)
		if err != nil {
			log.WithError(err).Error("Failed to decode read request")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp, err := r.read(rdr.Context(), req)
		if err != nil {
			log.WithError(err).Warn("Failed to execute query from remote storage")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := promremote.EncodeReadResponse(resp, w); err != nil {
			log.WithError(err).Warn("Failed to encode read response")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	log.Infof("Start listening for connections on %s", r.cfg.listenAddress)
	if err := http.ListenAndServe(r.cfg.listenAddress, nil); err != nil {
		return errors.Annotatef(err, "unable to listen addr %s", r.cfg.listenAddress)
	}

	return nil
}

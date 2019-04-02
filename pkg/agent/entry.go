package agent

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"strings"
	"time"

	"github.com/cockroachdb/cmux"
	"github.com/juju/errors"
	grpcproxy "github.com/mwitkow/grpc-proxy/proxy"
	promapi "github.com/prometheus/client_golang/api"
	promapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/rancher/prometheus-auth/pkg/data"
	"github.com/rancher/prometheus-auth/pkg/kube"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"golang.org/x/net/netutil"
	"google.golang.org/grpc"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func Run(cliContext *cli.Context) {
	// enable profiler
	go func() {
		log.Println(http.ListenAndServe("127.0.0.1:6060", nil))
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &agentConfig{
		ctx:                  ctx,
		listenAddress:        cliContext.String("listen-address"),
		readTimeout:          cliContext.Duration("read-timeout"),
		maxConnections:       cliContext.Int("max-connections"),
		filterReaderLabelSet: data.NewSet(cliContext.StringSlice("filter-reader-labels")...),
	}

	proxyURLString := cliContext.String("proxy-url")
	if len(proxyURLString) == 0 {
		log.Fatal("--agent.proxy-url is blank")
	}
	proxyURL, err := url.Parse(proxyURLString)
	if err != nil {
		log.Fatal("Unable to parse agent.proxy-url")
	}
	cfg.proxyURL = proxyURL

	accessTokenPath := "/var/run/secrets/kubernetes.io/serviceaccount/token"
	accessTokenBytes, err := ioutil.ReadFile(accessTokenPath)
	if err != nil {
		log.WithError(err).Fatalf("Failed to read token file %q", accessTokenPath)
	}
	accessToken := strings.TrimSpace(string(accessTokenBytes))
	if len(accessToken) == 0 {
		log.Fatalf("Read empty token from file %q", accessTokenPath)
	}
	cfg.myToken = accessToken

	log.Println(cfg)

	reader, err := createAgent(cfg)
	if err != nil {
		log.WithError(err).Fatal("Failed to create agent")
	}

	if err = reader.serve(); err != nil {
		log.WithError(err).Fatal("Failed to serve")
	}
}

type agentConfig struct {
	ctx                  context.Context
	myToken              string
	listenAddress        string
	proxyURL             *url.URL
	readTimeout          time.Duration
	maxConnections       int
	filterReaderLabelSet data.Set
}

func (a *agentConfig) String() string {
	sb := &strings.Builder{}

	sb.WriteString(fmt.Sprint("listening on ", a.listenAddress))
	sb.WriteString(fmt.Sprint(", proxying to ", a.proxyURL.String()))
	sb.WriteString(fmt.Sprintf(" with ignoring 'remote reader' labels [%s]", a.filterReaderLabelSet))
	sb.WriteString(fmt.Sprintf(", only allow maximum %d connections with %v read timeout", a.maxConnections, a.readTimeout))
	sb.WriteString(" .")

	return sb.String()
}

type agent struct {
	cfg        *agentConfig
	listener   net.Listener
	namespaces kube.Namespaces
	remoteAPI  promapiv1.API
}

func (a *agent) serve() error {
	listenerMux := cmux.New(a.listener)
	httpProxy := a.createHTTPProxy()
	grpcProxy := a.createGRPCProxy()

	errCh := make(chan error)
	go func() {
		if err := httpProxy.Serve(createHTTPListener(listenerMux)); err != nil {
			errCh <- errors.Annotate(err, "failed to start proxy http listener")
		}
	}()
	go func() {
		if err := grpcProxy.Serve(createGRPCListener(listenerMux, a.cfg.myToken)); err != nil {
			errCh <- errors.Annotate(err, "failed to start proxy grpc listener")
		}
	}()
	go func() {
		log.Infof("Start listening for connections on %s", a.cfg.listenAddress)

		if err := listenerMux.Serve(); err != nil {
			errCh <- errors.Annotatef(err, "failed to listen on %s", a.cfg.listenAddress)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-a.cfg.ctx.Done():
		grpcProxy.GracefulStop()
		httpProxy.Shutdown(a.cfg.ctx)
		return nil
	}
}

func createAgent(cfg *agentConfig) (*agent, error) {
	utilruntime.ReallyCrash = false
	utilruntime.PanicHandlers = []func(interface{}){
		func(i interface{}) {
			if err, ok := i.(error); ok {
				log.Error(errors.ErrorStack(err))
			} else {
				log.Error(i)
			}
		},
	}
	utilruntime.ErrorHandlers = []func(err error){
		func(err error) {
			log.Error(errors.ErrorStack(err))
		},
	}

	listener, err := net.Listen("tcp", cfg.listenAddress)
	if err != nil {
		return nil, errors.Annotatef(err, "unable to listen on addr %s", cfg.listenAddress)
	}
	listener = netutil.LimitListener(listener, cfg.maxConnections)

	// create Kubernetes client
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Annotate(err, "unable to create Kubernetes config by InClusterConfig()")
	}
	k8sClient, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, errors.Annotate(err, "unable to new Kubernetes clientSet")
	}

	// create Prometheus client
	promClient, err := promapi.NewClient(promapi.Config{
		Address: cfg.proxyURL.String(),
	})
	if err != nil {
		return nil, errors.Annotate(err, "unable to new Prometheus client")
	}

	return &agent{
		cfg:        cfg,
		listener:   listener,
		namespaces: kube.NewNamespaces(cfg.ctx, k8sClient),
		remoteAPI:  promapiv1.NewAPI(promClient),
	}, nil
}

func (a *agent) createHTTPProxy() *http.Server {
	return &http.Server{
		Handler:     a.httpBackend(),
		ReadTimeout: a.cfg.readTimeout,
	}
}

func (a *agent) createGRPCProxy() *grpc.Server {
	return grpc.NewServer(
		grpc.CustomCodec(grpcproxy.Codec()),
		grpc.UnknownServiceHandler(a.grpcBackend()),
	)
}

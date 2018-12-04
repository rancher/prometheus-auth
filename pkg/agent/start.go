package agent

import (
	"context"
	"github.com/cockroachdb/cmux"
	"github.com/juju/errors"
	grpcproxy "github.com/mwitkow/grpc-proxy/proxy"
	promapi "github.com/prometheus/client_golang/api"
	promapiv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/rancher/prometheus-auth/pkg/kubeauth"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"golang.org/x/net/netutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io/ioutil"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

const (
	v1Api = "/api/v1"
	v2Api = "/api/v2"
)

func Start() cli.Command {
	cmd := cli.Command{}
	cmd.Name = "start"
	cmd.Usage = "Start a Prometheus agent"
	cmd.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "listen.address",
			Usage: "[optional] Address to listening",
			Value: ":9090",
		},
		cli.StringFlag{
			Name:  "agent.proxy-url",
			Usage: "[optional] URL to proxy",
			Value: "http://localhost:9999",
		},
		cli.DurationFlag{
			Name:  "agent.read-timeout",
			Usage: "[optional] Maximum duration before timing out read of the request, and closing idle connections",
			Value: 5 * time.Minute,
		},
		cli.IntFlag{
			Name:  "agent.max-connections",
			Usage: "[optional] Maximum number of simultaneous connections",
			Value: 512,
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

		cfg := &agentConfig{
			ctx:                 ctx,
			listenAddress:       cliContext.String("listen.address"),
			agentReadTimeout:    cliContext.Duration("agent.read-timeout"),
			agentMaxConnections: cliContext.Int("agent.max-connections"),
			watchResyncPeriod:   cliContext.Duration("watch.resync-period"),
		}

		agentProxyURLString := cliContext.String("agent.proxy-url")
		if len(agentProxyURLString) == 0 {
			log.Fatal("--agent.proxy-url is blank")
		}
		agentProxyURL, err := url.Parse(agentProxyURLString)
		if err != nil {
			log.Fatal("Unable to parse agent.proxy-url")
		}
		cfg.proxyURL = agentProxyURL

		accessTokenPath := "/var/run/secrets/kubernetes.io/serviceaccount/token"
		accessTokenBytes, err := ioutil.ReadFile(accessTokenPath)
		if err != nil {
			log.WithError(err).Fatalf("Failed to read token file %q", accessTokenPath)
		}
		accessToken := strings.TrimSpace(string(accessTokenBytes))
		if len(accessToken) == 0 {
			log.Fatalf("Read empty token from file %q", accessTokenPath)
		}
		cfg.hostAccessToken = accessToken

		reader, err := createAgent(cfg)
		if err != nil {
			log.WithError(err).Fatal("Failed to create agent")
		}

		if err = reader.server(); err != nil {
			log.WithError(err).Fatal("Failed to serve")
		}
	}

	return cmd
}

type agentConfig struct {
	ctx                 context.Context
	listenAddress       string
	hostAccessToken     string
	proxyURL            *url.URL
	agentReadTimeout    time.Duration
	agentMaxConnections int
	watchResyncPeriod   time.Duration
}

type agent struct {
	cfg                       *agentConfig
	listener                  net.Listener
	globalNamespacesOwnedView kubeauth.GlobalNamespacesOwnedView
	backendHTTPApi            promapiv1.API
}

func createAgent(cfg *agentConfig) (*agent, error) {
	listener, err := net.Listen("tcp", cfg.listenAddress)
	if err != nil {
		return nil, errors.Annotatef(err, "unable to listen addr %s", cfg.listenAddress)
	}
	listener = netutil.LimitListener(listener, cfg.agentMaxConnections)

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
		cfg:                       cfg,
		listener:                  listener,
		globalNamespacesOwnedView: kubeauth.NewGlobalNamespacesOwnedView(informers.NewSharedInformerFactory(k8sClient, cfg.watchResyncPeriod), k8sClient),
		backendHTTPApi:            promapiv1.NewAPI(promClient),
	}, nil
}

func (a *agent) startSync() {
	go a.globalNamespacesOwnedView.Run(a.cfg.ctx.Done())
}

func (a *agent) server() error {
	a.startSync()

	mux := cmux.New(a.listener)
	httpProxy := a.createHTTPProxy()
	grpcProxy := a.createGRPCProxy()

	errCh := make(chan error)
	go func() {
		httpl, err := createHTTPListener(mux)
		if err != nil {
			errCh <- errors.Annotate(err, "failed to create http proxy")
		}

		err = httpProxy.Serve(httpl)
		if err != nil {
			errCh <- errors.Annotate(err, "failed to access http proxy")
		}
	}()
	go func() {
		grpcl, err := createGRPCListener(mux, a.cfg.hostAccessToken)
		if err != nil {
			errCh <- errors.Annotate(err, "failed to create grpc proxy")
		}

		err = grpcProxy.Serve(grpcl)
		if err != nil {
			errCh <- errors.Annotate(err, "failed to access grpc proxy")
		}
	}()
	go func() {
		log.Infof("Start listening for connections on %s", a.cfg.listenAddress)
		if err := mux.Serve(); err != nil {
			errCh <- errors.Annotatef(err, "failed to listen on %s", a.cfg.listenAddress)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-a.cfg.ctx.Done():
		httpProxy.Shutdown(a.cfg.ctx)
		grpcProxy.GracefulStop()
		return nil
	}
}

func (a *agent) createHTTPProxy() *http.Server {
	backend := httputil.NewSingleHostReverseProxy(a.cfg.proxyURL)

	mux := http.NewServeMux()
	mux.Handle("/", a.wrapBackend(backend))

	return &http.Server{
		Handler:     mux,
		ReadTimeout: a.cfg.agentReadTimeout,
	}
}

func (a *agent) createGRPCProxy() *grpc.Server {
	backend := grpcproxy.TransparentHandler(func(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, error) {
		con, err := grpc.DialContext(ctx, a.cfg.proxyURL.String(), grpc.WithDefaultCallOptions(grpc.CallCustomCodec(grpcproxy.Codec())))
		if err != nil {
			return ctx, nil, status.Errorf(codes.Unavailable, "Unavailable endpoint")
		}

		return ctx, con, nil
	})

	return grpc.NewServer(
		grpc.CustomCodec(grpcproxy.Codec()),
		grpc.UnknownServiceHandler(backend),
	)
}

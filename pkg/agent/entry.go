package agent

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
	"time"

	"github.com/rancher/prometheus-auth/pkg/data"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func Run(cliContext *cli.Context) {
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

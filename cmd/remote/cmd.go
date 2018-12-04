package remote

import (
	"github.com/rancher/prometheus-auth/pkg/remote"
	"github.com/urfave/cli"
)

func Command() cli.Command {
	cmd := cli.Command{}
	cmd.Name = "remote"
	cmd.Usage = "An authorization remote reader for Prometheus, with RBAC: [namespaces](list, watch, get)"

	cmd.Subcommands = cli.Commands{
		remote.Start(),
	}

	return cmd
}

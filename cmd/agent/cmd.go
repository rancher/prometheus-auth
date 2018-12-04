package agent

import (
	"github.com/rancher/prometheus-auth/pkg/agent"
	"github.com/urfave/cli"
)

func Command() cli.Command {
	cmd := cli.Command{}
	cmd.Name = "agent"
	cmd.Usage = "An authorization agent before Prometheus, with RBAC: [namespaces, secrets, serviceaccounts, clusterrole, clusterrolebindings, role, rolebindings](list, watch, get)"

	cmd.Subcommands = cli.Commands{
		agent.Start(),
	}

	return cmd
}

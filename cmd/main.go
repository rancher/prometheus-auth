package main

import (
	"fmt"
	"github.com/rancher/prometheus-auth/cmd/agent"
	"github.com/rancher/prometheus-auth/cmd/remote"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
)

var (
	VER  = "dev"
	HASH = "-"
)

func main() {
	app := cli.NewApp()
	app.Version = fmt.Sprintf("%s(%s)", VER, HASH)
	app.Name = "prometheus-auth"
	app.Usage = "Authorization plugins for Rancher monitoring"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "log.json",
			Usage: "[optional] Log as JSON",
		},
		cli.BoolFlag{
			Name:  "log.debug",
			Usage: "[optional] Log debug info",
		},
	}

	app.Commands = []cli.Command{
		remote.Command(),
		agent.Command(),
	}

	app.Before = func(context *cli.Context) error {
		if context.Bool("log.json") {
			log.SetFormatter(&log.JSONFormatter{})
		}

		if context.Bool("log.debug") {
			log.SetLevel(log.DebugLevel)
		}

		log.SetOutput(os.Stdout)

		return nil
	}

	app.Run(os.Args)
}

package main

import (
	"os"

	"github.com/dgate-io/dgate/cmd/dgate-cli/commands"
	"github.com/dgate-io/dgate/internal/extensions/telemetry"
	"github.com/dgate-io/dgate/pkg/dgclient"
)

var version string = "dev"

func main() {
	telemetry.SetupTelemetry("dgate-cli", version)
	client := dgclient.NewDGateClient()
	err := commands.Run(client, version)
	if err != nil {
		os.Stderr.WriteString(err.Error())
		os.Stderr.WriteString("\n")
		os.Exit(1)
	}
}

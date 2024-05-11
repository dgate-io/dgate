package main

import (
	"github.com/dgate-io/dgate/cmd/dgate-cli/commands"
	"github.com/dgate-io/dgate/pkg/dgclient"
)

var version string = "dev"

func main() {
	client := dgclient.NewDGateClient()
	commands.Run(client, version)
}

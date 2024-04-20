package main

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/dgate-io/dgate/cmd/dgate-cli/commands"
	"github.com/dgate-io/dgate/pkg/dgclient"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

var version string = "dev"

var client = &dgclient.DGateClient{}

func main() {
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		bv := buildInfo.Main.Version
		if bv != "" && bv != "(devel)" {
			version = buildInfo.Main.Version
		}
	}

	app := &cli.App{
		Name:    "dgate-cli",
		Usage:   "a command line interface for DGate (API Gateway) Admin API",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "admin",
				Value:   "http://localhost:9080",
				EnvVars: []string{"DGATE_ADMIN_API"},
				Usage:   "the url for the file client",
			},
			// only basic auth support for now
			&cli.StringFlag{
				Name:    "auth",
				Aliases: []string{"a"},
				EnvVars: []string{"DGATE_ADMIN_AUTH"},
				Usage:   "the authorization credentials or token",
			},
			// &cli.StringFlag{
			// 	Name:    "auth-scheme",
			// 	Value:   "Basic",
			// 	Aliases: []string{"A"},
			// 	EnvVars: []string{"DGATE_ADMIN_AUTH_SCHEME"},
			// 	Usage:   "the authorization scheme",
			// },
			&cli.StringFlag{
				Name:    "tag",
				Aliases: []string{"t"},
				Value:   "default",
				Usage:   "the namespace for the file client",
			},
		},
		Before: func(ctx *cli.Context) (err error) {
			var authOption dgclient.Options = func(dc *dgclient.DGateClient) {}
			if auth := ctx.String("auth"); strings.Contains(auth, ":") {
				pair := strings.SplitN(ctx.String("auth"), ":", 2)
				username := pair[0]
				password := ""
				if len(pair) > 1 {
					password = pair[1]
				}
				authOption = dgclient.WithBasicAuth(
					username, password,
				)
			} else if auth != "" {
				fmt.Printf("password for %s:", auth)
				password, err := term.ReadPassword(0)
				if err != nil {
					return err
				}
				fmt.Print("\n")
				authOption = dgclient.WithBasicAuth(
					auth, string(password),
				)
			}
			return client.Init(
				ctx.String("admin"),
				authOption,
				dgclient.WithUserAgent(
					"DGate CLI "+version+
						";os="+runtime.GOOS+
						";arch="+runtime.GOARCH,
				),
			)
		},
		Action: func(ctx *cli.Context) error {
			return ctx.App.Command("help").Run(ctx)
		},
		Commands: []*cli.Command{
			commands.NamespaceCommand(client),
			commands.ServiceCommand(client),
			commands.ModuleCommand(client),
			commands.RouteCommand(client),
			commands.DomainCommand(client),
			commands.CollectionCommand(client),
			commands.DocumentCommand(client),
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
	}
}

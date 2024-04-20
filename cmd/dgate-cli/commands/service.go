package commands

import (
	"github.com/dgate-io/dgate/pkg/dgclient"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/urfave/cli/v2"
)

func ServiceCommand(client *dgclient.DGateClient) *cli.Command {
	return &cli.Command{
		Name:      "service",
		Aliases:   []string{"svc"},
		Args:      true,
		ArgsUsage: "<command> <name>",
		Usage:     "service commands",
		Subcommands: []*cli.Command{
			{
				Name:    "create",
				Aliases: []string{"mk"},
				Usage:   "create a service",
				Action: func(ctx *cli.Context) error {
					svc, err := createMapFromArgs[spec.Service](
						ctx.Args().Slice(), "name", "urls",
					)
					if err != nil {
						return err
					}
					err = client.CreateService(svc)
					if err != nil {
						return err
					}
					return jsonPrettyPrint(svc)
				},
			},
			{
				Name:    "delete",
				Aliases: []string{"rm"},
				Usage:   "delete a service",
				Action: func(ctx *cli.Context) error {
					svc, err := createMapFromArgs[spec.Service](
						ctx.Args().Slice(), "name",
					)
					if err != nil {
						return err
					}
					err = client.DeleteService(
						svc.Name, svc.NamespaceName,
					)
					if err != nil {
						return err
					}
					return nil
				},
			},
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "list services",
				Action: func(ctx *cli.Context) error {
					svc, err := client.ListService()
					if err != nil {
						return err
					}
					return jsonPrettyPrint(svc)
				},
			},
			{
				Name:  "get",
				Usage: "get a service",
				Action: func(ctx *cli.Context) error {
					svc, err := createMapFromArgs[spec.Service](
						ctx.Args().Slice(), "name",
					)
					if err != nil {
						return err
					}
					ns, err := client.GetService(
						svc.Name, svc.NamespaceName,
					)
					if err != nil {
						return err
					}
					return jsonPrettyPrint(ns)
				},
			},
		},
	}
}

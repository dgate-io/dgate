package commands

import (
	"github.com/dgate-io/dgate/pkg/dgclient"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/urfave/cli/v2"
)

func RouteCommand(client *dgclient.DGateClient) *cli.Command {
	return &cli.Command{
		Name:      "route",
		Aliases:   []string{"rt"},
		Args:      true,
		ArgsUsage: "<command> <name>",
		Usage:     "route commands",
		Subcommands: []*cli.Command{
			{
				Name:    "create",
				Aliases: []string{"mk"},
				Usage:   "create a route",
				Action: func(ctx *cli.Context) error {
					rt, err := createMapFromArgs[spec.Route](
						ctx.Args().Slice(), "name",
						"paths", "methods",
					)
					if err != nil {
						return err
					}
					err = client.CreateRoute(rt)
					if err != nil {
						return err
					}
					return jsonPrettyPrint(rt)
				},
			},
			{
				Name:    "delete",
				Aliases: []string{"rm"},
				Usage:   "delete a route",
				Action: func(ctx *cli.Context) error {
					rt, err := createMapFromArgs[spec.Route](
						ctx.Args().Slice(), "name",
					)
					if err != nil {
						return err
					}
					err = client.DeleteRoute(
						rt.Name, rt.NamespaceName,
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
				Usage:   "list routes",
				Action: func(ctx *cli.Context) error {
					rt, err := client.ListRoute()
					if err != nil {
						return err
					}
					return jsonPrettyPrint(rt)
				},
			},
			{
				Name:  "get",
				Usage: "get a route",
				Action: func(ctx *cli.Context) error {
					rt, err := createMapFromArgs[spec.Route](
						ctx.Args().Slice(), "name",
					)
					if err != nil {
						return err
					}
					rt, err = client.GetRoute(
						rt.Name, rt.NamespaceName,
					)
					if err != nil {
						return err
					}
					return jsonPrettyPrint(rt)
				},
			},
		},
	}
}

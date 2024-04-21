package commands

import (
	"github.com/dgate-io/dgate/pkg/dgclient"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/urfave/cli/v2"
)

func DocumentCommand(client *dgclient.DGateClient) *cli.Command {
	return &cli.Command{
		Name:      "document",
		Aliases:   []string{"doc"},
		Args:      true,
		ArgsUsage: "<command> <name>",
		Usage:     "document commands",
		Subcommands: []*cli.Command{
			{
				Name:    "create",
				Aliases: []string{"mk"},
				Usage:   "create a document",
				Action: func(ctx *cli.Context) error {
					doc, err := createMapFromArgs[spec.Document](
						ctx.Args().Slice(), "id",
						"collection", "data",
					)
					if err != nil {
						return err
					}
					err = client.CreateDocument(doc)
					if err != nil {
						return err
					}
					return jsonPrettyPrint(doc)
				},
			},
			{
				Name:    "delete",
				Aliases: []string{"rm"},
				Usage:   "delete a document",
				Action: func(ctx *cli.Context) error {
					doc, err := createMapFromArgs[spec.Document](
						ctx.Args().Slice(), "id",
					)
					if err != nil {
						return err
					}
					err = client.DeleteDocument(
						doc.ID, doc.NamespaceName,
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
				Usage:   "list documents",
				Action: func(ctx *cli.Context) error {
					doc, err := client.ListDocument()
					if err != nil {
						return err
					}
					return jsonPrettyPrint(doc)
				},
			},
			{
				Name:  "get",
				Usage: "get a document",
				Action: func(ctx *cli.Context) error {
					doc, err := createMapFromArgs[spec.Document](
						ctx.Args().Slice(), "id",
					)
					if err != nil {
						return err
					}
					doc, err = client.GetDocument(
						doc.ID, doc.NamespaceName,
					)
					if err != nil {
						return err
					}
					return jsonPrettyPrint(doc)
				},
			},
		},
	}
}

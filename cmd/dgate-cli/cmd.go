package main

import (
	"github.com/spf13/cobra"
)

var (
	dgateClientCmd = &cobra.Command{
		Use:           "dgate-cli",
		Short:         "dgate-cli - a command line interface for dgate (API Gateway)",
		Long:          ``,
		Version:       "0.1.0",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	targetServer string
	basicAuth    string
	tags         []string
)

func Execute() error {
	return dgateClientCmd.Execute()
}

func init() {
	dgateClientCmd.PersistentFlags().StringVarP(
		&targetServer, "server", "S",
		"http://localhost:9080",
		"server location (default: http://localhost:9080)")
	dgateClientCmd.PersistentFlags().StringVarP(
		&basicAuth, "basic-auth", "A",
		"", "basic auth credentials")
	tags = *dgateClientCmd.PersistentFlags().StringArrayP(
		"tag", "s", nil, "tag to apply to resource")
	dgateClientCmd.AddCommand(namespaceCmd)
	// dgateClientCmd.AddCommand(serviceCmd)
	// dgateClientCmd.AddCommand(routeCmd)
	// dgateClientCmd.AddCommand(moduleCmd)
}

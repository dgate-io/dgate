package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
)

var (
	namespaceCmd = &cobra.Command{
		Use:   "namespace",
		Short: "",
		Args:  cobra.NoArgs,
	}
	listNamespaceCmd = &cobra.Command{
		Use:   "list",
		Short: "",
		RunE:  listNamespaces,
		Args:  cobra.NoArgs,
	}
	getNamespaceCmd = &cobra.Command{
		Use:   "get",
		Short: "",
		RunE:  getNamespace,
		Args:  cobra.ExactArgs(1),
	}
	createNamespaceCmd = &cobra.Command{
		Use:   "create",
		Short: "",
		RunE:  createNamespace,
		Args:  cobra.ExactArgs(1),
	}
	updateNamespaceCmd = &cobra.Command{
		Use:   "create",
		Short: "",
		RunE:  updateNamespace,
		Args:  cobra.ExactArgs(1),
	}
	deleteNamespaceCmd = &cobra.Command{
		Use:   "delete",
		Short: "",
		RunE:  deleteNamespace,
		Args:  cobra.ExactArgs(1),
	}
	client = &http.Client{}
)

func listNamespaces(ccmd *cobra.Command, args []string) error {
	nsUrl, err := url.JoinPath(targetServer, "namespace")
	if err != nil {
		return err
	}
	req, err := http.NewRequest("GET", nsUrl, nil)
	if err != nil {
		return err
	}
	if basicAuth != "" {
		fmt.Println("Using basic authentication creds: ", basicAuth)
		creds := strings.SplitN(basicAuth, ":", 2)
		if len(creds) == 2 {
			req.SetBasicAuth(creds[0], creds[1])
		} else {
			req.SetBasicAuth(creds[0], "")
		}
	} else {
		fmt.Println("Not using basic authentication creds: ")
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	// read the response body
	namespaceJson, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println(string(namespaceJson))
	return nil
}

func getNamespace(ccmd *cobra.Command, args []string) error {
	nsUrl, err := url.JoinPath(targetServer, "namespace", args[0])
	if err != nil {
		return err
	}
	req, err := http.NewRequest("GET", nsUrl, nil)
	if err != nil {
		return err
	}
	if basicAuth != "" {
		fmt.Println("Using basic authentication creds: ", basicAuth)
		creds := strings.SplitN(basicAuth, ":", 2)
		if len(creds) == 2 {
			req.SetBasicAuth(creds[0], creds[1])
		} else {
			req.SetBasicAuth(creds[0], "")
		}
	} else {
		fmt.Println("Not using basic authentication creds: ")
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	fmt.Println(resp.Body)
	return nil
}

func createNamespace(ccmd *cobra.Command, args []string) error {
	return nil
}

func updateNamespace(ccmd *cobra.Command, args []string) error {
	ccmd.Help()
	return nil
}

func deleteNamespace(ccmd *cobra.Command, args []string) error {
	ccmd.Help()
	return nil
}

func init() {
	includeFlags(namespaceCmd)
}

func includeFlags(cmd *cobra.Command) {
	cmd.AddCommand(listNamespaceCmd)
	cmd.AddCommand(createNamespaceCmd)
	cmd.AddCommand(getNamespaceCmd)
	cmd.AddCommand(updateNamespaceCmd)
	cmd.AddCommand(deleteNamespaceCmd)
}

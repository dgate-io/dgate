package dgclient

import (
	"net/url"

	"github.com/dgate-io/dgate/pkg/spec"
)

func (d *DGateClient) GetNamespace(name string) (*spec.Namespace, error) {
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/namespace", name)
	if err != nil {
		return nil, err
	}
	return commonGet[spec.Namespace](d.client, uri)
}

func (d *DGateClient) CreateNamespace(ns *spec.Namespace) error {
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/namespace")
	if err != nil {
		return err
	}
	return commonPut(d.client, uri, ns)
}

func (d *DGateClient) DeleteNamespace(name string) error {
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/namespace")
	if err != nil {
		return err
	}
	return commonDelete(d.client, uri, name, "")
}

func (d *DGateClient) ListNamespace() ([]*spec.Namespace, error) {
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/namespace")
	if err != nil {
		return nil, err
	}
	return commonGetList[*spec.Namespace](d.client, uri)
}

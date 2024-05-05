package dgclient

import (
	"net/url"

	"github.com/dgate-io/dgate/pkg/spec"
)

func (d *DGateClient) GetModule(name, namespace string) (*spec.Module, error) {
	query := d.baseUrl.Query()
	query.Set("namespace", namespace)
	d.baseUrl.RawQuery = query.Encode()
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/module", name)
	if err != nil {
		return nil, err
	}
	return commonGet[spec.Module](d.client, uri)
}

func (d *DGateClient) CreateModule(mod *spec.Module) error {
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/module")
	if err != nil {
		return err
	}
	return commonPut(d.client, uri, mod)
}

func (d *DGateClient) DeleteModule(name, namespace string) error {
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/module")
	if err != nil {
		return err
	}
	return commonDelete(d.client, uri, name, namespace)
}

func (d *DGateClient) ListModule(namespace string) ([]*spec.Module, error) {
	query := d.baseUrl.Query()
	query.Set("namespace", namespace)
	d.baseUrl.RawQuery = query.Encode()
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/module")
	if err != nil {
		return nil, err
	}
	return commonGetList[*spec.Module](d.client, uri)
}

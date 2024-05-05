package dgclient

import (
	"net/url"

	"github.com/dgate-io/dgate/pkg/spec"
)

func (d *DGateClient) GetService(name, namespace string) (*spec.Service, error) {
	query := d.baseUrl.Query()
	query.Set("namespace", namespace)
	d.baseUrl.RawQuery = query.Encode()
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/service", name)
	if err != nil {
		return nil, err
	}
	return commonGet[spec.Service](d.client, uri)
}

func (d *DGateClient) CreateService(svc *spec.Service) error {
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/service")
	if err != nil {
		return err
	}
	return commonPut(d.client, uri, svc)
}

func (d *DGateClient) DeleteService(name, namespace string) error {
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/service")
	if err != nil {
		return err
	}
	return commonDelete(d.client, uri, name, namespace)
}

func (d *DGateClient) ListService(namespace string) ([]*spec.Service, error) {
	query := d.baseUrl.Query()
	query.Set("namespace", namespace)
	d.baseUrl.RawQuery = query.Encode()
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/service")
	if err != nil {
		return nil, err
	}
	return commonGetList[*spec.Service](d.client, uri)
}

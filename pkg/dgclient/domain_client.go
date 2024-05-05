package dgclient

import (
	"net/url"

	"github.com/dgate-io/dgate/pkg/spec"
)

func (d *DGateClient) GetDomain(name, namespace string) (*spec.Domain, error) {
	query := d.baseUrl.Query()
	query.Set("namespace", namespace)
	d.baseUrl.RawQuery = query.Encode()
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/domain", name)
	if err != nil {
		return nil, err
	}
	return commonGet[spec.Domain](d.client, uri)
}

func (d *DGateClient) CreateDomain(dm *spec.Domain) error {
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/domain")
	if err != nil {
		return err
	}
	return commonPut(d.client, uri, dm)
}

func (d *DGateClient) DeleteDomain(name, namespace string) error {
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/domain")
	if err != nil {
		return err
	}
	return commonDelete(d.client, uri, name, namespace)
}

func (d *DGateClient) ListDomain(namespace string) ([]*spec.Domain, error) {
	query := d.baseUrl.Query()
	query.Set("namespace", namespace)
	d.baseUrl.RawQuery = query.Encode()
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/domain")
	if err != nil {
		return nil, err
	}
	return commonGetList[*spec.Domain](d.client, uri)
}

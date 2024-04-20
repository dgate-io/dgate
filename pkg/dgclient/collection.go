package dgclient

import (
	"net/url"

	"github.com/dgate-io/dgate/pkg/spec"
)

func (d *DGateClient) GetCollection(name, namespace string) (*spec.Collection, error) {
	query := d.baseUrl.Query()
	query.Set("namespace", namespace)
	d.baseUrl.RawQuery = query.Encode()
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/collection", name)
	if err != nil {
		return nil, err
	}
	return commonGet[spec.Collection](d.client, uri)
}

func (d *DGateClient) CreateCollection(svc *spec.Collection) error {
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/collection")
	if err != nil {
		return err
	}
	return commonPut(d.client, uri, svc)
}

func (d *DGateClient) DeleteCollection(name, namespace string) error {
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/collection")
	if err != nil {
		return err
	}
	return commonDelete(d.client, uri, name, namespace)
}

func (d *DGateClient) ListCollection() ([]*spec.Collection, error) {
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/collection")
	if err != nil {
		return nil, err
	}
	return commonGetList[*spec.Collection](d.client, uri)
}

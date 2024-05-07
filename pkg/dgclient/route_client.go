package dgclient

import (
	"net/url"

	"github.com/dgate-io/dgate/pkg/spec"
)

func (d *DGateClient) GetRoute(name, namespace string) (*spec.Route, error) {
	query := d.baseUrl.Query()
	query.Set("namespace", namespace)
	d.baseUrl.RawQuery = query.Encode()
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/route", name)
	if err != nil {
		return nil, err
	}
	return commonGet[spec.Route](d.client, uri)
}

func (d *DGateClient) CreateRoute(rt *spec.Route) error {
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/route")
	if err != nil {
		return err
	}
	return commonPut(d.client, uri, rt)
}

func (d *DGateClient) DeleteRoute(name, namespace string) error {
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/route")
	if err != nil {
		return err
	}
	return commonDelete(d.client, uri, name, namespace)
}

func (d *DGateClient) ListRoute(namespace string) ([]*spec.Route, error) {
	query := d.baseUrl.Query()
	query.Set("namespace", namespace)
	d.baseUrl.RawQuery = query.Encode()
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/route")
	if err != nil {
		return nil, err
	}
	return commonGetList[*spec.Route](d.client, uri)
}

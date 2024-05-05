package dgclient

import (
	"net/url"

	"github.com/dgate-io/dgate/pkg/spec"
)

func (d *DGateClient) GetDocument(id, namespace, collection string) (*spec.Document, error) {
	query := d.baseUrl.Query()
	query.Set("namespace", namespace)
	query.Set("collection", collection)
	d.baseUrl.RawQuery = query.Encode()
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/document", id)
	if err != nil {
		return nil, err
	}
	return commonGet[spec.Document](d.client, uri)
}

func (d *DGateClient) CreateDocument(doc *spec.Document) error {
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/document")
	if err != nil {
		return err
	}
	return commonPut(d.client, uri, doc)
}

func (d *DGateClient) DeleteDocument(id, namespace, collection string) error {
	query := d.baseUrl.Query()
	query.Set("namespace", namespace)
	query.Set("collection", collection)
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/document", id)
	if err != nil {
		return err
	}
	return basicDelete(d.client, uri, nil)
}

func (d *DGateClient) DeleteAllDocument(namespace, collection string) error {
	query := d.baseUrl.Query()
	query.Set("namespace", namespace)
	query.Set("collection", collection)
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/document")
	if err != nil {
		return err
	}
	return basicDelete(d.client, uri, nil)
}

func (d *DGateClient) ListDocument(namespace, collection string) ([]*spec.Document, error) {
	query := d.baseUrl.Query()
	query.Set("namespace", namespace)
	query.Set("collection", collection)
	d.baseUrl.RawQuery = query.Encode()
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/document")
	if err != nil {
		return nil, err
	}
	return commonGetList[*spec.Document](d.client, uri)
}

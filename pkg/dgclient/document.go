package dgclient

import (
	"net/url"

	"github.com/dgate-io/dgate/pkg/spec"
)

func (d *DGateClient) GetDocument(id, namespace string) (*spec.Document, error) {
	query := d.baseUrl.Query()
	query.Set("namespace", namespace)
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

func (d *DGateClient) DeleteDocument(id, namespace string) error {
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/document")
	if err != nil {
		return err
	}
	return commonDelete(d.client, uri, id, namespace)
}

func (d *DGateClient) ListDocument() ([]*spec.Document, error) {
	uri, err := url.JoinPath(d.baseUrl.String(), "/api/v1/document")
	if err != nil {
		return nil, err
	}
	return commonGetList[*spec.Document](d.client, uri)
}

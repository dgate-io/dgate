package resources

import (
	"github.com/dgate-io/dgate/pkg/spec"
)

type DocumentManager interface {
	GetDocumentByID(collection, namespace, id string) (*spec.Document, error)
	GetDocuments(collection, namespace string, limit, offset int) ([]*spec.Document, error)
}

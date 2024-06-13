package changestate

import (
	"github.com/dgate-io/dgate/internal/proxy"
	"github.com/dgate-io/dgate/internal/store"
	"github.com/dgate-io/dgate/pkg/resources"
	"github.com/dgate-io/dgate/pkg/spec"
)

type ChangeState interface {
	// Change state
	ApplyChangeLog(cl *spec.ChangeLog) error
	// Store
	Store() *store.Store
	// WaitForChanges() error
	ChangeHash() uint32

	// Readiness
	Ready() bool

	// Resources
	ResourceManager() *resources.ResourceManager
	DocumentManager() resources.DocumentManager
}

var _ ChangeState = (*proxy.ProxyState)(nil)

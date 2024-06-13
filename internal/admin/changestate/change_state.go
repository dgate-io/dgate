package changestate

import (
	"github.com/dgate-io/dgate/internal/proxy"
	"github.com/dgate-io/dgate/pkg/resources"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/hashicorp/raft"
)

type ChangeState interface {
	// Change state
	ApplyChangeLog(cl *spec.ChangeLog) error
	ProcessChangeLog(*spec.ChangeLog, bool) error
	WaitForChanges() error
	ReloadState(bool, ...*spec.ChangeLog) error
	ChangeHash() uint32

	// Readiness
	Ready() bool

	// Replication
	SetupRaft(*raft.Raft, chan raft.Observation)
	Raft() *raft.Raft

	// Resources
	ResourceManager() *resources.ResourceManager
	DocumentManager() resources.DocumentManager
}

var _ ChangeState = (*proxy.ProxyState)(nil)

package testutil

import (
	"io"
	"log/slog"

	"github.com/dgate-io/dgate/internal/admin/changestate"
	"github.com/dgate-io/dgate/pkg/resources"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockChangeState struct {
	mock.Mock
}

// ApplyChangeLog implements changestate.ChangeState.
func (m *MockChangeState) ApplyChangeLog(cl *spec.ChangeLog) error {
	return m.Called(cl).Error(0)
}

// ChangeHash implements changestate.ChangeState.
func (m *MockChangeState) ChangeHash() uint32 {
	return m.Called().Get(0).(uint32)
}

// DocumentManager implements changestate.ChangeState.
func (m *MockChangeState) DocumentManager() resources.DocumentManager {
	return m.Called().Get(0).(resources.DocumentManager)
}

// ResourceManager implements changestate.ChangeState.
func (m *MockChangeState) ResourceManager() *resources.ResourceManager {
	return m.Called().Get(0).(*resources.ResourceManager)
}

// Logger implements changestate.ChangeState.
func (m *MockChangeState) Logger() *zap.Logger {
	return m.Called().Get(0).(*zap.Logger)
}

// ProcessChangeLog implements changestate.ChangeState.
func (m *MockChangeState) ProcessChangeLog(cl *spec.ChangeLog, a bool) error {
	return m.Called(cl, a).Error(0)
}

// Raft implements changestate.ChangeState.
func (m *MockChangeState) Raft() *raft.Raft {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*raft.Raft)
}

// Ready implements changestate.ChangeState.
func (m *MockChangeState) Ready() bool {
	return m.Called().Get(0).(bool)
}

// ReloadState implements changestate.ChangeState.
func (m *MockChangeState) ReloadState(a bool, cls ...*spec.ChangeLog) error {
	return m.Called(a, cls).Error(0)
}

// SetupRaft implements changestate.ChangeState.
func (m *MockChangeState) SetupRaft(*raft.Raft, chan raft.Observation) {
	m.Called().Error(0)
}

// Version implements changestate.ChangeState.
func (m *MockChangeState) Version() string {
	return m.Called().Get(0).(string)
}

// WaitForChanges implements changestate.ChangeState.
func (m *MockChangeState) WaitForChanges() error {
	return m.Called().Error(0)
}

var _ changestate.ChangeState = &MockChangeState{}

func NewMockChangeState() *MockChangeState {
	mcs := &MockChangeState{}
	mcs.On("Logger").Return(slog.New(slog.NewTextHandler(io.Discard, nil)))
	mcs.On("Raft").Return(nil)
	return mcs
}

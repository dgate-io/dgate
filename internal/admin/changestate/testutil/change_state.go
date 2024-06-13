package testutil

import (
	"io"
	"log/slog"

	"github.com/dgate-io/dgate/internal/admin/changestate"
	"github.com/dgate-io/dgate/internal/store"
	"github.com/dgate-io/dgate/pkg/resources"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/dgate-io/raft"
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

// WaitForChanges implements changestate.ChangeState.
func (m *MockChangeState) WaitForChanges() error {
	return m.Called().Error(0)
}

// Store implements changestate.ChangeState.
func (m *MockChangeState) Store() *store.Store {
	args := m.Called()
	if st := args.Get(0); st != nil {
		return st.(*store.Store)
	}
	return nil
}

var _ changestate.ChangeState = &MockChangeState{}

func NewMockChangeState() *MockChangeState {
	mcs := &MockChangeState{}
	mcs.On("Logger").Return(slog.New(slog.NewTextHandler(io.Discard, nil)))
	mcs.On("Raft").Return(nil)
	return mcs
}

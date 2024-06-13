package store

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/dgate-io/raft"
	"go.uber.org/zap"
)

type FSM struct {
	state  StateManager
	store  *Store
	logger *zap.Logger
}

var _ raft.StateMachine = &FSM{}

type ReadRequest struct {
	Key  string
	Data []byte
}

func (a *ReadRequest) Marshal() ([]byte, error) {
	return json.Marshal(a)
}

func UnmarshalReadRequest(data []byte) (*ReadRequest, error) {
	req := &ReadRequest{}
	if err := json.Unmarshal(data, req); err != nil {
		return nil, err
	}
	return req, nil
}

type ReadResponse struct {
	Success bool
	Data    []byte
	Error   error
}

type ApplyResponse struct {
	Success bool
	Error   error
	Data    []byte
}

// Apply implements raft.StateMachine.
func (s *FSM) Apply(operation *raft.Operation) any {
	switch operation.OperationType {
	case raft.Replicated:
		cl := spec.NewChangeLogFromBytes(operation.Bytes)
		s.logger.Info("change log received applying",
			zap.String("id", cl.ID),
			// zap.Bool("leader", leader),
		)
		err := s.state.ProcessChangeLog(cl, true)
		return &ApplyResponse{
			Success: err == nil,
			Error:   err,
		}
		// TODO: ensure the changelog was already seen/applied
		// return &ApplyResponse{
		// 	Success: true,
		// 	Error:   nil,
		// }
	case raft.Broadcasted:
		req, err := UnmarshalReadRequest(operation.Bytes)
		if err != nil {
			return &ReadResponse{
				Success: false,
				Error:   err,
			}
		}
		s.logger.Info("read request",
			zap.String("key", req.Key),
		)
		switch req.Key {
		case "leaderInfo":
			info := &ServerInfo{}
			if err := json.Unmarshal(req.Data, info); err != nil {
				s.logger.Error("could not parse leaderInfo data",
					zap.String("key", req.Key),
					zap.String("data", string(req.Data)),
				)
			} else {
				s.store.leaderInfo = info
				s.logger.Info("received leader info",
					zap.Any("leaderInfo", info),
				)
			}
		default:
			s.logger.Error("unknown key",
				zap.String("key", req.Key),
			)
		}
		return nil
	default:
		err := errors.New("unknown operation type")
		s.logger.Error(err.Error(),
			zap.Stringer("type", operation.OperationType),
			zap.Uint64("index", operation.LogIndex),
			zap.Uint64("term", operation.LogTerm),
		)
		return &ApplyResponse{
			Success: false,
			Error:   err,
		}
	}
}

// NeedSnapshot implements raft.StateMachine.
func (s *FSM) NeedSnapshot(logSize int) bool {
	return false
}

// Restore implements raft.StateMachine.
func (s *FSM) Restore(snapshotReader io.Reader) error {
	panic("unimplemented")
}

// Snapshot implements raft.StateMachine.
func (s *FSM) Snapshot(snapshotWriter io.Writer) error {
	panic("unimplemented")
}

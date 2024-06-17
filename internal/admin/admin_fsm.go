package admin

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/dgate-io/dgate/internal/admin/changestate"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/hashicorp/raft"
	"go.uber.org/zap"
)

type dgateAdminFSM struct {
	cs     changestate.ChangeState
	logger *zap.Logger
}

var _ raft.BatchingFSM = (*dgateAdminFSM)(nil)

func newDGateAdminFSM(logger *zap.Logger, cs changestate.ChangeState) *dgateAdminFSM {
	return &dgateAdminFSM{cs, logger}
}

func (fsm *dgateAdminFSM) applyLog(log *raft.Log, reload bool) (*spec.ChangeLog, error) {
	switch log.Type {
	case raft.LogCommand:
		var cl spec.ChangeLog
		if err := json.Unmarshal(log.Data, &cl); err != nil {
			fsm.logger.Error("Error unmarshalling change log", zap.Error(err))
			return nil, err
		}

		if cl.ID == "" {
			fsm.logger.Error("Change log ID is empty")
			return nil, errors.New("change log ID is empty")
		} else if cl.Cmd.IsNoop() {
			return nil, nil
		}
		// find a way to only reload if latest index to save time
		return &cl, fsm.cs.ProcessChangeLog(&cl, reload)
	case raft.LogConfiguration:
		servers := raft.DecodeConfiguration(log.Data).Servers
		fsm.logger.Debug("configuration update server",
			zap.Any("address", servers),
			zap.Uint64("index", log.Index),
			zap.Uint64("term", log.Term),
			zap.Time("appended", log.AppendedAt),
		)
	default:
		fsm.logger.Error("Unknown log type in FSM Apply")
	}
	return nil, nil
}

func (fsm *dgateAdminFSM) Apply(log *raft.Log) any {
	rft := fsm.cs.Raft()
	fsm.logger.Debug("apply single log",
		zap.Uint64("applied", rft.AppliedIndex()),
		zap.Uint64("commit", rft.CommitIndex()),
		zap.Uint64("last", rft.LastIndex()),
		zap.Uint64("logIndex", log.Index),
	)
	_, err := fsm.applyLog(log, true)
	return err
}

func (fsm *dgateAdminFSM) ApplyBatch(logs []*raft.Log) []any {
	rft := fsm.cs.Raft()
	lastIndex := len(logs) - 1
	fsm.logger.Debug("apply log batch",
		zap.Uint64("applied", rft.AppliedIndex()),
		zap.Uint64("commit", rft.CommitIndex()),
		zap.Uint64("last", rft.LastIndex()),
		zap.Uint64("log[0]", logs[0].Index),
		zap.Uint64("log[-1]", logs[lastIndex].Index),
		zap.Int("logs", len(logs)),
	)
	results := make([]any, len(logs))
	for i, log := range logs {
		// TODO: check to see if this can be optimized channels raft node provides
		_, err := fsm.applyLog(log, lastIndex == i)
		if err != nil {
			fsm.logger.Error("Error applying log", zap.Error(err))
			results[i] = err
		}
	}
	return results
}

func (fsm *dgateAdminFSM) Snapshot() (raft.FSMSnapshot, error) {
	fsm.cs = nil
	fsm.logger.Warn("snapshots not supported")
	return nil, errors.New("snapshots not supported")
}

func (fsm *dgateAdminFSM) Restore(rc io.ReadCloser) error {
	fsm.logger.Warn("snapshots not supported, cannot restore")
	return nil
}

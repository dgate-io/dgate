package admin

import (
	"encoding/json"
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

func (fsm *dgateAdminFSM) isLatestLog(log *raft.Log) bool {
	rft := fsm.cs.Raft()
	return log.Index == rft.CommitIndex() ||
		log.Index+1 == rft.CommitIndex()
}

func (fsm *dgateAdminFSM) reload(cls ...*spec.ChangeLog) {
	if err := fsm.cs.ReloadState(false, cls...); err != nil {
		fsm.logger.Error("Error processing change log in FSM", zap.Error(err))
	} else {
		fsm.cs.SetReady()
	}
}

func (fsm *dgateAdminFSM) applyLog(log *raft.Log) (*spec.ChangeLog, error) {
	switch log.Type {
	case raft.LogCommand:
		var cl spec.ChangeLog
		if err := json.Unmarshal(log.Data, &cl); err != nil {
			fsm.logger.Error("Error unmarshalling change log", zap.Error(err))
			return nil, err
		} else if cl.Cmd.IsNoop() {
			return nil, nil
		} else if cl.ID == "" {
			fsm.logger.Error("Change log ID is empty", zap.Error(err))
			panic("change log ID is empty")
		}
		// find a way to apply only if latest index to save time
		return &cl, fsm.cs.ProcessChangeLog(&cl, false)
	case raft.LogNoop:
		fsm.logger.Debug("Noop Log - current leader is still leader")
	case raft.LogConfiguration:
		servers := raft.DecodeConfiguration(log.Data).Servers
		for i, server := range servers {
			fsm.logger.Debug("configuration update server",
				zap.Any("address", server.Address),
				zap.Int("index", i),
			)
		}
	case raft.LogBarrier:
		err := fsm.cs.WaitForChanges()
		if err != nil {
			fsm.logger.Error("Error waiting for changes", zap.Error(err))
		}
	default:
		fsm.logger.Error("Unknown log type in FSM Apply")
	}
	return nil, nil
}

func (fsm *dgateAdminFSM) Apply(log *raft.Log) any {
	rft := fsm.cs.Raft()
	fsm.logger.Debug("applying log",
		zap.Uint64("current", log.Index),
		zap.Uint64("applied", rft.AppliedIndex()),
		zap.Uint64("commit", rft.CommitIndex()),
		zap.Uint64("last", rft.LastIndex()),
	)
	cl, err := fsm.applyLog(log)
	if err != nil && !fsm.cs.Ready() {
		fsm.reload(cl)
	} else {
		fsm.logger.Error("Error processing change log in FSM", zap.Error(err))
	}
	return err
}

func (fsm *dgateAdminFSM) ApplyBatch(logs []*raft.Log) []any {
	if len(logs) == 0 || logs == nil {
		fsm.logger.Warn("No logs to apply in ApplyBatch")
		return nil
	}
	lastLog := logs[len(logs)-1]
	rft := fsm.cs.Raft()
	fsm.logger.Info("applying batch logs",
		zap.Int("size", len(logs)),
		zap.Uint64("current", lastLog.Index),
		zap.Uint64("applied", rft.AppliedIndex()),
		zap.Uint64("commit", rft.CommitIndex()),
		zap.Uint64("last", rft.LastIndex()),
	)

	cls := make([]*spec.ChangeLog, 0, len(logs))
	results := make([]any, len(logs))
	for i, log := range logs {
		var (
			cl  *spec.ChangeLog
			err error
		)
		if cl, err = fsm.applyLog(log); err != nil {
			results[i] = err
			fsm.logger.Error("Error processing change log in FSM", zap.Error(err))
		} else {
			cls = append(cls, cl)
		}
	}

	if fsm.cs.Ready() || fsm.isLatestLog(lastLog) {
		fsm.reload(cls...)
	}

	return results
}

func (fsm *dgateAdminFSM) Snapshot() (raft.FSMSnapshot, error) {
	panic("snapshots not supported")
}

func (fsm *dgateAdminFSM) Restore(rc io.ReadCloser) error {
	panic("snapshots not supported")
}

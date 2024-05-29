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

func (fsm *dgateAdminFSM) isReplay(log *raft.Log) bool {
	return !fsm.cs.Ready() &&
		log.Index+1 >= fsm.cs.Raft().LastIndex() &&
		log.Index+1 >= fsm.cs.Raft().AppliedIndex()
}

func (fsm *dgateAdminFSM) checkLast(log *raft.Log) {
	rft := fsm.cs.Raft()
	if !fsm.cs.Ready() && fsm.isReplay(log) {
		fsm.logger.Info("FSM is not ready, setting ready",
			zap.Uint64("index", log.Index),
			zap.Uint64("applied-index", rft.AppliedIndex()),
			zap.Uint64("last-index", rft.LastIndex()),
		)
		defer func() {
			if err := fsm.cs.ReloadState(false); err != nil {
				fsm.logger.Error("Error processing change log in FSM", zap.Error(err))
			} else {
				fsm.cs.SetReady()
			}
		}()
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
	defer fsm.checkLast(log)
	_, err := fsm.applyLog(log)
	return err
}

func (fsm *dgateAdminFSM) ApplyBatch(logs []*raft.Log) []any {
	lastLog := logs[len(logs)-1]
	if fsm.isReplay(lastLog) {
		rft := fsm.cs.Raft()
		fsm.logger.Info("applying log batch logs",
			zap.Int("size", len(logs)),
			zap.Uint64("current", lastLog.Index),
			zap.Uint64("applied", rft.AppliedIndex()),
			zap.Uint64("commit", rft.CommitIndex()),
			zap.Uint64("last", rft.LastIndex()),
		)
	}
	cls := make([]*spec.ChangeLog, 0, len(logs))
	defer func() {
		if !fsm.cs.Ready() {
			fsm.checkLast(logs[len(logs)-1])
			return
		}

		if err := fsm.cs.ReloadState(true, cls...); err != nil {
			fsm.logger.Error("Error reloading state @ FSM ApplyBatch", zap.Error(err))
		}
	}()

	results := make([]any, len(logs))
	for i, log := range logs {
		var cl *spec.ChangeLog
		cl, results[i] = fsm.applyLog(log)
		if cl != nil {
			cls = append(cls, cl)
		}
	}
	return results
}

func (fsm *dgateAdminFSM) Snapshot() (raft.FSMSnapshot, error) {
	panic("snapshots not supported")
}

func (fsm *dgateAdminFSM) Restore(rc io.ReadCloser) error {
	panic("snapshots not supported")
}

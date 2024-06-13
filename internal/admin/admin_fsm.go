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
	index  uint64
}

var _ raft.BatchingFSM = (*dgateAdminFSM)(nil)

func newDGateAdminFSM(logger *zap.Logger, cs changestate.ChangeState) *dgateAdminFSM {
	return &dgateAdminFSM{cs, logger, 0}
}

func (fsm *dgateAdminFSM) SetIndex(index uint64) {
	fsm.index = index
}

func (fsm *dgateAdminFSM) applyLog(log *raft.Log, replay bool) (*spec.ChangeLog, error) {
	log.Index = fsm.index
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
		// find a way to only reload if latest index to save time
		return &cl, fsm.cs.ProcessChangeLog(&cl, replay)
	case raft.LogConfiguration:
		servers := raft.DecodeConfiguration(log.Data).Servers
		for i, server := range servers {
			fsm.logger.Debug("configuration update server",
				zap.Any("address", server.Address),
				zap.Int("index", i),
			)
		}
	default:
		fsm.logger.Error("Unknown log type in FSM Apply")
	}
	return nil, nil
}

func (fsm *dgateAdminFSM) Apply(log *raft.Log) any {
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
		zap.Uint64("fsmLastIndex", fsm.index),
		zap.Uint64("log[0]", logs[0].Index),
		zap.Uint64("log[-1]", logs[lastIndex].Index),
		zap.Int("logs", len(logs)),
	)
	results := make([]any, len(logs))
	for i, log := range logs {
		// TODO: check to see if this can be optimized channels raft node provides
		_, results[i] = fsm.applyLog(
			log, lastIndex == i,
		)
	}
	return results
}

func (fsm *dgateAdminFSM) Snapshot() (raft.FSMSnapshot, error) {
	panic("snapshots not supported")
}

func (fsm *dgateAdminFSM) Restore(rc io.ReadCloser) error {
	panic("snapshots not supported")
}

package admin

import (
	"encoding/json"
	"io"
	"log/slog"

	"github.com/dgate-io/dgate/internal/admin/changestate"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/hashicorp/raft"
)

type dgateAdminFSM struct {
	cs     changestate.ChangeState
	logger *slog.Logger
}

var _ raft.BatchingFSM = (*dgateAdminFSM)(nil)

func newDGateAdminFSM(cs changestate.ChangeState) *dgateAdminFSM {
	dgateFSMLogger := cs.Logger().WithGroup("admin-raft-fsm")
	return &dgateAdminFSM{
		cs:     cs,
		logger: dgateFSMLogger,
	}
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
			"Index", log.Index,
			"AIndex", rft.AppliedIndex(),
			"LIndex", rft.LastIndex(),
		)
		defer func() {
			if err := fsm.cs.ReloadState(false); err != nil {
				fsm.logger.Error("Error processing change log in FSM",
					"error", err,
				)
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
			fsm.logger.Error("Error unmarshalling change log")
			return nil, err
		} else if cl.Cmd.IsNoop() {
			return nil, nil
		} else if cl.ID == "" {
			fsm.logger.Error("Change log ID is empty")
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
				"address", server.Address, "index", i,
			)
		}
	case raft.LogBarrier:
		err := fsm.cs.WaitForChanges()
		if err != nil {
			fsm.logger.Error("Error waiting for changes", "error", err)
		}
	default:
		fsm.cs.Logger().Error("Unknown log type in FSM Apply")
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
			"size", len(logs),
			"current", lastLog.Index,
			"applied", rft.AppliedIndex(),
			"commit", rft.CommitIndex(),
			"last", rft.LastIndex(),
		)
	}
	cls := make([]*spec.ChangeLog, 0, len(logs))
	defer func() {
		if !fsm.cs.Ready() {
			fsm.checkLast(logs[len(logs)-1])
			return
		}

		if err := fsm.cs.ReloadState(true, cls...); err != nil {
			fsm.logger.Error("Error reloading state @ FSM ApplyBatch")
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

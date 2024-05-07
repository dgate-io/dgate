package admin

import (
	"encoding/json"
	"io"

	"github.com/dgate-io/dgate/internal/proxy"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/hashicorp/raft"
	"github.com/rs/zerolog"
)

type dgateAdminFSM struct {
	ps     *proxy.ProxyState
	logger zerolog.Logger
}

var _ raft.BatchingFSM = (*dgateAdminFSM)(nil)

func newDGateAdminFSM(ps *proxy.ProxyState) *dgateAdminFSM {
	dgateFSMLogger := ps.Logger().With().
		Str("component", "dgateAdminFSM").
		Logger()
	return &dgateAdminFSM{
		ps:     ps,
		logger: dgateFSMLogger,
	}
}

func (fsm *dgateAdminFSM) isReplay(log *raft.Log) bool {
	return !fsm.ps.Ready() &&
		log.Index+1 >= fsm.ps.Raft().LastIndex() &&
		log.Index+1 >= fsm.ps.Raft().AppliedIndex()
}

func (fsm *dgateAdminFSM) checkLast(log *raft.Log) {
	rft := fsm.ps.Raft()
	if !fsm.ps.Ready() && fsm.isReplay(log) {
		fsm.logger.Info().
			Msgf("FSM is not ready, setting ready @ Index: %d, AIndex: %d, LIndex: %d",
				log.Index, rft.AppliedIndex(), rft.LastIndex())
		defer func() {
			if err := fsm.ps.ReloadState(false); err != nil {
				fsm.logger.Error().Err(err).
					Msg("Error processing change log in FSM")
			} else {
				fsm.ps.SetReady()
			}
		}()
	}
}

func (fsm *dgateAdminFSM) applyLog(log *raft.Log) (*spec.ChangeLog, error) {
	rft := fsm.ps.Raft()
	switch log.Type {
	case raft.LogCommand:
		fsm.logger.Debug().
			Msgf("log cmd: %d, %v, %s - applied: %v, latest: %v",
				log.Index, log.Type, log.Data,
				rft.AppliedIndex(), rft.LastIndex())
		var cl spec.ChangeLog
		if err := json.Unmarshal(log.Data, &cl); err != nil {
			fsm.logger.Error().Err(err).
				Msg("Error unmarshalling change log")
			return nil, err
		} else if cl.Cmd.IsNoop() {
			return nil, nil
		} else if cl.ID == "" {
			fsm.logger.Error().
				Msg("Change log ID is empty")
			panic("change log ID is empty")
		}
		// find a way to apply only if latest index to save time
		return &cl, fsm.ps.ProcessChangeLog(&cl, false)
	case raft.LogNoop:
		fsm.logger.Debug().Msg("Noop Log - current leader is still leader")
	case raft.LogConfiguration:
		servers := raft.DecodeConfiguration(log.Data).Servers
		for i, server := range servers {
			fsm.logger.Debug().
				Msgf("%d: config update - server: %s", i, server.Address)
		}
	case raft.LogBarrier:
		err := fsm.ps.WaitForChanges()
		if err != nil {
			fsm.logger.Err(err).
				Msg("Error waiting for changes")
		}
	default:
		fsm.ps.Logger().Error().
			Msg("Unknown log type in FSM Apply")
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
		rft := fsm.ps.Raft()
		fsm.logger.Info().
			Msgf("applying log batch of %d logs - current:%d applied:%d commit:%d last:%d",
				len(logs), lastLog.Index, rft.AppliedIndex(), rft.CommitIndex(), rft.LastIndex())
	}
	cls := make([]*spec.ChangeLog, 0, len(logs))
	defer func() {
		if !fsm.ps.Ready() {
			fsm.checkLast(logs[len(logs)-1])
			return
		}

		if err := fsm.ps.ReloadState(true, cls...); err != nil {
			fsm.logger.Error().Err(err).
				Msg("Error reloading state @ FSM ApplyBatch")
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

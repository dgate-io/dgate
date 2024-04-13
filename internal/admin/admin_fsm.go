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

func (fsm *dgateAdminFSM) Apply(log *raft.Log) interface{} {
	switch log.Type {
	case raft.LogCommand:
		rft := fsm.ps.Raft()
		fsm.logger.Debug().
			Msgf("log cmd: %d, %v, %s - applied: %v, latest: %v",
				log.Index, log.Type, log.Data,
				rft.AppliedIndex(), rft.LastIndex())
		var cl spec.ChangeLog
		err := json.Unmarshal(log.Data, &cl)
		if err != nil {
			fsm.logger.Error().Err(err).
				Msg("Error unmarshalling change log")
			return err
		}
		// find a way to apply only if latest index to save time
		return fsm.ps.ProcessChangeLog(&cl, false)
	case raft.LogNoop:
		fsm.logger.Debug().Msg("Noop Log - current leader is still leader")
	case raft.LogConfiguration:
		servers := raft.DecodeConfiguration(log.Data).Servers
		for i, server := range servers {
			fsm.logger.Debug().
				Msgf("%d: config update - server: %s", i, server.Address)
		}
	case raft.LogBarrier:
		fsm.ps.WaitForChanges()
	default:
		fsm.ps.Logger().Error().
			Msg("Unknown log type in FSM Apply")
	}
	return nil
}

func (fsm *dgateAdminFSM) ApplyBatch(logs []*raft.Log) []interface{} {
	fsm.logger.Debug().
		Msgf("applying log batch of %d logs", len(logs))
	results := make([]interface{}, len(logs))
	for i, log := range logs {
		results[i] = fsm.Apply(log)
	}
	defer func() {
		if err := fsm.ps.ReloadState(); err != nil {
			fsm.logger.Error().Err(err).
				Msg("Error reloading state")
		}
	}()
	return results
}

func (fsm *dgateAdminFSM) Snapshot() (raft.FSMSnapshot, error) {
	return &dgateAdminState{
		snap: fsm.ps.Snapshot(),
	}, nil
}

func (fsm *dgateAdminFSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()
	return fsm.ps.RestoreState(rc)
}

type dgateAdminState struct {
	snap  *proxy.ProxySnapshot
	psRef *proxy.ProxyState
}

func (state *dgateAdminState) Persist(sink raft.SnapshotSink) error {
	defer sink.Close()
	return state.snap.PersistState(sink)
}

func (state *dgateAdminState) Release() {
	state.psRef = nil
	state.snap = nil
}

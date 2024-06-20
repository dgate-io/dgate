package raftadmin

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"path"
	"sync"
	"time"

	"github.com/hashicorp/raft"
	"go.uber.org/zap"
)

// Server provides a HTTP-based transport that can be used to
// communicate with Raft on remote machines. It is convenient to use if your
// application is an HTTP server already and you do not want to use multiple
// different transports (if not, you can use raft.NetworkTransport).
type Server struct {
	logger *zap.Logger
	r      *raft.Raft
	addrs []raft.ServerAddress
}

// NewServer creates a new HTTP transport on the given addr.
func NewServer(r *raft.Raft, logger *zap.Logger, addrs []raft.ServerAddress) *Server {
	return &Server{
		logger: logger,
		r:      r,
		addrs:  addrs,
	}
}

func unmarshalBody[T any](req *http.Request, out T) error {
	buf, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(buf, out)
}

func timeout(ctx context.Context) time.Duration {
	if dl, ok := ctx.Deadline(); ok {
		return time.Until(dl)
	}
	return 0
}

var (
	mtx        sync.Mutex
	operations = map[string]*future{}
)

type future struct {
	rf  raft.Future
	mtx sync.Mutex
}

func toFuture(f raft.Future) (*Future, error) {
	token := fmt.Sprintf("%x", sha1.Sum([]byte(fmt.Sprintf("%d", rand.Uint64()))))
	mtx.Lock()
	operations[token] = &future{rf: f}
	mtx.Unlock()
	return &Future{
		OperationToken: token,
	}, nil
}

func (a *Server) Await(ctx context.Context, req *Future) (*AwaitResponse, error) {
	mtx.Lock()
	f, ok := operations[req.OperationToken]
	defer func() {
		mtx.Lock()
		delete(operations, req.OperationToken)
		mtx.Unlock()
	}()
	mtx.Unlock()
	if !ok {
		return nil, fmt.Errorf("token %q unknown", req.OperationToken)
	}
	f.mtx.Lock()
	errChan := make(chan error, 1)
	go func() {
		errChan <- f.rf.Error()
	}()
	select {
	case <-ctx.Done():
		f.mtx.Unlock()
		return nil, ctx.Err()
	case err := <-errChan:
		f.mtx.Unlock()
		if err != nil {
			return &AwaitResponse{
				Error: err.Error(),
			}, nil
		}
	}
	r := &AwaitResponse{}
	if ifx, ok := f.rf.(raft.IndexFuture); ok {
		r.Index = ifx.Index()
	}
	return r, nil
}

func (a *Server) Forget(ctx context.Context, req *Future) (*ForgetResponse, error) {
	mtx.Lock()
	delete(operations, req.OperationToken)
	mtx.Unlock()
	return &ForgetResponse{
		OperationToken: req.OperationToken,
	}, nil
}

func (a *Server) AddNonvoter(ctx context.Context, req *AddNonvoterRequest) (*Future, error) {
	return toFuture(a.r.AddNonvoter(raft.ServerID(req.ID), raft.ServerAddress(req.Address), uint64(req.PrevIndex), timeout(ctx)))
}

func (a *Server) AddVoter(ctx context.Context, req *AddVoterRequest) (*Future, error) {
	return toFuture(a.r.AddVoter(raft.ServerID(req.ID), raft.ServerAddress(req.Address), uint64(req.PrevIndex), timeout(ctx)))
}

func (a *Server) AppliedIndex(ctx context.Context) (*AppliedIndexResponse, error) {
	return &AppliedIndexResponse{
		Index: a.r.AppliedIndex(),
	}, nil
}

func (a *Server) Barrier(ctx context.Context) (*Future, error) {
	return toFuture(a.r.Barrier(timeout(ctx)))
}

func (a *Server) DemoteVoter(ctx context.Context, req *DemoteVoterRequest) (*Future, error) {
	return toFuture(a.r.DemoteVoter(raft.ServerID(req.ID), req.PrevIndex, timeout(ctx)))
}

func (a *Server) GetConfiguration(ctx context.Context) (*GetConfigurationResponse, error) {
	f := a.r.GetConfiguration()
	if err := f.Error(); err != nil {
		return nil, err
	}
	resp := &GetConfigurationResponse{}
	for _, s := range f.Configuration().Servers {
		cs := &GetConfigurationServer{
			ID:      string(s.ID),
			Address: string(s.Address),
		}
		switch s.Suffrage {
		case raft.Voter:
			cs.Suffrage = RaftSuffrageVoter
		case raft.Nonvoter, raft.Staging:
			cs.Suffrage = RaftSuffrageNonvoter
		default:
			return nil, fmt.Errorf("unknown server suffrage %v for server %q", s.Suffrage, s.ID)
		}
		resp.Servers = append(resp.Servers, cs)
	}
	return resp, nil
}

func (a *Server) LastContact(ctx context.Context) (*LastContactResponse, error) {
	t := a.r.LastContact()
	return &LastContactResponse{
		UnixNano: t.UnixNano(),
	}, nil
}

func (a *Server) LastIndex(ctx context.Context) (*LastIndexResponse, error) {
	return &LastIndexResponse{
		Index: a.r.LastIndex(),
	}, nil
}

func (a *Server) CurrentNodeIsLeader(ctx context.Context) bool {
	return a.r.State() == raft.Leader
}

func (a *Server) Leader(ctx context.Context) (*LeaderResponse, error) {
	for _, s := range a.r.GetConfiguration().Configuration().Servers {
		if s.Suffrage == raft.Voter && s.Address == a.r.Leader() {
			return &LeaderResponse{
				ID:      string(s.ID),
				Address: string(s.Address),
			}, nil
		}
	}
	return &LeaderResponse{
		Address: string(a.r.Leader()),
	}, nil
}

func (a *Server) LeadershipTransfer(ctx context.Context) (*Future, error) {
	return toFuture(a.r.LeadershipTransfer())
}

func (a *Server) LeadershipTransferToServer(ctx context.Context, req *LeadershipTransferToServerRequest) (*Future, error) {
	return toFuture(a.r.LeadershipTransferToServer(raft.ServerID(req.ID), raft.ServerAddress(req.Address)))
}

func (a *Server) RemoveServer(ctx context.Context, req *RemoveServerRequest) (*Future, error) {
	return toFuture(a.r.RemoveServer(raft.ServerID(req.ID), req.PrevIndex, timeout(ctx)))
}

func (a *Server) Shutdown(ctx context.Context) (*Future, error) {
	return toFuture(a.r.Shutdown())
}

func (a *Server) Snapshot(ctx context.Context) (*Future, error) {
	return toFuture(a.r.Snapshot())
}

func (a *Server) State(ctx context.Context) (*StateResponse, error) {
	switch s := a.r.State(); s {
	case raft.Follower:
		return &StateResponse{State: RaftStateFollower}, nil
	case raft.Candidate:
		return &StateResponse{State: RaftStateCandidate}, nil
	case raft.Leader:
		return &StateResponse{State: RaftStateLeader}, nil
	case raft.Shutdown:
		return &StateResponse{State: RaftStateShutdown}, nil
	default:
		return nil, fmt.Errorf("unknown raft state %v", s)
	}
}

func (a *Server) Stats(ctx context.Context) (*StatsResponse, error) {
	ret := &StatsResponse{}
	ret.Stats = map[string]string{}
	for k, v := range a.r.Stats() {
		ret.Stats[k] = v
	}
	return ret, nil
}

func (a *Server) VerifyLeader(ctx context.Context) (*Future, error) {
	return toFuture(a.r.VerifyLeader())
}

// ServeHTTP implements the net/http.Handler interface, so that you can use
func (t *Server) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	cmd := path.Base(req.URL.Path)

	if cmdRequiresLeader(cmd) && t.r.State() != raft.Leader {
		leaderAddr, _ := t.r.LeaderWithID()
		if leaderAddr == "" {
			http.Error(res, "no leader", http.StatusServiceUnavailable)
			return
		}
		req.URL.Host = string(leaderAddr)
		http.Redirect(res, req, req.URL.String(),
			http.StatusTemporaryRedirect)
		return
	}

	switch cmd {
	case "AddNonvoter":
		var body AddNonvoterRequest
		err := unmarshalBody(req, &body)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		f, err := t.AddNonvoter(req.Context(), &body)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		t.genericResponse(req, res, f, cmd)
		return
	case "AddVoter":
		var body AddVoterRequest
		err := unmarshalBody(req, &body)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		f, err := t.AddVoter(req.Context(), &body)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		t.genericResponse(req, res, f, cmd)
		return
	case "AppliedIndex":
		resp, err := t.AppliedIndex(req.Context())
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		if err = json.NewEncoder(res).Encode(resp); err != nil {
			t.logger.Error("error occurred when handling command", zap.String("command", cmd), zap.Error(err))
		}
		return
	case "Barrier":
		f, err := t.Barrier(req.Context())
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		resp, err := t.Await(req.Context(), f)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		if resp.Error != "" {
			http.Error(res, resp.Error, http.StatusBadRequest)
			return
		}
		res.Header().Set("X-Raft-Index", fmt.Sprintf("%d", resp.Index))
		res.WriteHeader(http.StatusAccepted)
		if err = json.NewEncoder(res).Encode(resp); err != nil {
			t.logger.Error("error occurred when handling command", zap.String("command", cmd), zap.Error(err))
		}
		return
	case "DemoteVoter":
		var body DemoteVoterRequest
		err := unmarshalBody(req, &body)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		f, err := t.DemoteVoter(req.Context(), &body)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		t.genericResponse(req, res, f, cmd)
		return
	case "GetConfiguration":
		resp, err := t.GetConfiguration(req.Context())
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		if err = json.NewEncoder(res).Encode(resp); err != nil {
			t.logger.Error("error occurred when handling command", zap.String("command", cmd), zap.Error(err))
		}
		return
	case "LastContact":
		resp, err := t.LastContact(req.Context())
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		if err = json.NewEncoder(res).Encode(resp); err != nil {
			t.logger.Error("error occurred when handling command", zap.String("command", cmd), zap.Error(err))
		}
		return
	case "LastIndex":
		resp, err := t.LastIndex(req.Context())
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		if err = json.NewEncoder(res).Encode(resp); err != nil {
			t.logger.Error("error occurred when handling command", zap.String("command", cmd), zap.Error(err))
		}
		return
	case "Leader":
		resp, err := t.Leader(req.Context())
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		if err = json.NewEncoder(res).Encode(resp); err != nil {
			t.logger.Error("error occurred when handling command", zap.String("command", cmd), zap.Error(err))
		}
		return
	case "LeadershipTransfer":
		f, err := t.LeadershipTransfer(req.Context())
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		t.genericResponse(req, res, f, cmd)
		return
	case "LeadershipTransferToServer":
		var body LeadershipTransferToServerRequest
		err := unmarshalBody(req, &body)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		f, err := t.LeadershipTransferToServer(req.Context(), &body)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		t.genericResponse(req, res, f, cmd)
		return
	case "RemoveServer":
		var body RemoveServerRequest
		err := unmarshalBody(req, &body)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		f, err := t.RemoveServer(req.Context(), &body)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		t.genericResponse(req, res, f, cmd)
		return
	case "Shutdown":
		f, err := t.Shutdown(req.Context())
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		t.genericResponse(req, res, f, cmd)
		return
	case "Snapshot":
		f, err := t.Snapshot(req.Context())
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		t.genericResponse(req, res, f, cmd)
		return
	case "State":
		resp, err := t.State(req.Context())
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		if err = json.NewEncoder(res).Encode(resp); err != nil {
			t.logger.Error("error occurred when handling command", zap.String("command", cmd), zap.Error(err))
		}
		return
	case "Stats":
		resp, err := t.Stats(req.Context())
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		if err = json.NewEncoder(res).Encode(resp); err != nil {
			t.logger.Error("error occurred when handling command", zap.String("command", cmd), zap.Error(err))
		}
		return
	case "VerifyLeader":
		f, err := t.VerifyLeader(req.Context())
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		t.genericResponse(req, res, f, cmd)
		return
	default:
		err := fmt.Errorf("unknown command %q", cmd)
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
}

func cmdRequiresLeader(cmd string) bool {
	switch cmd {
	case "GetConfiguration", "AppliedIndex", "LastContact", "LastIndex", "Leader", "State", "Stats":
		return false
	default:
		return true
	}
}

func (t *Server) genericResponse(req *http.Request, res http.ResponseWriter, f *Future, cmd string) {
	resp, err := t.Await(req.Context(), f)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	if resp.Error != "" {
		http.Error(res, resp.Error, http.StatusBadRequest)
		return
	}
	res.Header().Set("X-Raft-Index", fmt.Sprintf("%d", resp.Index))
	res.WriteHeader(http.StatusAccepted)
	err = json.NewEncoder(res).Encode(resp)
	if err != nil {
		t.logger.Error("error occurred when handling command", zap.String("command", cmd), zap.Error(err))
	}
}

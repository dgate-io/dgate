package rafthttp_test

import (
	"io"
	"log"
	"net"
	"net/http"
	"testing"
	"time"

	"log/slog"

	"github.com/dgate-io/dgate/pkg/rafthttp"
	"github.com/dgate-io/dgate/pkg/util/logger"
	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/mock"
)

type MockFSM struct {
	mock.Mock
}

var _ raft.FSM = (*MockFSM)(nil)

func (m *MockFSM) Apply(l *raft.Log) interface{} {
	args := m.Called(l)
	return args.Get(0)
}

func (m *MockFSM) Snapshot() (raft.FSMSnapshot, error) {
	args := m.Called()
	return args.Get(0).(raft.FSMSnapshot), args.Error(1)
}

func (m *MockFSM) Restore(io.ReadCloser) error {
	args := m.Called()
	return args.Error(0)
}

func TestExample(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	log.Printf("Listening on %s", ln.Addr().String())
	srvAddr := raft.ServerAddress(ln.Addr().String())
	lgr := slog.New(slog.NewTextHandler(io.Discard, nil))
	transport := rafthttp.NewHTTPTransport(
		srvAddr, http.DefaultClient, lgr,
		"http://(address)/raft",
	)
	srv := &http.Server{
		Handler: transport,
	}
	go srv.Serve(ln)

	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = "1"
	raftConfig.Logger = logger.NewSLogHCAdapter(lgr)

	mockFSM := &MockFSM{}
	logStore := raft.NewInmemStore()
	stableStore := raft.NewInmemStore()
	snapStore := raft.NewInmemSnapshotStore()

	raftNode, err := raft.NewRaft(
		raftConfig, mockFSM,
		logStore, stableStore, snapStore,
		transport,
	)
	if err != nil {
		t.Fatal(err)
	}

	raftNode.Apply([]byte("foo"), time.Duration(0))
}

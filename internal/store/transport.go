package store

import "github.com/dgate-io/raft"

type StoreTransport struct {
	t raft.Transport
}

func NewStoreTransport(t raft.Transport) *StoreTransport {
	return &StoreTransport{t: t}
}

// Address implements raft.Transport.
func (c *StoreTransport) Address() string {
	return c.t.Address()
}

// DecodeConfiguration implements raft.Transport.
func (c *StoreTransport) DecodeConfiguration(data []byte) (raft.Configuration, error) {
	return c.t.DecodeConfiguration(data)
}

// EncodeConfiguration implements raft.Transport.
func (c *StoreTransport) EncodeConfiguration(configuration *raft.Configuration) ([]byte, error) {
	return c.t.EncodeConfiguration(configuration)
}

// RegisterAppendEntriesHandler implements raft.Transport.
func (c *StoreTransport) RegisterAppendEntriesHandler(handler func(*raft.AppendEntriesRequest, *raft.AppendEntriesResponse) error) {
	c.t.RegisterAppendEntriesHandler(handler)
}

// RegisterRequestVoteHandler implements raft.Transport.
func (c *StoreTransport) RegisterRequestVoteHandler(handler func(*raft.RequestVoteRequest, *raft.RequestVoteResponse) error) {
	c.t.RegisterRequestVoteHandler(handler)
}

// RegisterInstallSnapshotHandler implements raft.Transport.
func (c *StoreTransport) RegisterInstallSnapshotHandler(handler func(*raft.InstallSnapshotRequest, *raft.InstallSnapshotResponse) error) {
	c.t.RegisterInstallSnapshotHandler(handler)
}

// Run implements raft.Transport.
func (c *StoreTransport) Run() error {
	return c.t.Run()
}

// SendAppendEntries implements raft.Transport.
func (c *StoreTransport) SendAppendEntries(address string, request raft.AppendEntriesRequest) (raft.AppendEntriesResponse, error) {
	return c.t.SendAppendEntries(address, request)
}

// SendInstallSnapshot implements raft.Transport.
func (c *StoreTransport) SendInstallSnapshot(address string, request raft.InstallSnapshotRequest) (raft.InstallSnapshotResponse, error) {
	return c.t.SendInstallSnapshot(address, request)
}

// SendRequestVote implements raft.Transport.
func (c *StoreTransport) SendRequestVote(address string, request raft.RequestVoteRequest) (raft.RequestVoteResponse, error) {
	return c.t.SendRequestVote(address, request)
}

// Shutdown implements raft.Transport.
func (c *StoreTransport) Shutdown() error {
	return c.t.Shutdown()
}

var _ raft.Transport = &StoreTransport{}

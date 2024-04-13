package proxy

import "github.com/hashicorp/raft"

type ProxyReplication struct {
	raft       *raft.Raft
	raftConfig *raft.Config
}

func NewProxyReplication(raft *raft.Raft, raftConfig *raft.Config) *ProxyReplication {
	return &ProxyReplication{
		raft:       raft,
		raftConfig: raftConfig,
	}
}

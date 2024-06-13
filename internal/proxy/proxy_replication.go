package proxy

import "github.com/hashicorp/raft"

type ProxyReplication struct {
	raft *raft.Raft
}

func NewProxyReplication(raft *raft.Raft) *ProxyReplication {
	return &ProxyReplication{
		raft: raft,
	}
}

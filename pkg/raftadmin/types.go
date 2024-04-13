package raftadmin

type Future struct {
	OperationToken string
}

type AwaitResponse struct {
	Index uint64 `json:"index"`
	Error string `json:"error"`
}

type ForgetResponse struct {
	OperationToken string `json:"operation_token"`
}

type AddNonvoterRequest struct {
	ID        string `json:"id"`
	Address   string `json:"address"`
	PrevIndex int64  `json:"prev_index"`
}

type AddVoterRequest struct {
	ID        string `json:"id"`
	Address   string `json:"address"`
	PrevIndex int64  `json:"prev_index"`
}

type AppliedIndexResponse struct {
	Index uint64 `json:"index"`
}

type ApplyLogRequest struct {
	Data       []byte `json:"data"`
	Extensions []byte `json:"extensions"`
}

type DemoteVoterRequest struct {
	ID        string `json:"id"`
	PrevIndex uint64 `json:"prev_index"`
}

type RaftSuffrage string

const (
	RaftSuffrageVoter    RaftSuffrage = "voter"
	RaftSuffrageNonvoter RaftSuffrage = "nonvoter"
)

type GetConfigurationResponse struct {
	Servers []*GetConfigurationServer `json:"servers"`
}

type GetConfigurationServer struct {
	Suffrage RaftSuffrage `json:"suffrage"`
	ID       string       `json:"id"`
	Address  string       `json:"address"`
}

type LastContactResponse struct {
	UnixNano int64 `json:"unix_nano"`
}

type LastIndexResponse struct {
	Index uint64 `json:"index"`
}

type LeaderResponse struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

type LeadershipTransferToServerRequest struct {
	ID      string `json:"id"`
	Address string `json:"address"`
}

type RemoveServerRequest struct {
	ID        string `json:"id"`
	PrevIndex uint64 `json:"prev_index"`
}

type RaftState string

const (
	RaftStateLeader    RaftState = "leader"
	RaftStateFollower  RaftState = "follower"
	RaftStateCandidate RaftState = "candidate"
	RaftStateShutdown  RaftState = "shutdown"
)

type StateResponse struct {
	Index int64 `json:"index"`
	State RaftState
}

type StatsResponse struct {
	Stats map[string]string `json:"stats"`
}

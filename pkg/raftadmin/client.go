package raftadmin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/raft"
	"go.uber.org/zap"
)

type Doer func(*http.Request) (*http.Response, error)

type Client struct {
	do     Doer
	scheme string
	logger *zap.Logger
}

func NewClient(doer Doer, logger *zap.Logger, scheme string) *Client {
	if doer == nil {
		doer = http.DefaultClient.Do
	}
	if scheme == "" {
		scheme = "http"
	}
	return &Client{
		do:     doer,
		scheme: scheme,
		logger: logger,
	}
}

func (c *Client) generateUrl(target raft.ServerAddress, action string) string {
	uri := fmt.Sprintf("%s://%s/raftadmin/%s", c.scheme, target, action)
	// c.logger.Debug("raftadmin: generated url", zap.String("url", uri))
	return uri
}
func (c *Client) AddNonvoter(ctx context.Context, target raft.ServerAddress, req *AddNonvoterRequest) (*AwaitResponse, error) {
	url := c.generateUrl(target, "AddNonvoter")
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequest("POST", url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(ctx, r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("unexpected HTTP status code: %v", res.Status)
	}
	var out AwaitResponse
	err = json.NewDecoder(res.Body).Decode(&out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) AddVoter(ctx context.Context, target raft.ServerAddress, req *AddVoterRequest) (*AwaitResponse, error) {
	url := c.generateUrl(target, "AddVoter")
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequest("POST", url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(ctx, r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("unexpected HTTP status code: %v", res.Status)
	}
	var out AwaitResponse
	err = json.NewDecoder(res.Body).Decode(&out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) AppliedIndex(ctx context.Context, target raft.ServerAddress) (*AppliedIndexResponse, error) {
	url := c.generateUrl(target, "AppliedIndex")
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(ctx, r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status code: %v", res.Status)
	}
	var out AppliedIndexResponse
	err = json.NewDecoder(res.Body).Decode(&out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ApplyLog(ctx context.Context, target raft.ServerAddress, req *ApplyLogRequest) (*AwaitResponse, error) {
	url := c.generateUrl(target, "ApplyLog")
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequest("POST", url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(ctx, r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("unexpected HTTP status code: %v", res.Status)
	}
	var out AwaitResponse
	err = json.NewDecoder(res.Body).Decode(&out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Barrier(ctx context.Context, target raft.ServerAddress) (*AwaitResponse, error) {
	url := c.generateUrl(target, "Barrier")
	r, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(ctx, r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("unexpected HTTP status code: %v", res.Status)
	}
	var out AwaitResponse
	err = json.NewDecoder(res.Body).Decode(&out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DemoteVoter(ctx context.Context, target raft.ServerAddress, req *DemoteVoterRequest) (*AwaitResponse, error) {
	url := c.generateUrl(target, "DemoteVoter")
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequest("POST", url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(ctx, r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("unexpected HTTP status code: %v", res.Status)
	}
	var out AwaitResponse
	err = json.NewDecoder(res.Body).Decode(&out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetConfiguration(ctx context.Context, target raft.ServerAddress) (*GetConfigurationResponse, error) {
	url := c.generateUrl(target, "GetConfiguration")
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(ctx, r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status code: %v", res.Status)
	}
	var out GetConfigurationResponse
	err = json.NewDecoder(res.Body).Decode(&out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) LastContact(ctx context.Context, target raft.ServerAddress) (*LastContactResponse, error) {
	url := c.generateUrl(target, "LastContact")
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(ctx, r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status code: %v", res.Status)
	}
	var out LastContactResponse
	err = json.NewDecoder(res.Body).Decode(&out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) LastIndex(ctx context.Context, target raft.ServerAddress) (*LastIndexResponse, error) {
	url := c.generateUrl(target, "LastIndex")
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(ctx, r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status code: %v", res.Status)
	}
	var out LastIndexResponse
	err = json.NewDecoder(res.Body).Decode(&out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

var ErrNotLeader = errors.New("not leader")

func (c *Client) Leader(ctx context.Context, target raft.ServerAddress) (*LeaderResponse, error) {
	url := c.generateUrl(target, "Leader")
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(ctx, r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		var out LeaderResponse
		err = json.NewDecoder(res.Body).Decode(&out)
		if err != nil {
			return nil, err
		}
		return &out, nil
	case http.StatusTemporaryRedirect:
		return nil, ErrNotLeader
	default:
		return nil, fmt.Errorf("unexpected HTTP status code: %v", res.Status)
	}
}

func (c *Client) LeadershipTransfer(ctx context.Context, target raft.ServerAddress, req *LeadershipTransferToServerRequest) (*AwaitResponse, error) {
	url := c.generateUrl(target, "LeadershipTransfer")
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequest("POST", url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(ctx, r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusAccepted:
		var out AwaitResponse
		err = json.NewDecoder(res.Body).Decode(&out)
		if err != nil {
			return nil, err
		}
		return &out, nil
	case http.StatusTemporaryRedirect:
		return nil, ErrNotLeader
	default:
		return nil, fmt.Errorf("unexpected HTTP status code: %v", res.Status)
	}
}

func (c *Client) RemoveServer(ctx context.Context, target raft.ServerAddress, req *RemoveServerRequest) (*AwaitResponse, error) {
	url := c.generateUrl(target, "RemoveServer")
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequest("POST", url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(ctx, r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusAccepted:
		var out AwaitResponse
		err = json.NewDecoder(res.Body).Decode(&out)
		if err != nil {
			return nil, err
		}
		return &out, nil
	case http.StatusTemporaryRedirect:
		return nil, ErrNotLeader
	default:
		return nil, fmt.Errorf("unexpected HTTP status code: %v", res.Status)
	}
}

func (c *Client) Shutdown(ctx context.Context, target raft.ServerAddress) (*AwaitResponse, error) {
	url := c.generateUrl(target, "Shutdown")
	r, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(ctx, r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusAccepted:
		var out AwaitResponse
		err = json.NewDecoder(res.Body).Decode(&out)
		if err != nil {
			return nil, err
		}
		return &out, nil
	case http.StatusTemporaryRedirect:
		return nil, ErrNotLeader
	default:
		return nil, fmt.Errorf("unexpected HTTP status code: %v", res.Status)
	}
}

func (c *Client) State(ctx context.Context, target raft.ServerAddress) (*StateResponse, error) {
	url := c.generateUrl(target, "State")
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(ctx, r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		var out StateResponse
		err = json.NewDecoder(res.Body).Decode(&out)
		if err != nil {
			return nil, err
		}
		return &out, nil
	case http.StatusTemporaryRedirect:
		return nil, ErrNotLeader
	default:
		return nil, fmt.Errorf("unexpected HTTP status code: %v", res.Status)
	}
}

func (c *Client) Stats(ctx context.Context, target raft.ServerAddress) (*StatsResponse, error) {
	url := c.generateUrl(target, "Stats")
	r, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(ctx, r)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		var out StatsResponse
		err = json.NewDecoder(res.Body).Decode(&out)
		if err != nil {
			return nil, err
		}
		return &out, nil
	case http.StatusTemporaryRedirect:
		return nil, ErrNotLeader
	default:
		return nil, fmt.Errorf("unexpected HTTP status code: %v", res.Status)
	}
}

func (c *Client) VerifyLeader(ctx context.Context, target raft.ServerAddress) error {
	url := c.generateUrl(target, "VerifyLeader")
	r, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	res, err := c.clientRetry(ctx, r)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK, http.StatusAccepted:
		return nil
	case http.StatusTemporaryRedirect:
		return ErrNotLeader
	default:
		return fmt.Errorf("unexpected HTTP status code: %v", res.Status)
	}
}

func (c *Client) clientRetry(ctx context.Context, r *http.Request) (*http.Response, error) {
	retries := 0
	r = r.WithContext(ctx)
RETRY:
	res, err := c.do(r)
	if err != nil {
		if retries > 3 {
			return nil, err
		} else if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		retries++
		goto RETRY
	}
	return res, nil
}

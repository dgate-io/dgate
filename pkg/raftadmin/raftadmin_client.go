package raftadmin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/raft"
	"github.com/rs/zerolog"
)

type Doer func(*http.Request) (*http.Response, error)

type HTTPAdminClient struct {
	do     Doer
	urlFmt string
	logger zerolog.Logger
}

func NewHTTPAdminClient(doer Doer, urlFmt string, logger zerolog.Logger) *HTTPAdminClient {
	if doer == nil {
		doer = http.DefaultClient.Do
	}
	if urlFmt == "" {
		urlFmt = "http://(address)/raftadmin/"
	} else {
		if !strings.Contains(urlFmt, "(address)") {
			panic("urlFmt must contain the string '(address)'")
		}
		if !strings.HasSuffix(urlFmt, "/") {
			urlFmt += "/"
		}
	}
	return &HTTPAdminClient{
		do:     doer,
		urlFmt: urlFmt,
		logger: logger,
	}
}

func (c *HTTPAdminClient) generateUrl(target raft.ServerAddress, action string) string {
	return strings.ReplaceAll(c.urlFmt+action,
		"(address)", string(target))
}

func (c *HTTPAdminClient) AddNonvoter(ctx context.Context, target raft.ServerAddress, req *AddNonvoterRequest) (*AwaitResponse, error) {
	url := c.generateUrl(target, "AddNonvoter")
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequest("POST", url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(r)
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

func (c *HTTPAdminClient) AddVoter(ctx context.Context, target raft.ServerAddress, req *AddVoterRequest) (*AwaitResponse, error) {
	url := c.generateUrl(target, "AddVoter")
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequest("POST", url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(r)
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

func (c *HTTPAdminClient) AppliedIndex(ctx context.Context, target raft.ServerAddress) (*AppliedIndexResponse, error) {
	url := c.generateUrl(target, "AppliedIndex")
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(r)
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

func (c *HTTPAdminClient) ApplyLog(ctx context.Context, target raft.ServerAddress, req *ApplyLogRequest) (*AwaitResponse, error) {
	url := c.generateUrl(target, "ApplyLog")
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequest("POST", url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(r)
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

func (c *HTTPAdminClient) Barrier(ctx context.Context, target raft.ServerAddress) (*AwaitResponse, error) {
	url := c.generateUrl(target, "Barrier")
	r, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(r)
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

func (c *HTTPAdminClient) DemoteVoter(ctx context.Context, target raft.ServerAddress, req *DemoteVoterRequest) (*AwaitResponse, error) {
	url := c.generateUrl(target, "DemoteVoter")
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequest("POST", url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(r)
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

func (c *HTTPAdminClient) GetConfiguration(ctx context.Context, target raft.ServerAddress) (*GetConfigurationResponse, error) {
	url := c.generateUrl(target, "GetConfiguration")
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(r)
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

func (c *HTTPAdminClient) LastContact(ctx context.Context, target raft.ServerAddress) (*LastContactResponse, error) {
	url := c.generateUrl(target, "LastContact")
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(r)
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

func (c *HTTPAdminClient) LastIndex(ctx context.Context, target raft.ServerAddress) (*LastIndexResponse, error) {
	url := c.generateUrl(target, "LastIndex")
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(r)
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

func (c *HTTPAdminClient) Leader(ctx context.Context, target raft.ServerAddress) (*LeaderResponse, error) {
	url := c.generateUrl(target, "Leader")
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(r)
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

func (c *HTTPAdminClient) LeadershipTransfer(ctx context.Context, target raft.ServerAddress, req *LeadershipTransferToServerRequest) (*AwaitResponse, error) {
	url := c.generateUrl(target, "LeadershipTransfer")
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequest("POST", url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(r)
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

func (c *HTTPAdminClient) RemoveServer(ctx context.Context, target raft.ServerAddress, req *RemoveServerRequest) (*AwaitResponse, error) {
	url := c.generateUrl(target, "RemoveServer")
	buf, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequest("POST", url, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(r)
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

func (c *HTTPAdminClient) Shutdown(ctx context.Context, target raft.ServerAddress) (*AwaitResponse, error) {
	url := c.generateUrl(target, "Shutdown")
	r, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(r)
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

func (c *HTTPAdminClient) State(ctx context.Context, target raft.ServerAddress) (*StateResponse, error) {
	url := c.generateUrl(target, "State")
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(r)
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

func (c *HTTPAdminClient) Stats(ctx context.Context, target raft.ServerAddress) (*StatsResponse, error) {
	url := c.generateUrl(target, "Stats")
	r, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.clientRetry(r)
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

func (c *HTTPAdminClient) VerifyLeader(ctx context.Context, target raft.ServerAddress) error {
	url := c.generateUrl(target, "VerifyLeader")
	r, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	res, err := c.clientRetry(r)
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

func (c *HTTPAdminClient) clientRetry(r *http.Request) (*http.Response, error) {
	retries := 0
RETRY:
	res, err := c.do(r)
	if err != nil {
		if retries > 5 {
			return nil, err
		}
		<-time.After(1 * time.Second)
		retries++
		goto RETRY
	}
	return res, nil
}

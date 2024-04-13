package types

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/dgate-io/dgate/pkg/eventloop"
	"github.com/dop251/goja"
)

type RequestWrapper struct {
	req  *http.Request
	loop *eventloop.EventLoop

	Body          io.ReadCloser
	Method        string
	URL           string
	Headers       http.Header
	Query         url.Values
	Host          string
	RemoteAddress string
	Proto         string
	ContentLength int64
}

func NewRequestWrapper(
	req *http.Request,
	loop *eventloop.EventLoop,
) *RequestWrapper {
	ip, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		ip = req.RemoteAddr
	}
	return &RequestWrapper{
		loop:  loop,
		req:   req,
		Query: req.URL.Query(),
		URL:   req.URL.String(),

		Host:          req.Host,
		Proto:         req.Proto,
		Headers:       req.Header,
		Body:          req.Body,
		Method:        req.Method,
		RemoteAddress: ip,
		ContentLength: req.ContentLength,
	}
}

func (g *RequestWrapper) GetBody() (*goja.ArrayBuffer, error) {
	if g.Body == nil {
		return nil, errors.New("body is not set")
	}
	buf, err := io.ReadAll(g.Body)
	if err != nil {
		return nil, err
	}
	defer g.Body.Close()
	arrBuf := g.loop.Runtime().NewArrayBuffer(buf)
	return &arrBuf, nil
}

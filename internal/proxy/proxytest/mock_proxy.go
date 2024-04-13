package proxytest

import (
	"bytes"
	"io"
	"net/http"

	"github.com/stretchr/testify/mock"
)

func NewMockRequestAndResponseWriter(
	method string,
	url string,
	data []byte,
) (*http.Request, *MockResponseWriter) {
	body := io.NopCloser(bytes.NewReader(data))
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		panic(err)
	}
	req.ContentLength = int64(len(data))
	rw := &MockResponseWriter{}
	return req, rw
}

type MockResponseWriter struct {
	mock.Mock

	writeFallthrough bool
}

func (rw *MockResponseWriter) Header() http.Header {
	args := rw.Called()
	return args.Get(0).(http.Header)
}

func (rw *MockResponseWriter) Write(bytes []byte) (int, error) {
	args := rw.Called(bytes)
	if rw.writeFallthrough {
		return len(bytes), nil
	}
	return args.Int(0), args.Error(1)
}

func (rw *MockResponseWriter) WriteHeader(statusCode int) {
	rw.Called(statusCode)
}

func (rw *MockResponseWriter) SetWriteFallThrough() {
	rw.writeFallthrough = true
}

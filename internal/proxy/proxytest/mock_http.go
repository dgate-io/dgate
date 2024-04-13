package proxytest

import (
	"net/http"

	"github.com/stretchr/testify/mock"
)

type S string

type mockTransport struct {
	mock.Mock
	CallCount int
}

var _ http.RoundTripper = (*mockTransport)(nil)
var _ http.ResponseWriter = (*mockResponseWriter)(nil)

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	m.CallCount++
	args := m.Called(req)
	resp := args.Get(0)
	if resp == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), nil
}

type mockResponseWriter struct {
	mock.Mock
}

func (rw *mockResponseWriter) Header() http.Header {
	if firstArg := rw.Called().Get(0); firstArg == nil {
		return nil
	} else {
		return firstArg.(http.Header)
	}
}

func (rw *mockResponseWriter) Write(b []byte) (int, error) {
	args := rw.Called(b)
	return args.Int(0), args.Error(1)
}

func (rw *mockResponseWriter) WriteHeader(i int) {
	rw.Called(i)
}

func CreateMockTransport() *mockTransport {
	return new(mockTransport)
}

func CreateMockResponseWriter() *mockResponseWriter {
	return new(mockResponseWriter)
}

func CreateMockRequest(method string, url string) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	return req
}

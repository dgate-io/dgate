package spec

import (
	"net/http"
)

type ResponseWriterTracker interface {
	http.ResponseWriter
	Status() int
	HeadersSent() bool
	BytesWritten() int64
}

type rwTracker struct {
	rw           http.ResponseWriter
	status       int
	bytesWritten int64
}

var _ ResponseWriterTracker = (*rwTracker)(nil)

func NewResponseWriterTracker(rw http.ResponseWriter) ResponseWriterTracker {
	if rwt, ok := rw.(ResponseWriterTracker); ok {
		return rwt
	}
	return &rwTracker{
		rw: rw,
	}
}

func (t *rwTracker) Header() http.Header {
	return t.rw.Header()
}

func (t *rwTracker) Write(b []byte) (int, error) {
	if !t.HeadersSent() {
		t.WriteHeader(http.StatusOK)
	}
	n, err := t.rw.Write(b)
	t.bytesWritten += int64(n)
	return n, err
}

func (t *rwTracker) WriteHeader(statusCode int) {
	if statusCode == 0 {
		panic("rwTracker.WriteHeader: statusCode cannot be 0")
	}
	t.status = statusCode
	t.rw.WriteHeader(statusCode)
}

func (t *rwTracker) Status() int {
	return t.status
}

func (t *rwTracker) HeadersSent() bool {
	return t.status != 0
}

func (t *rwTracker) BytesWritten() int64 {
	return t.bytesWritten
}

package types_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/dgate-io/dgate/pkg/modules/types"
	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
)

// TODO: test all methods and fields
func TestHttpRequestWrapper(t *testing.T) {
	req, err := http.NewRequest("GET", "http://localhost", bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	reqWrapper := types.NewRequestWrapper(req, nil)
	if reqWrapper == nil {
		t.Fatal("NewGojaRequestWrapper failed")
	}

	rt := goja.New()

	rt.Set("req", reqWrapper)
	v, err := rt.RunString(`req.Method`)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "GET", v.Export())

	v, err = rt.RunString(`req.Body`)
	if err != nil {
		t.Fatal(err)
	}
	assert.NotNil(t, v.Export())
}

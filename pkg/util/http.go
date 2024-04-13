package util

import (
	"fmt"
	"net/http"
)

func WriteStatusCodeError(w http.ResponseWriter, code int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	w.Write([]byte(
		fmt.Sprintf("DGate: %d %s", code, http.StatusText(code)),
	))
}

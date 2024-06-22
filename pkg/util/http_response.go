package util

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"

	"github.com/dgate-io/dgate/internal/extensions/telemetry"
)

func isSlice(v interface{}) bool {
	return reflect.TypeOf(v).Kind() == reflect.Slice
}

func sliceLen(v interface{}) int {
	return reflect.ValueOf(v).Len()
}

func JsonResponse(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if data == nil {
		w.Write([]byte("{}"))
		return
	}
	responseData := map[string]any{
		"status_code": statusCode,
		"data":        data,
	}
	if isSlice(data) {
		responseData["count"] = sliceLen(data)
	}
	value, err := json.Marshal(responseData)
	if err != nil {
		JsonError(w, http.StatusInternalServerError, "Error marshalling response data")
	} else {
		w.Write(value)
	}
}

func JsonError[T any](w http.ResponseWriter, statusCode int, message T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]any{
		"error":  message,
		"status": statusCode,
	})
}

func JsonErrors(w http.ResponseWriter, statusCode int, errs []error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	telemetry.CaptureError(errors.Join(errs...))
	json.NewEncoder(w).Encode(map[string]any{
		"errors": errs,
		"status": statusCode,
	})
}

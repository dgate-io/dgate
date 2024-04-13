package util

import (
	"fmt"

	"github.com/dop251/goja"
)

func ToBytes(a any) ([]byte, error) {
	switch dt := a.(type) {
	case string:
		return []byte(dt), nil
	case []byte:
		return dt, nil
	case goja.ArrayBuffer:
		return dt.Bytes(), nil
	default:
		return nil, fmt.Errorf("invalid type %T, expected (string, []byte or ArrayBuffer)", a)
	}
}

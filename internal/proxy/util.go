package proxy

import (
	"bytes"
	"cmp"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hash"
	"hash/crc64"
	"net/http"
	"slices"
	"sort"
)

func saltHash[T any](salt uint64, objs ...T) (hash.Hash64, error) {
	hash := crc64.New(crc64.MakeTable(crc64.ECMA))
	if salt != 0 {
		// uint32 to byte array
		b := make([]byte, 4)
		b[0] = byte(salt >> 24)
		b[1] = byte(salt >> 16)
		b[2] = byte(salt >> 8)
		b[3] = byte(salt)
		hash.Write(b)
	}

	if len(objs) == 0 {
		return nil, errors.New("no objects provided")
	}
	for _, r := range objs {
		b := bytes.Buffer{}
		err := json.NewEncoder(&b).Encode(r)
		if err != nil {
			return nil, err
		}
		hash.Write(b.Bytes())
	}
	return hash, nil
}

func HashAny[T any](salt uint64, objs ...T) (uint64, error) {
	h, err := saltHash(salt, objs...)
	if err != nil {
		return 0, err
	}
	return h.Sum64(), nil
}

func HashString[T any](salt uint64, objs ...T) (string, error) {
	h, err := saltHash(salt, objs...)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func findInSortedWith[T any, K cmp.Ordered](arr []T, k K, f func(T) K) (T, bool) {
	i := sort.Search(len(arr), func(i int) bool {
		return f(arr[i]) >= k
	})
	var t T
	if i < len(arr) && f(arr[i]) == k {
		return arr[i], true
	}
	return t, false
}

var validMethods = []string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodOptions,
	http.MethodHead,
	http.MethodConnect,
	http.MethodTrace,
}

func ValidateMethods(methods []string) error {
	methodCount := 0
	for _, m := range methods {
		if m == "" {
			continue
		} else if slices.ContainsFunc(validMethods, func(v string) bool {
			return v == m
		}) {
			methodCount++
		} else {
			return errors.New("unsupported method: " + m)
		}
	}
	if methodCount == 0 {
		return errors.New("no valid methods provided")
	}
	return nil
}

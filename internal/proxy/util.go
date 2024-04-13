package proxy

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"hash/crc32"
	"sort"
)

func HashAny[T any](salt uint32, objs ...any) (uint32, error) {
	hash := crc32.NewIEEE()
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
		return 0, errors.New("no objects provided")
	}
	for _, r := range objs {
		b := bytes.Buffer{}
		err := json.NewEncoder(&b).Encode(r)
		if err != nil {
			return 0, err
		}
		hash.Write(b.Bytes())
	}
	return hash.Sum32(), nil
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

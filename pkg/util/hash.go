package util

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"hash"
	"hash/crc32"
)
func jsonHash(objs ...any) (hash.Hash32, error) {
	hash, err := crc32Hash(func(a any) []byte {
		b, err := json.Marshal(a)
		if err != nil {
			return nil
		}
		return b
	}, objs...)
	if err != nil {
		return nil, err
	}
	return hash, nil
}
func JsonHash(objs ...any) (uint32, error) {
	hash, err := jsonHash(objs...)
	if err != nil {
		return 0, err
	}
	return hash.Sum32(), nil
}


func JsonHashBytes(objs ...any) ([]byte, error) {
	hash, err := jsonHash(objs...)
	if err != nil {
		return nil, err
	}
	return hash.Sum(nil), nil
}

func GobHashBytes(objs ...any) (uint32, error) {
	hash, err := crc32Hash(func(a any) []byte {
		b := bytes.Buffer{}
		enc := gob.NewEncoder(&b)
		err := enc.Encode(a)
		if err != nil {
			return nil
		}
		return b.Bytes()
	}, objs...)
	if err != nil {
		return 0, err
	}
	return hash.Sum32(), nil
}

func GobHash(objs ...any) (uint32, error) {
	hash, err := crc32Hash(func(a any) []byte {
		b := bytes.Buffer{}
		enc := gob.NewEncoder(&b)
		err := enc.Encode(a)
		if err != nil {
			return nil
		}
		return b.Bytes()
	}, objs...)
	if err != nil {
		return 0, err
	}
	return hash.Sum32(), nil
}

func crc32Hash(encoder func(any) []byte, objs ...any) (hash.Hash32, error) {
	hash := crc32.NewIEEE()
	if len(objs) == 0 {
		return nil, errors.New("no values provided")
	}
	for _, r := range objs {
		b := bytes.Buffer{}
		_, err := b.Write(encoder(r))
		if err != nil {
			return nil, err
		}
		hash.Write(b.Bytes())
	}
	return hash, nil
}

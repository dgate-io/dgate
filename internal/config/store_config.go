package config

import (
	"github.com/mitchellh/mapstructure"
)

type StorageType string

const (
	StorageTypeMemory StorageType = "memory"
	StorageTypeFile   StorageType = "file"
)

func StoreConfig[T any, C any](config C) (T, error) {
	var output T
	cfg := &mapstructure.DecoderConfig{
		TagName:  "koanf",
		Metadata: nil,
		Result:   &output,
	}
	decoder, err := mapstructure.NewDecoder(cfg)
	if err != nil {
		return output, err
	}
	err = decoder.Decode(config)
	return output, err
}

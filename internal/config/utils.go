package config

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"

	"github.com/knadh/koanf/v2"
	"github.com/mitchellh/mapstructure"
)

type FoundFunc func(string, map[string]string) (string, error)
type ErrorFunc func(map[string]string, error)

func resolveConfigStringPattern(
	data map[string]any,
	re *regexp.Regexp,
	foundFunc FoundFunc,
	errorFunc ErrorFunc,
) {
	for k, v := range data {
		var values []string
		switch vt := v.(type) {
		case string:
			values = []string{vt}
		case []string:
			values = vt
		case []any:
			values = sliceMap(vt, func(val any) string {
				if s, ok := val.(string); ok {
					return s
				}
				return fmt.Sprint(val)
			})
			if len(values) == 0 {
				continue
			}
		case any:
			if vv, ok := vt.(string); ok {
				values = []string{vv}
			} else {
				continue
			}
		default:
			continue
		}
		if len(values) == 0 {
			continue
		}
		hasMatch := false
		for i, value := range values {
			if value == "" {
				continue
			}
			newValue := re.ReplaceAllStringFunc(value, func(s string) string {
				matches := re.FindAllStringSubmatch(s, -1)
				results := make(map[string]string)
				for _, match := range matches {
					for name, match := range mapSlices(re.SubexpNames(), match) {
						if name != "" {
							results[name] = match
							hasMatch = true
						}
					}
				}
				result, err := foundFunc(value, results)
				if err != nil {
					errorFunc(results, err)
				}

				return result
			})

			values[i] = newValue
		}
		if hasMatch {
			if len(values) == 1 {
				data[k] = values[0]
				continue
			}
			data[k] = values
		}
	}
}

func mapSlices[K comparable, V any](ks []K, vs []V) map[K]V {
	if len(ks) != len(vs) {
		panic("length of ks and vs must be equal")
	}
	result := make(map[K]V, len(ks))
	for i, k := range ks {
		result[k] = vs[i]
	}
	return result
}

func sliceMap[T any, V any](s []T, f func(T) V) []V {
	result := make([]V, len(s))
	for i, v := range s {
		result[i] = f(v)
	}
	return result
}

func StringToIntHookFunc() mapstructure.DecodeHookFuncType {
	return func(from, to reflect.Type, data any) (any, error) {
		if from.Kind() != reflect.String {
			return data, nil
		}
		if to.Kind() != reflect.Int {
			return data, nil
		}
		return strconv.Atoi(data.(string))
	}
}

func StringToBoolHookFunc() mapstructure.DecodeHookFuncType {
	return func(f, t reflect.Type, data any) (any, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t.Kind() != reflect.Bool {
			return data, nil
		}
		return strconv.ParseBool(data.(string))
	}
}

func kDefault(k *koanf.Koanf, key string, value any) {
	initialValue := k.Get(key)
	if !k.Exists(key) || initialValue == nil || value == "" {
		k.Set(key, value)
	}
}

func kRequireAll(k *koanf.Koanf, keys ...string) error {
	for _, key := range keys {
		if !k.Exists(key) {
			return errors.New(key + " is required")
		}
	}
	return nil
}

func kRequireAny(k *koanf.Koanf, keys ...string) error {
	for _, key := range keys {
		if k.Exists(key) {
			return nil
		}
	}
	return fmt.Errorf("one of %v is required", keys)
}

func kRequireIfExists(k *koanf.Koanf, dependent string, targets ...string) error {
	for _, target := range targets {
		if k.Exists(dependent) {
			if k.Get(target) == "" {
				return fmt.Errorf("%s is required, if %s is set", target, dependent)
			}
		}
	}
	return nil
}

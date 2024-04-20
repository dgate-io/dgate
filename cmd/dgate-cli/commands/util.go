package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

func createMapFromArgs[N any](
	args []string,
	// tags []string,
	required ...string,
) (*N, error) {
	m := make(map[string]any)
	// parse json strings
	for i, arg := range args {
		pair := strings.SplitN(arg, ":=", 2)
		if len(pair) != 2 || pair[0] == "" || pair[1] == "" {
			continue
		}
		var v any
		err := json.Unmarshal([]byte(pair[1]), &v)
		if err != nil {
			if _, ok := err.(*json.SyntaxError); ok {
				err = fmt.Errorf("error parsing values: invalid json for key '%s'", pair[0])
				return nil, err
			}
			return nil, fmt.Errorf("invalid json value - key:'%s' value:'%s'", pair[0], pair[1])
		}
		m[pair[0]] = v
		args[i] = ""
	}

	// parse raw strings
	for _, arg := range args {
		pair := strings.SplitN(arg, "=", 2)
		if len(pair) != 2 || pair[0] == "" || pair[1] == "" {
			continue
		}
		m[pair[0]] = pair[1]
	}

	var missingKeys []string
	for _, requiredKey := range required {
		if _, ok := m[requiredKey]; !ok {
			missingKeys = append(missingKeys, requiredKey)
		}
	}
	if len(missingKeys) > 0 {
		return nil, errors.New("missing required keys: " +
			strings.Join(required, ", "))
	}

	mapJson, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var resource N
	err = json.Unmarshal(mapJson, &resource)
	if err != nil {
		if ute, ok := err.(*json.UnmarshalTypeError); ok {
			err = fmt.Errorf("error parsing values: field '%s' expected type %s", ute.Field, ute.Type.String())
			if ute.Type.Kind() != reflect.String {
				// TODO: add suggestion to use `:=` instead of `=`
				fmt.Println("*suggestion: try using `:=` instead of `=`")
			}
			return nil, err
		}
		return nil, err
	}
	return &resource, nil
}

func jsonPrettyPrint(item any) error {
	b, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

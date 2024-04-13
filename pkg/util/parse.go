package util

import "strconv"

func ParseInt(s string, def int) (int, error) {
	if s == "" {
		return def, nil
	}
	if i, err := strconv.Atoi(s); err == nil {
		return i, nil
	} else {
		return def, err
	}
}

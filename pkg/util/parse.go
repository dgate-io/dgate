package util

import (
	"strconv"
	"time"
)

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

func ParseBase36Timestamp(s string) (time.Time, error) {
	if i, err := strconv.ParseInt(s, 36, 64); err == nil {
		
		return time.Unix(0, i), nil
	} else {
		return time.Time{}, err
	}
}

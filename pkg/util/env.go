package util

import (
	"os"
	"strings"
)

func EnvVarCheckBool(name string) bool {
	val := strings.ToLower(os.Getenv(name))
	if val == "true" || val == "1" || val == "yes" || val == "y" {
		return true
	}
	return false
}

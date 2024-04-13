package util

func Default[T any](value *T, defaultValue *T) *T {
	if value == nil {
		return defaultValue
	}
	return value
}

func DefaultString(value string, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

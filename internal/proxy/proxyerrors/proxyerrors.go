package proxyerrors

import "errors"

type ProxyError struct {
	DisableRetry bool
	StatusCode   int
	Err          error
}

func NewProxyError(text string) error {
	return &ProxyError{
		Err: errors.New(text),
	}
}

func (e *ProxyError) Error() string {
	return e.Err.Error()
}

func GetProxyError(err error) *ProxyError {
	if err == nil {
		return nil
	}
	pxyErr, _ := err.(*ProxyError)
	return pxyErr
}

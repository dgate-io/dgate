package dgclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type clientDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type NamePayload struct {
	Name string `json:"name"`
}

type NamespacePayload struct {
	Namespace string `json:"namespace"`
}

type DocumentPayload struct {
	Document   string `json:"document"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
	Count      int    `json:"count"`
	Collection string `json:"collection"`
	Namespace  string `json:"namespace"`
}

type ListResponseWrapper[T any] struct {
	StatusCode int
	Count      int
	Data       []T
}

type ResponseWrapper[T any] struct {
	StatusCode int
	Count      int
	Data       T
}

func commonGetList[T any](client clientDoer, uri string) ([]T, error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err = validateStatusCode(resp.StatusCode); err != nil {
		return nil, parseApiError(resp.Body, err)
	}
	var items ListResponseWrapper[T]
	err = json.NewDecoder(resp.Body).Decode(&items)
	if err != nil {
		return nil, err
	}
	return items.Data, nil
}

func commonGet[T any](client clientDoer, uri string) (*T, error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err = validateStatusCode(resp.StatusCode); err != nil {
		return nil, parseApiError(resp.Body, err)
	}
	var item ResponseWrapper[T]
	err = json.NewDecoder(resp.Body).Decode(&item)
	if err != nil {
		return nil, err
	}
	return &item.Data, nil
}

func commonPut[T any](client clientDoer, uri string, item T) error {
	nsJson, err := json.Marshal(item)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", uri, bytes.NewReader(nsJson))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err = validateStatusCode(resp.StatusCode); err != nil {
		return parseApiError(resp.Body, err)
	}
	return nil
}

func basicDelete(client clientDoer, uri string, rdr io.Reader) error {
	req, err := http.NewRequest("DELETE", uri, rdr)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err = validateStatusCode(resp.StatusCode); err != nil {
		return parseApiError(resp.Body, err)
	}
	return nil
}

type M map[string]any

func commonDelete(client clientDoer, uri, name, namespace string) error {
	payload, err := json.Marshal(M{
		"name":      name,
		"namespace": namespace,
	})
	if err != nil {
		return err
	}
	return basicDelete(client, uri, bytes.NewReader(payload))
}

func validateStatusCode(code int) error {
	if code < 300 {
		return nil
	} else if code < 400 {
		return errors.New("redirect from server; retry with the --follow flag")
	}
	return fmt.Errorf("%d error from server", code)
}

func parseApiError(body io.Reader, wrapErr error) error {
	var apiError struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(body).Decode(&apiError); err != nil || apiError.Error == "" {
		return wrapErr
	}
	return fmt.Errorf("%s: %s", wrapErr, apiError.Error)
}

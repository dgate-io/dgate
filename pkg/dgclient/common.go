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

type M map[string]any

func commonDelete(client clientDoer, uri, name, namespace string) error {
	payload, err := json.Marshal(M{
		"name":      name,
		"namespace": namespace,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest("DELETE", uri, bytes.NewReader(payload))
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

func validateStatusCode(code int) error {
	if code < 300 {
		return nil
	} else if code < 400 {
		return errors.New("unexpected Redirect")
	}
	return fmt.Errorf("error code %d: %s", code, http.StatusText(code))
}

func parseApiError(body io.Reader, defaultErr error) error {
	var apiError struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(body).Decode(&apiError); err != nil || apiError.Error == "" {
		return defaultErr
	}
	return errors.New("api error: " + apiError.Error)
}

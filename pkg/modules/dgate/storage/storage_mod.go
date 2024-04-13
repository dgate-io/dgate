package storage

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/dgate-io/dgate/pkg/modules"
	"github.com/dgate-io/dgate/pkg/spec"
)

type StorageModule struct {
	modCtx modules.RuntimeContext
}

var _ modules.GoModule = &StorageModule{}

func New(modCtx modules.RuntimeContext) modules.GoModule {
	return &StorageModule{modCtx}
}

func (sm *StorageModule) Exports() *modules.Exports {
	return &modules.Exports{
		Named: map[string]any{
			"useCache": sm.UseCache,
			"getCache": sm.GetCache,
			"setCache": sm.SetCache,
		},
	}
}

type UpdateFunc func(any, any) any

type CacheOptions struct {
	InitialValue any           `json:"initialValue"`
	Callback     UpdateFunc    `json:"updateCallback"`
	TTL          time.Duration `json:"ttl"`
}

func (sm *StorageModule) UseCache(cacheId string, opts CacheOptions) (arr [2]any, err error) {
	if cacheId == "" {
		err = errors.New("cache id cannot be empty")
		return
	}
	nsValue := sm.modCtx.Context().
		Value(spec.Name("namespace"))
	if nsValue == nil || nsValue.(string) == "" {
		err = errors.New("namespace is not set")
		return
	}
	namespace := nsValue.(string)

	if opts.TTL < 0 {
		err = errors.New("TTL cannot be negative")
		return
	}

	bucket := sm.modCtx.State().SharedCache().
		Bucket("storage:cache:" + namespace)

	val, ok := bucket.Get(cacheId)
	if !ok && opts.InitialValue != nil {
		val = opts.InitialValue
	}

	return [2]any{
		val, func(newVal any) {
			bucket.SetWithTTL(cacheId, newVal, opts.TTL)
			if opts.Callback != nil {
				newVal = opts.Callback(val, newVal)
				// change val to newVal in case
				// this function is called multiple times
				val = newVal
			}
		},
	}, nil
}

func (sm *StorageModule) SetCache(cacheId string, val any, opts CacheOptions) error {
	if cacheId == "" {
		return errors.New("cache id cannot be empty")
	}
	namespace := sm.modCtx.Context().
		Value(spec.Name("namespace"))
	if namespace == nil || namespace.(string) == "" {
		return errors.New("namespace is not set")
	}
	uniqueId := "storage:cache:" + namespace.(string) + ":" + cacheId
	bucket := sm.modCtx.State().SharedCache().Bucket(uniqueId)
	if opts.TTL < 0 {
		return errors.New("TTL cannot be negative")
	}
	bucket.SetWithTTL(uniqueId, val, opts.TTL)
	return nil
}

func (sm *StorageModule) GetCache(cacheId string) (any, error) {
	if cacheId == "" {
		return nil, errors.New("cache id cannot be empty")
	}
	namespace := sm.modCtx.Context().
		Value(spec.Name("namespace"))
	if namespace == nil || namespace.(string) == "" {
		return nil, errors.New("namespace is not set")
	}
	bucket := sm.modCtx.State().SharedCache().
		Bucket("storage:cache:" + namespace.(string))

	if val, ok := bucket.Get(cacheId); ok {
		return val, nil
	}
	return nil, errors.New("cache not found")
}

func (sm *StorageModule) ReadWriteBody(res *http.Response, callback func(string) string) string {
	if callback == nil {
		return ""
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}
	err = res.Body.Close()
	if err != nil {
		panic(err)
	}
	newBody := callback(string(body))
	res.Body = io.NopCloser(bytes.NewReader([]byte(newBody)))
	res.ContentLength = int64(len(newBody))
	res.Header.Set("Content-Length", strconv.Itoa(len(newBody)))
	return string(newBody)
}

func (sm *StorageModule) GetCollection(collectionName string) *spec.Collection {
	if collectionName == "" {
		return nil
	}
	return nil
}

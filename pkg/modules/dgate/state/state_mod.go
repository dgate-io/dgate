package state

import (
	"encoding/json"
	"errors"

	"github.com/dgate-io/dgate/pkg/modules"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/dop251/goja"
)

type ResourcesModule struct {
	modCtx modules.RuntimeContext
}

var _ modules.GoModule = &ResourcesModule{}

func New(modCtx modules.RuntimeContext) modules.GoModule {
	return &ResourcesModule{modCtx}
}

func (hp *ResourcesModule) Exports() *modules.Exports {
	return &modules.Exports{
		Named: map[string]any{
			"getCollection":    hp.fetchCollection,
			"getDocument":      hp.fetchDocument,
			"getDocuments":     hp.fetchDocuments,
			"addCollection":    writeFunc[*spec.Collection](hp, spec.AddCollectionCommand),
			"addDocument":      writeFunc[*spec.Document](hp, spec.AddDocumentCommand),
			"deleteCollection": writeFunc[*spec.Collection](hp, spec.DeleteCollectionCommand),
			"deleteDocument":   writeFunc[*spec.Document](hp, spec.DeleteDocumentCommand),
		},
	}
}

func (hp *ResourcesModule) fetchCollection(name string) *goja.Promise {
	ctx := hp.modCtx.Context()
	state := hp.modCtx.State()
	loop := hp.modCtx.EventLoop()
	rt := hp.modCtx.Runtime()
	rm := state.ResourceManager()
	docPromise, resolve, reject := rt.NewPromise()
	loop.RunOnLoop(func(rt *goja.Runtime) {
		namespace := ctx.Value(spec.Name("namespace"))
		if namespace == nil {
			reject(rt.NewGoError(
				errors.New("namespace not found in context"),
			))
			return
		}
		collection, ok := rm.GetCollection(namespace.(string), name)
		if !ok {
			reject(goja.Null())
			return
		}
		resolve(rt.ToValue(collection))
	})
	return docPromise
}

func (hp *ResourcesModule) fetchDocument(collection, id string) *goja.Promise {
	ctx := hp.modCtx.Context()
	state := hp.modCtx.State()
	loop := hp.modCtx.EventLoop()
	rt := loop.Runtime()
	docPromise, resolve, reject := rt.NewPromise()
	loop.RunOnLoop(func(rt *goja.Runtime) {
		namespace := ctx.Value(spec.Name("namespace"))
		if namespace == nil {
			reject(rt.NewGoError(errors.New("namespace not found in context")))
			return
		}
		doc, err := state.DocumentManager().
			GetDocumentByID(namespace.(string), collection, id)
		if err != nil {
			reject(rt.NewGoError(err))
			return
		}
		resolve(rt.ToValue(doc))
	})
	return docPromise
}

func (hp *ResourcesModule) fetchDocuments(args goja.FunctionCall) (*goja.Promise, error) {
	ctx := hp.modCtx.Context()
	state := hp.modCtx.State()
	loop := hp.modCtx.EventLoop()
	rt := hp.modCtx.Runtime()

	collection_name := ""
	if args.Argument(0) == goja.Undefined() {
		return nil, errors.New("collection name is required")
	} else {
		collection_name = args.Argument(0).String()
	}
	limit := 0
	if args.Argument(1) != goja.Undefined() {
		limit = int(args.Argument(1).ToInteger())
	}
	offset := 0
	if args.Argument(2) != goja.Undefined() {
		offset = int(args.Argument(2).ToInteger())
	}

	namespaceVal := ctx.Value(spec.Name("namespace"))
	if namespaceVal == nil {
		return nil, errors.New("namespace not found in context")
	}
	namespace := namespaceVal.(string)

	docPromise, resolve, reject := rt.NewPromise()
	loop.RunOnLoop(func(rt *goja.Runtime) {
		doc, err := state.DocumentManager().
			GetDocuments(namespace, collection_name, limit, offset)
		if err != nil {
			reject(rt.NewGoError(err))
			return
		}
		resolve(rt.ToValue(doc))
	})
	return docPromise, nil
}

func writeFunc[T spec.Named](hp *ResourcesModule, cmd spec.Command) func(map[string]any) (*goja.Promise, error) {
	return func(item map[string]any) (*goja.Promise, error) {
		if item == nil {
			return nil, errors.New("item is nil")
		}
		ctx := hp.modCtx.Context()
		state := hp.modCtx.State()
		loop := hp.modCtx.EventLoop()
		rt := hp.modCtx.Runtime()
		docPromise, resolve, reject := rt.NewPromise()
		loop.RunOnLoop(func(rt *goja.Runtime) {
			rs, err := remarshalNamed[T](item)
			if err != nil {
				reject(rt.NewGoError(err))
				return
			}

			namespaceVal := ctx.Value(spec.Name("namespace"))
			if namespaceVal == nil {
				reject(rt.NewGoError(errors.New("namespace not found in context")))
				return
			}
			namespace := namespaceVal.(string)

			err = state.ApplyChangeLog(spec.NewChangeLog(rs, namespace, cmd))
			if err != nil {
				reject(rt.NewGoError(err))
				return
			}
			resolve(rt.ToValue(rs))
		})
		return docPromise, nil
	}
}

func remarshalNamed[T spec.Named](obj map[string]any) (T, error) {
	var str T
	objBytes, err := json.Marshal(obj)
	if err != nil {
		return str, err
	}
	err = json.Unmarshal(objBytes, &str)
	return str, err
}

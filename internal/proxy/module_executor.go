package proxy

import (
	"context"
	"errors"
)

type ModuleBuffer interface {
	// Load(cb func())
	Borrow() (ModuleExtractor, bool)
	Return(me ModuleExtractor)
	Close()
}

type moduleBuffer struct {
	modExtBuffer chan ModuleExtractor
	min, max     int

	ctxCancel context.CancelFunc
	ctx       context.Context

	createModuleExtract func() ModuleExtractor
}

func NewModuleBuffer(
	minBuffers, maxBuffers int,
	reqCtxProvider *RequestContextProvider,
	createModExts func(*RequestContextProvider) ModuleExtractor,
) (ModuleBuffer, error) {
	if minBuffers < 1 {
		panic("module concurrency must be greater than 0")
	}
	if maxBuffers < minBuffers {
		panic("maxBuffers must be greater than minBuffers")
	}

	me := createModExts(reqCtxProvider)
	if me == nil {
		return nil, errors.New("could not load moduleExtract")
	}
	mb := &moduleBuffer{
		min:          minBuffers,
		max:          maxBuffers,
		modExtBuffer: make(chan ModuleExtractor, maxBuffers),
	}
	mb.createModuleExtract = func() ModuleExtractor {
		return createModExts(reqCtxProvider)
	}
	mb.ctx, mb.ctxCancel = context.WithCancel(reqCtxProvider.ctx)
	return mb, nil
}

// func (mb *moduleBuffer) Load(cb func()) {
// 	go func() {
// 		for i := 0; i < mb.min; i++ {
// 			me := mb.createModuleExtract()
// 			if me == nil {
// 				panic("could not load moduleExtract")
// 			}
// 			mb.modExtBuffer <- me
// 		}
// 		if cb != nil {
// 			cb()
// 		}
// 	}()
// }

func (mb *moduleBuffer) Borrow() (ModuleExtractor, bool) {
	if mb == nil || mb.ctx == nil || mb.ctx.Err() != nil {
		return nil, false
	}
	var me ModuleExtractor
	select {
	case me = <-mb.modExtBuffer:
		break
	// NOTE: important for performance
	default:
		me = mb.createModuleExtract()
	}
	return me, true
}

func (mb *moduleBuffer) Return(me ModuleExtractor) {
	defer me.SetModuleContext(nil)
	// if context is canceled, do not return module extract
	if mb.ctx != nil && mb.ctx.Err() == nil {
		select {
		case mb.modExtBuffer <- me:
			return
		default:
			// if buffer is full, discard module extract
		}
	}
	me.Stop(true)
}

func (mb *moduleBuffer) Close() {
	if mb.ctxCancel != nil {
		mb.ctxCancel()
	}
	close(mb.modExtBuffer)
}

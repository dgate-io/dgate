package proxy

import (
	"context"
)

type ModulePool interface {
	Borrow() ModuleExtractor
	Return(me ModuleExtractor)
	Close()
}

type modulePool struct {
	modExtBuffer chan ModuleExtractor
	min, max     int

	ctxCancel context.CancelFunc
	ctx       context.Context

	createModuleExtract func() (ModuleExtractor, error)
}

func NewModulePool(
	minBuffers, maxBuffers int,
	reqCtxProvider *RequestContextProvider,
	createModExts ModuleExtractorFunc,
) (ModulePool, error) {
	if minBuffers < 1 {
		panic("module concurrency must be greater than 0")
	}
	if maxBuffers < minBuffers {
		panic("maxBuffers must be greater than minBuffers")
	}

	if _, err := createModExts(reqCtxProvider); err != nil {
		return nil, err
	}
	mb := &modulePool{
		min:          minBuffers,
		max:          maxBuffers,
		modExtBuffer: make(chan ModuleExtractor, maxBuffers),
	}
	mb.createModuleExtract = func() (ModuleExtractor, error) {
		return createModExts(reqCtxProvider)
	}
	mb.ctx, mb.ctxCancel = context.WithCancel(reqCtxProvider.ctx)
	return mb, nil
}

func (mb *modulePool) Borrow() ModuleExtractor {
	if mb == nil || mb.ctx == nil || mb.ctx.Err() != nil {
		return nil
	}
	var (
		me  ModuleExtractor
		err error
	)
	select {
	case me = <-mb.modExtBuffer:
		break
	// NOTE: important for performance
	default:
		me, err = mb.createModuleExtract()
		if err != nil {
			return nil
		}
	}
	return me
}

func (mb *modulePool) Return(me ModuleExtractor) {
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

func (mb *modulePool) Close() {
	if mb.ctxCancel != nil {
		mb.ctxCancel()
	}
	close(mb.modExtBuffer)
}

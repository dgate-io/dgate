package proxy

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type ModulePool interface {
	Borrow() ModuleExtractor
	Return(me ModuleExtractor)
	Close()
}

type modulePool struct {
	modExtChan chan ModuleExtractor
	min, max   int
	cancel  context.CancelFunc
	ctx        context.Context

	createModExt func() (ModuleExtractor, error)
}

func NewModulePool(
	minBuffers, maxBuffers int,
	bufferTimeout time.Duration,
	reqCtxProvider *RequestContextProvider,
	createModExts ModuleExtractorFunc,
) (ModulePool, error) {
	if maxBuffers < minBuffers {
		panic("maxBuffers must be greater than minBuffers")
	}
	if _, err := createModExts(reqCtxProvider); err != nil {
		return nil, err
	}
	modExtChan := make(chan ModuleExtractor, maxBuffers)
	mb := &modulePool{
		min:        minBuffers,
		max:        maxBuffers,
		modExtChan: modExtChan,
		createModExt: func() (ModuleExtractor, error) {
			return createModExts(reqCtxProvider)
		},
	}
	mb.ctx, mb.cancel = context.WithCancel(reqCtxProvider.ctx)

	// add min module extractors to the pool
	defer func() {
		for i := 0; i < minBuffers; i++ {
			me, err := mb.createModExt()
			if err == nil {
				mb.modExtChan <- me
			}
		}
	}()

	return mb, nil
}

func (mb *modulePool) Borrow() ModuleExtractor {
	if mb == nil || mb.ctx.Err() != nil {
		zap.L().Warn("stale use of module pool",
			zap.Any("modPool", mb),
		)
		return nil
	}
	var (
		me  ModuleExtractor
		err error
	)
	select {
	case me = <-mb.modExtChan:
		break
	default:
		if me, err = mb.createModExt(); err != nil {
			return nil
		}
	}
	return me
}

func (mb *modulePool) Return(me ModuleExtractor) {
	// if context is canceled, do not return module extract
	if mb.ctx != nil && mb.ctx.Err() == nil {
		select {
		case mb.modExtChan <- me:
			return
		default:
			// if buffer is full, discard module extract
		}
	}
	me.Stop(false)
}

func (mb *modulePool) Close() {
	if mb.cancel != nil {
		mb.cancel()
	}
	close(mb.modExtChan)
}

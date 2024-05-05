package proxy

import (
	"github.com/dgate-io/dgate/pkg/modules"
	"github.com/dgate-io/dgate/pkg/modules/extractors"
	"github.com/dgate-io/dgate/pkg/modules/types"
)

type ModuleExtractor interface {
	// Start starts the event loop for the module extractor; ret is true if the event loop was started, false otherwise
	Start()
	// Stop stops the event loop for the module extractor
	Stop(wait bool)
	// RuntimeContext returns the runtime context for the module extractor
	RuntimeContext() modules.RuntimeContext
	// SetModuleContext sets the module context for the module extractor
	SetModuleContext(*types.ModuleContext)
	// ModuleContext returns the module context for the module extractor
	ModuleContext() *types.ModuleContext
	// ModHash returns the hash of the module
	ModHash() uint32

	FetchUpstreamUrlFunc() (extractors.FetchUpstreamUrlFunc, bool)
	RequestModifierFunc() (extractors.RequestModifierFunc, bool)
	ResponseModifierFunc() (extractors.ResponseModifierFunc, bool)
	ErrorHandlerFunc() (extractors.ErrorHandlerFunc, bool)
	RequestHandlerFunc() (extractors.RequestHandlerFunc, bool)
}

type moduleExtract struct {
	runtimeContext   modules.RuntimeContext
	moduleContext    *types.ModuleContext
	fetchUpstreamUrl extractors.FetchUpstreamUrlFunc
	requestModifier  extractors.RequestModifierFunc
	responseModifier extractors.ResponseModifierFunc
	errorHandler     extractors.ErrorHandlerFunc
	requestHandler   extractors.RequestHandlerFunc
	numUses          int
	modHash          uint32
}

func NewModuleExtractor(
	runtimeContext modules.RuntimeContext,
	fetchUpstreamUrl extractors.FetchUpstreamUrlFunc,
	requestModifier extractors.RequestModifierFunc,
	responseModifier extractors.ResponseModifierFunc,
	errorHandler extractors.ErrorHandlerFunc,
	requestHandler extractors.RequestHandlerFunc,
) ModuleExtractor {
	return &moduleExtract{
		runtimeContext:   runtimeContext,
		fetchUpstreamUrl: fetchUpstreamUrl,
		requestModifier:  requestModifier,
		responseModifier: responseModifier,
		errorHandler:     errorHandler,
		requestHandler:   requestHandler,
	}
}

func (me *moduleExtract) Start() {
	me.runtimeContext.EventLoop().Start()
	me.numUses++
}

// Stop stops the event loop for the module extractor
func (me *moduleExtract) Stop(wait bool) {
	me.moduleContext = nil
	if wait {
		me.runtimeContext.EventLoop().Stop()
	} else {
		me.runtimeContext.EventLoop().StopNoWait()
	}
}

func (me *moduleExtract) SetModuleContext(moduleContext *types.ModuleContext) {
	me.moduleContext = moduleContext
}

func (me *moduleExtract) RuntimeContext() modules.RuntimeContext {
	return me.runtimeContext
}

func (me *moduleExtract) ModuleContext() *types.ModuleContext {
	return me.moduleContext
}

func (me *moduleExtract) ModHash() uint32 {
	return me.modHash
}

func (me *moduleExtract) FetchUpstreamUrlFunc() (extractors.FetchUpstreamUrlFunc, bool) {
	return me.fetchUpstreamUrl, me.fetchUpstreamUrl != nil
}

func (me *moduleExtract) RequestModifierFunc() (extractors.RequestModifierFunc, bool) {
	return me.requestModifier, me.requestModifier != nil
}

func (me *moduleExtract) ResponseModifierFunc() (extractors.ResponseModifierFunc, bool) {
	return me.responseModifier, me.responseModifier != nil
}

func (me *moduleExtract) ErrorHandlerFunc() (extractors.ErrorHandlerFunc, bool) {
	return me.errorHandler, me.errorHandler != nil
}

func (me *moduleExtract) RequestHandlerFunc() (extractors.RequestHandlerFunc, bool) {
	return me.requestHandler, me.requestHandler != nil
}

func NewEmptyModuleExtractor() ModuleExtractor {
	return &moduleExtract{}
}

type ModuleExtractorFunc func(*RequestContextProvider) ModuleExtractor

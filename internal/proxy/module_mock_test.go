package proxy_test

import (
	"net/http"

	"github.com/dgate-io/dgate/internal/proxy"
	"github.com/dgate-io/dgate/pkg/modules"
	"github.com/dgate-io/dgate/pkg/modules/extractors"
	"github.com/dgate-io/dgate/pkg/modules/types"
	"github.com/dgate-io/dgate/pkg/spec"
	"github.com/stretchr/testify/mock"
)

type mockModuleBuffer struct {
	mock.Mock
}

var _ proxy.ModuleBuffer = &mockModuleBuffer{}

func NewMockModuleBuffer() *mockModuleBuffer {
	return &mockModuleBuffer{}
}

// Borrow implements proxy.ModuleBuffer.
func (mb *mockModuleBuffer) Borrow() (proxy.ModuleExtractor, bool) {
	args := mb.Called()
	return args.Get(0).(proxy.ModuleExtractor), args.Bool(1)
}

// Close implements proxy.ModuleBuffer.
func (mb *mockModuleBuffer) Close() {
	mb.Called()
}

// Load implements proxy.ModuleBuffer.
func (mb *mockModuleBuffer) Load(cb func()) {
	mb.Called(cb)
}

// Return implements proxy.ModuleBuffer.
func (mb *mockModuleBuffer) Return(me proxy.ModuleExtractor) {
	mb.Called(me)
}

type mockModuleExtractor struct {
	mock.Mock
}

var _ proxy.ModuleExtractor = &mockModuleExtractor{}

func NewMockModuleExtractor() *mockModuleExtractor {
	return &mockModuleExtractor{}
}

func (m *mockModuleExtractor) ConfigureEmptyMock() {
	rtCtx := proxy.NewRuntimeContext(nil, nil)
	m.On("Start").Return()
	m.On("Stop", mock.Anything).Return()
	m.On("RuntimeContext").Return(rtCtx)
	m.On("ModuleContext").Return(nil)
	m.On("ModHash").Return(uint32(1))
	m.On("SetModuleContext", mock.Anything).Return()
	m.On("FetchUpstreamUrlFunc").Return(nil, false)
	m.On("RequestModifierFunc").Return(nil, false)
	m.On("ResponseModifierFunc").Return(nil, false)
	m.On("ErrorHandlerFunc").Return(nil, false)
	m.On("RequestHandlerFunc").Return(nil, false)
}

func (m *mockModuleExtractor) ConfigureDefaultMock(
	req *http.Request,
	rw http.ResponseWriter,
	ps *proxy.ProxyState,
	rt *spec.DGateRoute,
	mods ...*spec.DGateModule,
) {
	rtCtx := proxy.NewRuntimeContext(ps, rt, mods...)
	modCtx := types.NewModuleContext(nil, rw, req, rt, nil)
	m.On("Start").Return().Maybe()
	m.On("Stop", mock.Anything).Return().Maybe()
	m.On("RuntimeContext").Return(rtCtx).Maybe()
	m.On("ModuleContext").Return(modCtx).Maybe()
	m.On("ModHash").Return(uint32(123)).Maybe()
	m.On("SetModuleContext", mock.Anything).Return().Maybe()
	m.On("FetchUpstreamUrlFunc").Return(extractors.DefaultFetchUpstreamFunction(), true).Maybe()
	m.On("ErrorHandlerFunc").Return(extractors.DefaultErrorHandlerFunction(), true).Maybe()
	m.On("RequestModifierFunc").Return(nil, false).Maybe()
	m.On("ResponseModifierFunc").Return(nil, false).Maybe()
	m.On("RequestHandlerFunc").Return(nil, false).Maybe()
}

func (m *mockModuleExtractor) Start() {
	m.Called()
}

func (m *mockModuleExtractor) Stop(wait bool) {
	m.Called(wait)
}

func (m *mockModuleExtractor) RuntimeContext() modules.RuntimeContext {
	args := m.Called()
	return args.Get(0).(modules.RuntimeContext)
}

func (m *mockModuleExtractor) ModuleContext() *types.ModuleContext {
	args := m.Called()
	return args.Get(0).(*types.ModuleContext)
}

func (m *mockModuleExtractor) ModHash() uint32 {
	args := m.Called()
	return args.Get(0).(uint32)
}

func (m *mockModuleExtractor) SetModuleContext(modCtx *types.ModuleContext) {
	m.Called(modCtx)
}

func (m *mockModuleExtractor) FetchUpstreamUrlFunc() (extractors.FetchUpstreamUrlFunc, bool) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(extractors.FetchUpstreamUrlFunc), args.Bool(1)
}

func (m *mockModuleExtractor) RequestModifierFunc() (extractors.RequestModifierFunc, bool) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(extractors.RequestModifierFunc), args.Bool(1)
}

func (m *mockModuleExtractor) ResponseModifierFunc() (extractors.ResponseModifierFunc, bool) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(extractors.ResponseModifierFunc), args.Bool(1)
}

func (m *mockModuleExtractor) ErrorHandlerFunc() (extractors.ErrorHandlerFunc, bool) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(extractors.ErrorHandlerFunc), args.Bool(1)
}

func (m *mockModuleExtractor) RequestHandlerFunc() (extractors.RequestHandlerFunc, bool) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0).(extractors.RequestHandlerFunc), args.Bool(1)
}

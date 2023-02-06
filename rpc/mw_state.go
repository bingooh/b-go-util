package rpc

import (
	"context"
	"github.com/bingooh/b-go-util/util"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"google.golang.org/grpc"
)

type ctxKeyState struct{}

func StateFromContext(ctx context.Context) State {
	v, _ := ctx.Value(ctxKeyState{}).(State)
	return v
}

func StateIntoContext(ctx context.Context, state State) context.Context {
	return context.WithValue(ctx, ctxKeyState{}, state)
}

type State interface {
	Put(key, val interface{})
	Del(key interface{})
	Clear()
	ToMap() map[interface{}]interface{}
	Has(key interface{}) bool
	Get(key interface{}) (interface{}, bool)
	MustInt(key interface{}) int
	MustInt64(key interface{}) int64
	MustString(key interface{}) string
}

type BaseState struct {
	m map[interface{}]interface{}
}

func NewState() State {
	return NewStateOf(nil)
}

func NewStateOf(vals map[interface{}]interface{}) State {
	m := make(map[interface{}]interface{}, len(vals))
	for k, v := range vals {
		m[k] = v
	}

	return &BaseState{m: m}
}

func (s *BaseState) Put(key, val interface{}) {
	s.m[key] = val
}

func (s *BaseState) Del(key interface{}) {
	if _, ok := s.m[key]; ok {
		delete(s.m, key)
	}
}

func (s *BaseState) Clear() {
	s.m = make(map[interface{}]interface{})
}

func (s *BaseState) ToMap() map[interface{}]interface{} {
	m := make(map[interface{}]interface{}, len(s.m))
	for k, v := range s.m {
		m[k] = v
	}

	return m
}

func (s *BaseState) Has(key interface{}) bool {
	_, ok := s.m[key]
	return ok
}

func (s *BaseState) Get(key interface{}) (interface{}, bool) {
	v, ok := s.m[key]
	return v, ok
}

func (s *BaseState) panicWithCastErr(kind string, k, v interface{}) {
	panic(util.NewAssertFailError(`State值不能转换为%v[key=%v,value=%v(%T)]`, kind, k, v, v))
}

func (s *BaseState) MustInt(key interface{}) int {
	if v, ok := s.m[key].(int); ok {
		return v
	}

	s.panicWithCastErr(`int`, key, s.m[key])
	return 0
}

func (s *BaseState) MustInt64(key interface{}) int64 {
	if v, ok := s.m[key].(int64); ok {
		return v
	}

	s.panicWithCastErr(`int64`, key, s.m[key])
	return 0
}

func (s *BaseState) MustString(key interface{}) string {
	if v, ok := s.m[key].(string); ok {
		return v
	}

	s.panicWithCastErr(`string`, key, s.m[key])
	return ``
}

// MWState State中间件，用于传递上下文状态值
type MWState struct {
	onRequest MWStateOnRequest
	state     State
}

// MWStateOnRequest 拦截器回调函数，可更新State值
type MWStateOnRequest func(ctx context.Context, state State, method string) error

func NewMWState(fn MWStateOnRequest) *MWState {
	return NewMWStateOf(NewState(), fn)
}

func NewMWStateOf(state State, fn MWStateOnRequest) *MWState {
	util.AssertOk(state != nil, `state为空`)
	util.AssertOk(fn != nil, `fn为空`)
	return &MWState{state: state, onRequest: fn}
}

func (s *MWState) handle(ctx context.Context, method string) (context.Context, error) {
	state := StateFromContext(ctx)
	if state == nil {
		state = s.state
		ctx = StateIntoContext(ctx, state)
	}

	return ctx, s.onRequest(ctx, state, method)
}

func (s *MWState) NewUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
		if ctx, err = s.handle(ctx, method); err != nil {
			return
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func (s *MWState) NewStreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (cs grpc.ClientStream, err error) {
		if ctx, err = s.handle(ctx, method); err != nil {
			return
		}
		return streamer(ctx, desc, cc, method, opts...)
	}
}

func (s *MWState) NewUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if ctx, err = s.handle(ctx, info.FullMethod); err != nil {
			return
		}
		return handler(ctx, req)
	}
}

func (s *MWState) NewStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		ctx, err := s.handle(ss.Context(), info.FullMethod)
		if err != nil {
			return err
		}

		//无法设置请求的context，只能转换为string然后设置到metadata
		if ctx != ss.Context() {
			ss = &grpc_middleware.WrappedServerStream{
				ServerStream:   ss,
				WrappedContext: ctx, //ctx已包含auth value
			}
		}

		return handler(srv, ss)
	}
}

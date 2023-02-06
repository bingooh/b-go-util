package rpc

import (
	"context"
	"google.golang.org/grpc"
)

// MWError 错误处理中间件，用于转换接口返回的错误对象
type MWError struct {
	onHandle MWErrorOnHandle
}

// MWErrorOnHandle 拦截器回调函数，用于转换接口返回的错误
// 接口返回的错误对象，默认将转换为：
// - 服务端拦截器：转换为status.Error
// - 客户端拦截器：转换为util.BizError
type MWErrorOnHandle func(ctx context.Context, method string, isFromServer bool, cause error) error

func defaultMWErrorOnHandle(ctx context.Context, method string, isFromServer bool, cause error) error {
	if isFromServer {
		return ToRpcErr(cause)
	}

	return ToBizErr(cause)
}

func NewMWError() *MWError {
	return NewMWErrorWithHandler(nil)
}

func NewMWErrorWithHandler(fn MWErrorOnHandle) *MWError {
	if fn == nil {
		fn = defaultMWErrorOnHandle
	}

	return &MWError{onHandle: fn}
}

func (s *MWError) handle(ctx context.Context, method string, isFromServer bool, cause error) error {
	return s.onHandle(ctx, method, isFromServer, cause)
}

func (s *MWError) NewUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
		if err = invoker(ctx, method, req, reply, cc, opts...); err != nil {
			err = s.handle(ctx, method, false, err)
		}
		return
	}
}

func (s *MWError) NewStreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (cs grpc.ClientStream, err error) {
		//todo 如果服务端接口返回err，streamer()仍然返回nil error。直到cs读取第1个消息时可以获取服务端返回的err
		//如果要获取服务端返回的err，可考虑封装返回的cs，但其相关接口要求返回的err为status.Error
		if cs, err = streamer(ctx, desc, cc, method, opts...); err != nil {
			err = s.handle(ctx, method, false, err)
		}

		return
	}
}

func (s *MWError) NewUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if resp, err = handler(ctx, req); err != nil {
			err = s.handle(ctx, info.FullMethod, true, err)
		}
		return
	}
}

func (s *MWError) NewStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		if err = handler(srv, ss); err != nil {
			err = s.handle(ss.Context(), info.FullMethod, true, err)
		}
		return
	}
}

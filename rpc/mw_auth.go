package rpc

import (
	"context"
	"fmt"
	"github.com/bingooh/b-go-util/_string"
	"github.com/bingooh/b-go-util/util"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"strings"
	"time"
)

type ctxKeyAuthValue struct{}

func AuthValueFromContext(ctx context.Context) interface{} {
	return ctx.Value(ctxKeyAuthValue{})
}

func AuthValueIntoContext(ctx context.Context, val interface{}) context.Context {
	return context.WithValue(ctx, ctxKeyAuthValue{}, val)
}

type AuthHandler interface {
	// OnNewToken 创建token回调函数，客户端拦截器调用
	OnNewToken(ctx context.Context, method string) (string, error)

	// OnCheckToken 校验token回调函数，服务端拦截器调用
	// 校验失败应返回错误，如果返回值val不为nil，则将存放到context，以便后续函数获取
	OnCheckToken(ctx context.Context, token, method string) (val interface{}, err error)
}

// MWAuthOnNewToken 创建token回调函数，客户端拦截器调用
type MWAuthOnNewToken func(ctx context.Context, method string) (string, error)

// MWAuthOnCheckToken 校验token回调函数，服务端拦截器调用
// 校验失败应返回错误，如果返回值val不为nil，则将存放到context，以便后续函数获取
type MWAuthOnCheckToken func(ctx context.Context, token, method string) (val interface{}, err error)

// MWAuthenticator 身份验证器中间件
// token值保存在请求头：authorization: Bearer {token}
type MWAuthenticator struct {
	logger   *zap.Logger
	disabled bool
	handler  AuthHandler
}

func NewMWAuthenticator(handler AuthHandler) *MWAuthenticator {
	util.AssertOk(handler != nil, `handler为空`)
	return &MWAuthenticator{logger: newLogger(`mw.auth`), handler: handler}
}

func (s *MWAuthenticator) Disabled(v bool) *MWAuthenticator {
	s.disabled = v
	return s
}

func (s *MWAuthenticator) appendTokenToOutgoingContext(ctx context.Context, method string) (context.Context, error) {
	if s.disabled {
		return ctx, nil
	}

	token, err := s.handler.OnNewToken(ctx, method)
	if err != nil {
		return nil, status.Errorf(codes.Internal, `create token err->%v`, err)
	}

	val := fmt.Sprintf(`Bearer %v`, token)
	return metadata.AppendToOutgoingContext(ctx, `authorization`, val), nil //metadata会将http header名称转为小写
}

// NewUnaryClientInterceptor 客户端拦截器，添加token
//
// 也可考虑使用grpc.WithPerRPCCredentials()，实现credentials.PerRPCCredentials接口
// 接口方法将在客户端每次发送请求时被调用以获取新的token，具体可参考官方示例
func (s *MWAuthenticator) NewUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
		if ctx, err = s.appendTokenToOutgoingContext(ctx, method); err != nil {
			return err
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func (s *MWAuthenticator) NewStreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (cs grpc.ClientStream, err error) {
		if ctx, err = s.appendTokenToOutgoingContext(ctx, method); err != nil {
			return nil, err
		}

		return streamer(ctx, desc, cc, method, opts...)
	}
}

func (s *MWAuthenticator) checkTokenFromIncomingContext(ctx context.Context, method string) (context.Context, error) {
	if s.disabled || strings.HasPrefix(method, `/grpc.health.`) {
		return ctx, nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, `metadata is empty`)
	}

	val, ok := GetMDVal(md, `authorization`, -1) //获取最后1个值
	if !ok || _string.Empty(val) {
		return nil, status.Error(codes.Unauthenticated, `token is empty`)
	}

	token := strings.TrimPrefix(val, "Bearer ")
	v, err := s.handler.OnCheckToken(ctx, token, method)
	if err != nil {
		if status.Code(err) != codes.Unknown {
			return nil, err
		}

		return nil, status.Errorf(codes.Unauthenticated, `invalid token[%v]->%v`, token, err)
	}

	if v != nil {
		ctx = AuthValueIntoContext(ctx, v)
	}

	return ctx, nil
}

// NewUnaryServerInterceptor 服务端拦截器验证客户端身份
func (s *MWAuthenticator) NewUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if ctx, err = s.checkTokenFromIncomingContext(ctx, info.FullMethod); err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

func (s *MWAuthenticator) NewStreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		ctx := ss.Context()
		if ctx, err = s.checkTokenFromIncomingContext(ctx, info.FullMethod); err != nil {
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

type SimpleAuthOption struct {
	Key                   string   //密码明文
	Disabled              bool     //是否禁用
	TokenHeaders          []string //token头
	EnableUseKeyAsToken   bool     //是否直接使用key作为token
	IgnoreAuthMethods     []string //不需身份校验的方法名称前缀
	EnablePutTokenHeaders bool     //是否在校验成功后保存token头到context里(作为auth value)
}

// SimpleAuthHandler 简单身份验证器，算法使用 util.Token
type SimpleAuthHandler struct {
	option       *SimpleAuthOption
	onNewToken   func(ctx context.Context, key, method string) (string, error)
	onCheckToken func(ctx context.Context, key, token, method string) (val interface{}, err error)
}

func NewSimpleAuthHandler(option *SimpleAuthOption) *SimpleAuthHandler {
	util.AssertOk(option != nil, `option为空`)
	return &SimpleAuthHandler{option: option}
}

func (s *SimpleAuthHandler) WithOnNewToken(fn func(ctx context.Context, key, method string) (string, error)) *SimpleAuthHandler {
	s.onNewToken = fn
	return s
}

func (s *SimpleAuthHandler) WithOnCheckToken(fn func(ctx context.Context, key, token, method string) (val interface{}, err error)) *SimpleAuthHandler {
	s.onCheckToken = fn
	return s
}

func (s *SimpleAuthHandler) MW() *MWAuthenticator {
	return NewMWAuthenticator(s).Disabled(s.option.Disabled)
}

func (s *SimpleAuthHandler) ignore(method string) bool {
	if s.option.Disabled {
		return true
	}

	for _, m := range s.option.IgnoreAuthMethods {
		if strings.HasPrefix(method, m) {
			return true
		}
	}

	return false
}

func (s *SimpleAuthHandler) OnNewToken(ctx context.Context, method string) (token string, err error) {
	if s.ignore(method) {
		return
	}

	if s.onNewToken != nil {
		return s.onNewToken(ctx, s.option.Key, method)
	}

	if s.option.EnableUseKeyAsToken {
		return s.option.Key, nil
	}

	return util.NewToken(s.option.TokenHeaders...).Encode(s.option.Key)
}

func (s *SimpleAuthHandler) OnCheckToken(ctx context.Context, token, method string) (val interface{}, err error) {
	if s.ignore(method) {
		return
	}

	if s.onCheckToken != nil {
		return s.onCheckToken(ctx, s.option.Key, token, method)
	}

	if s.option.EnableUseKeyAsToken {
		if s.option.Key == token {
			return
		}

		return nil, fmt.Errorf(`invalid token[%v]`, token)
	}

	tk, err := util.ParseAndCheckToken(s.option.Key, token, 2*time.Minute)
	if err == nil && s.option.EnablePutTokenHeaders {
		return tk.Headers, nil
	}

	return nil, err
}

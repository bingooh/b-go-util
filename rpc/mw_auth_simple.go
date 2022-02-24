package rpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/bingooh/b-go-util/_string"
	"github.com/bingooh/b-go-util/util"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"strings"
	"sync/atomic"
	"time"
)

//简单身份验证
//token加密算法见util.NewToken()
//token值保存在请求头：authorization: Bearer {token}
type SimpleAuthenticator struct {
	logger *zap.Logger

	keyHolder           *atomic.Value //密码明文
	disabled            bool          //是否禁用
	enableUseKeyAsToken bool          //是否直接使用key作为token,默认false
}

func MustNewSimpleAuthenticator(key string) *SimpleAuthenticator {
	util.AssertOk(!_string.Empty(key), `key is empty`)

	return &SimpleAuthenticator{
		logger:              newLogger(`simple-auth-mw`),
		keyHolder:           util.NewAtomicValue(key),
		enableUseKeyAsToken: false,
	}
}

//是否禁用，如禁用则不检验身份
func (s *SimpleAuthenticator) Disabled(disable bool) *SimpleAuthenticator {
	s.disabled = disable
	return s
}

//是否直接使用key作为token
func (s *SimpleAuthenticator) EnableUseKeyAsToken(enable bool) *SimpleAuthenticator {
	s.enableUseKeyAsToken = enable
	return s
}

//设置key，如果为空则直接返回
func (s *SimpleAuthenticator) SetKey(key string) {
	if !_string.Empty(key) {
		s.keyHolder.Store(key)
		return
	}

	s.logger.Warn(`key为空，将被忽略`)
}

func (s *SimpleAuthenticator) NewToken() string {
	key := s.keyHolder.Load().(string)
	if s.enableUseKeyAsToken {
		return key
	}

	return util.NewToken(key, time.Now())
}

func (s *SimpleAuthenticator) CheckToken(token string) error {
	key := s.keyHolder.Load().(string)
	if s.enableUseKeyAsToken {
		if key == token {
			return nil
		}

		return errors.New(`invalid token`)
	}

	return util.CheckToken(key, token, 2*time.Minute)
}

//客户端拦截器添加客户端认证信息
//
//也可考虑使用grpc.WithPerRPCCredentials()，实现credentials.PerRPCCredentials接口
//接口方法将在客户端每次发送请求时被调用以获取新的token，具体可参考官方示例
func (s *SimpleAuthenticator) NewUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if !s.disabled {
			val := fmt.Sprintf(`Bearer %v`, s.NewToken())
			ctx = metadata.AppendToOutgoingContext(ctx, `authorization`, val) //metadata会将http header名称转为小写
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

//服务端拦截器验证客户端身份
func (s *SimpleAuthenticator) NewUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if s.disabled {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, `metadata is empty`)
		}

		val, ok := GetMDVal(md, `authorization`, -1) //获取最后1个值，即覆盖ctx已有值
		if !ok || _string.Empty(val) {
			return nil, status.Error(codes.Unauthenticated, `token is empty`)
		}

		token := strings.TrimPrefix(val, "Bearer ")
		if err := s.CheckToken(token); err != nil {
			return nil, status.Errorf(codes.Unauthenticated, `invalid token[%v]->%w`, token, err)
		}

		return handler(ctx, req)
	}
}

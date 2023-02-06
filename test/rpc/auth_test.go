package rpc

import (
	"context"
	"fmt"
	"github.com/bingooh/b-go-util/rpc"
	"github.com/bingooh/b-go-util/test/rpc/pb"
	"github.com/bingooh/b-go-util/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"testing"
	"time"
)

var port = `:9090`

func mustStartAuthServer(auth *rpc.MWAuthenticator) *grpc.Server {
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(auth.NewUnaryServerInterceptor()),
		grpc.ChainStreamInterceptor(auth.NewStreamServerInterceptor()),
	)

	pb.RegisterGreeterServer(server, &pb.HiAuthServer{})
	rpc.MustStartServer(server, port)
	return server
}

func mustNewAuthClient(auth *rpc.MWAuthenticator) pb.GreeterClient {
	conn := rpc.MustNewInsecureClientConn(port, 0,
		grpc.WithChainUnaryInterceptor(auth.NewUnaryClientInterceptor()),
		grpc.WithChainStreamInterceptor(auth.NewStreamClientInterceptor()),
	)

	return pb.NewGreeterClient(conn) //未关闭连接
}

func consumeStream(stream grpc.ClientStream) {
	for {
		msg := &pb.HiRsp{}

		if err := stream.RecvMsg(msg); err != nil {
			fmt.Println(`client rev:`, err)
			return
		} else {
			fmt.Println(`client rev:`, msg.Msg)
		}
	}
}

func TestSimpleAuth(t *testing.T) {
	r := require.New(t)

	ctx := context.Background()
	req := &pb.HiReq{Name: `bb`}

	assertAuthFail := func(err error) {
		r.Equal(codes.Unauthenticated, status.Code(err))
	}

	o1 := &rpc.SimpleAuthOption{
		Key:                   "123",
		Disabled:              false,
		TokenHeaders:          nil,
		EnableUseKeyAsToken:   false,
		EnablePutTokenHeaders: false,
	}

	o2 := &rpc.SimpleAuthOption{Key: "456"}

	auth1 := rpc.NewSimpleAuthHandler(o1)
	auth1.WithOnCheckToken(func(ctx context.Context, key, token, method string) (val interface{}, err error) {
		var tk *util.Token
		tk, err = util.ParseAndCheckToken(key, token, 1*time.Minute)

		if err == nil && len(tk.Headers) > 0 {
			val = tk.Headers //将保存headers到ctx
		}

		return
	})

	auth2 := rpc.NewSimpleAuthHandler(o2)

	server := mustStartAuthServer(auth1.MW())
	defer server.GracefulStop()

	client := mustNewAuthClient(auth1.MW())
	_, err := client.Hi(ctx, req)
	r.NoError(err) //验证通过

	stream1, err := client.NewStream(ctx)
	r.NoError(err)
	consumeStream(stream1)

	stream2, err := client.NewServerStream(ctx, req)
	r.NoError(err)
	consumeStream(stream2)

	client2 := mustNewAuthClient(auth2.MW())
	_, err = client2.Hi(ctx, req)
	assertAuthFail(err) //验证失败

	stream1, err = client2.NewStream(ctx)
	r.NoError(err)
	consumeStream(stream1)

	stream2, err = client2.NewServerStream(ctx, &pb.HiReq{Name: `a`})
	r.NoError(err)
	consumeStream(stream2)

	//覆盖请求头metadata的验证信息，仍然可以请求成功,因会被客户端拦截器再次覆盖
	ctx2 := metadata.AppendToOutgoingContext(ctx, `authorization`, `xxx`)
	_, err = client.Hi(ctx2, req)
	r.NoError(err)

	//添加token头，如设置为用户ID。服务端将输出auth value为token headers
	o1.TokenHeaders = []string{`1`}
	_, err = client.Hi(ctx, req)
	r.NoError(err) //验证通过，需要通过断点打印token值
}

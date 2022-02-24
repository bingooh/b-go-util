package rpc

import (
	"context"
	"github.com/bingooh/b-go-util/rpc"
	"github.com/bingooh/b-go-util/test/rpc/pb"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"testing"
)

var port = `:9090`

func TestSimpleAuth(t *testing.T) {
	r := require.New(t)

	key1 := `123`
	key2 := `456`
	ctx := context.Background()
	req := &pb.HiReq{Name: `bb`}
	auth := rpc.MustNewSimpleAuthenticator(key1)
	//auth=auth.EnableUseKeyAsToken(true)//设置使用key作为token

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(auth.NewUnaryServerInterceptor()),
	)
	defer server.GracefulStop()

	pb.RegisterGreeterServer(server, &pb.HiServer{})
	rpc.MustStartServer(server, port)

	conn := rpc.MustNewInsecureClientConn(port, 0,
		grpc.WithChainUnaryInterceptor(auth.NewUnaryClientInterceptor()),
	)

	client := pb.NewGreeterClient(conn)
	_, err := client.Hi(ctx, req)
	r.NoError(err)

	auth.SetKey(key2)
	_, err = client.Hi(ctx, req)
	r.NoError(err)

	//覆盖请求头metadata的验证信息，仍然可以请求成功,因会被拦截器再次覆盖
	ctx2 := metadata.AppendToOutgoingContext(ctx, `authorization`, `xxx`)
	_, err = client.Hi(ctx2, req)
	r.NoError(err)

	//创建1个新的验证器，使用不同的密钥
	auth2 := rpc.MustNewSimpleAuthenticator(`xxx`)
	conn2 := rpc.MustNewInsecureClientConn(port, 0,
		grpc.WithChainUnaryInterceptor(auth2.NewUnaryClientInterceptor()),
	)
	client2 := pb.NewGreeterClient(conn2)
	_, err = client2.Hi(ctx, req)
	r.Error(err)
	r.Equal(codes.Unauthenticated, status.Code(err))
}

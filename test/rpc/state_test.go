package rpc

import (
	"context"
	"github.com/bingooh/b-go-util/rpc"
	"github.com/bingooh/b-go-util/test/rpc/pb"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"testing"
	"time"
)

func mustStartStateServer(mw1, mw2 *rpc.MWState) *grpc.Server {
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(mw1.NewUnaryServerInterceptor(), mw2.NewUnaryServerInterceptor()),
		grpc.ChainStreamInterceptor(mw1.NewStreamServerInterceptor(), mw2.NewStreamServerInterceptor()),
	)

	pb.RegisterGreeterServer(server, &pb.HiStateServer{})
	rpc.MustStartServer(server, port)
	return server
}

func mustNewStateClient(mw1, mw2 *rpc.MWState) pb.GreeterClient {
	conn := rpc.MustNewInsecureClientConn(port, 0,
		grpc.WithChainUnaryInterceptor(mw1.NewUnaryClientInterceptor(), mw2.NewUnaryClientInterceptor()),
		grpc.WithChainStreamInterceptor(mw1.NewStreamClientInterceptor(), mw2.NewStreamClientInterceptor()),
	)

	return pb.NewGreeterClient(conn) //未关闭连接
}

func TestMWState(t *testing.T) {
	r := require.New(t)

	ctx := context.Background()
	req := &pb.HiReq{Name: `bb`}

	mw1 := rpc.NewMWState(func(ctx context.Context, state rpc.State, method string) error {
		state.Put(1, 1)
		state.Put(`method`, method)
		return nil
	})

	mw2 := rpc.NewMWState(func(ctx context.Context, state rpc.State, method string) error {
		r.Equal(1, state.MustInt(1)) //来自mw1
		state.Put(`ts`, time.Now().UnixMilli())
		return nil
	})

	mustStartStateServer(mw1, mw2)
	client := mustNewStateClient(mw1, mw2)

	_, err1 := client.Hi(ctx, req)
	r.NoError(err1)

	stream2, err2 := client.NewStream(ctx)
	r.NoError(err2)
	consumeStream(stream2)

	stream3, err3 := client.NewServerStream(ctx, req)
	r.NoError(err3)
	consumeStream(stream3)

}

package rpc

import (
	"context"
	"github.com/bingooh/b-go-util/rpc"
	"github.com/bingooh/b-go-util/test/rpc/pb"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
)

func TestLogger(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	mw := rpc.NewMWLogger().EnableLogPayload(false)
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			mw.NewUnaryServerInterceptor(),
			mw.NewPayloadUnaryServerInterceptor(),
		),
	)
	defer server.GracefulStop()

	pb.RegisterGreeterServer(server, &pb.HiAuthServer{})
	rpc.MustStartServer(server, port)

	conn := rpc.MustNewInsecureClientConn(port, 0,
		grpc.WithChainUnaryInterceptor(
			mw.NewUnaryClientInterceptor(),
			mw.NewPayloadUnaryClientInterceptor(),
		),
	)
	client := pb.NewGreeterClient(conn)

	_, err := client.Hi(ctx, &pb.HiReq{Name: `bingo`})
	r.NoError(err)

	_, err = client.Hi(ctx, &pb.HiReq{Name: ``})
	r.Error(err)
	r.Equal(codes.InvalidArgument, status.Code(err))

}

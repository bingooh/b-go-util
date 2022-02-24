package rpc

import (
	"b-go-util/rpc"
	"b-go-util/test/rpc/pb"
	"context"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
)

func TestLogger(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	logger := rpc.NewGRPCLogger(`server`)
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			rpc.LogUnaryServerInterceptor(logger),
			rpc.LogPayloadUnaryServerInterceptor(logger),
		),
	)
	defer server.GracefulStop()

	pb.RegisterGreeterServer(server, &pb.HiServer{})
	rpc.MustStartServer(server, port)

	logger = rpc.NewGRPCLogger(`client`)
	conn := rpc.MustNewInsecureClientConn(port, 0,
		grpc.WithChainUnaryInterceptor(
			rpc.LogUnaryClientInterceptor(logger),
			rpc.LogPayloadUnaryClientInterceptor(logger),
		),
	)
	client := pb.NewGreeterClient(conn)

	_, err := client.Hi(ctx, &pb.HiReq{Name: `bingo`})
	r.NoError(err)

	_, err = client.Hi(ctx, &pb.HiReq{Name: ``})
	r.Error(err)
	r.Equal(codes.InvalidArgument, status.Code(err))

}

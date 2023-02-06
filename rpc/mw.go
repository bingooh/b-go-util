package rpc

import "google.golang.org/grpc"

type MW interface {
	NewUnaryClientInterceptor() grpc.UnaryClientInterceptor
	NewStreamClientInterceptor() grpc.StreamClientInterceptor
	NewUnaryServerInterceptor() grpc.UnaryServerInterceptor
	NewStreamServerInterceptor() grpc.StreamServerInterceptor
}

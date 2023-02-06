package pb

import (
	"context"
	"fmt"
	"github.com/bingooh/b-go-util/_string"
	"github.com/bingooh/b-go-util/rpc"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

type HiAuthServer struct {
	UnimplementedGreeterServer
}

func (s *HiAuthServer) Hi(ctx context.Context, req *HiReq) (*HiRsp, error) {
	fmt.Println(`auth value:`, rpc.AuthValueFromContext(ctx))

	if _string.Empty(req.Name) {
		//不建议使用此方式获取logger，建议使用tracing
		logger := ctxzap.Extract(ctx)
		logger.Error(`无效参数`)

		return nil, status.Error(codes.InvalidArgument, `name is empty`)
	}

	return &HiRsp{Msg: fmt.Sprintf(`hi,%v`, req.Name)}, nil
}

func (s *HiAuthServer) NewServerStream(req *HiReq, stream Greeter_NewServerStreamServer) error {
	fmt.Println(`auth value:`, rpc.AuthValueFromContext(stream.Context()))

	for i := 0; i < 3; i++ {
		if err := stream.Send(&HiRsp{Msg: time.Now().Format(`0405`)}); err != nil {
			return err
		}
		time.Sleep(500 * time.Millisecond)
	}

	return nil
}

func (s *HiAuthServer) NewStream(stream Greeter_NewStreamServer) error {
	fmt.Println(`auth value:`, rpc.AuthValueFromContext(stream.Context()))

	for i := 0; i < 3; i++ {
		if err := stream.Send(&HiRsp{Msg: time.Now().Format(`0405`)}); err != nil {
			return err
		}
		time.Sleep(500 * time.Millisecond)
	}

	return nil
}

type HiStateServer struct {
	UnimplementedGreeterServer
}

func (s *HiStateServer) printlnState(ctx context.Context) {
	state := rpc.StateFromContext(ctx)
	fmt.Println(`state:`, state.ToMap())
}

func (s *HiStateServer) Hi(ctx context.Context, req *HiReq) (*HiRsp, error) {
	s.printlnState(ctx)
	return &HiRsp{Msg: fmt.Sprintf(`hi,%v`, req.Name)}, nil
}

func (s *HiStateServer) NewServerStream(req *HiReq, stream Greeter_NewServerStreamServer) error {
	s.printlnState(stream.Context())
	return stream.Send(&HiRsp{Msg: `bye`})
}

func (s *HiStateServer) NewStream(stream Greeter_NewStreamServer) error {
	s.printlnState(stream.Context())
	return stream.Send(&HiRsp{Msg: `bye`})
}

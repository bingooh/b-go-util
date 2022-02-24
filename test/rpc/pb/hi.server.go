package pb

import (
	"b-go-util/_string"
	"context"
	"fmt"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type HiServer struct {
	UnimplementedGreeterServer
}

func (h *HiServer) Hi(ctx context.Context, req *HiReq) (*HiRsp, error) {
	if _string.Empty(req.Name) {

		//不建议使用此方式获取logger，建议使用tracing
		logger := ctxzap.Extract(ctx)
		logger.Error(`无效参数`)

		return nil, status.Error(codes.InvalidArgument, `name is empty`)
	}

	return &HiRsp{Msg: fmt.Sprintf(`hi,%v`, req.Name)}, nil
}

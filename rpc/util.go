package rpc

import (
	"b-go-util/async"
	"b-go-util/slog"
	"b-go-util/util"
	"context"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"net"
	"time"
)

func newLogger(tag string) *zap.Logger {
	return slog.NewLogger(`rpc`, tag)
}

func StartServer(server *grpc.Server, listenAddress string) error {
	listener, err := net.Listen(`tcp`, listenAddress)
	if err == nil {
		async.EnsureRun(func() {
			err = server.Serve(listener)
		})
	}

	return err
}

func MustStartServer(server *grpc.Server, listenAddress string) {
	util.AssertNilErr(StartServer(server, listenAddress), `start server failed`)
}

func MustNewClientConn(server string, opts ...grpc.DialOption) *grpc.ClientConn {
	conn, err := grpc.Dial(server, opts...)
	util.AssertNilErr(err, `创建GRPC客户端连接出错[server=%v]`, server)
	return conn
}

func MustNewClientConnWithContext(ctx context.Context, server string, opts ...grpc.DialOption) *grpc.ClientConn {
	conn, err := grpc.DialContext(ctx, server, opts...)
	util.AssertNilErr(err, `创建GRPC客户端连接出错[server=%v]`, server)
	return conn
}

//如果请求参数timeout>0，则将添加grpc.WithBlock()，即等待连接成功或超时
func MustNewInsecureClientConn(server string, timeout time.Duration, opts ...grpc.DialOption) *grpc.ClientConn {
	o := []grpc.DialOption{grpc.WithInsecure()}
	if timeout > 0 {
		o = append(o, grpc.WithBlock())
	}
	opts = append(o, opts...) //输入参数优先级更高

	if timeout <= 0 {
		return MustNewClientConn(server, opts...)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return MustNewClientConnWithContext(ctx, server, opts...)
}

//获取md的值,idx可为负数，如-1表示获取最后1个值
func GetMDVal(md metadata.MD, key string, idx int) (string, bool) {
	if md == nil || len(md) == 0 {
		return ``, false
	}

	vals := md.Get(key)
	n := len(vals)
	switch {
	case n > 0 && idx >= 0 && idx < n:
		return vals[idx], true
	case n > 0 && idx < 0 && idx >= -n:
		return vals[idx+n], true
	default:
		return ``, false
	}
}

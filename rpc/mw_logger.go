package rpc

import (
	"context"
	"github.com/bingooh/b-go-util/slog"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

//创建用于输出grpc日志的日志器，默认读取conf/log_grpc配置文件，如无则调用slog.NewLogger()
func NewGRPCLogger(tag string) *zap.Logger {
	if logger, err := slog.NewLoggerFromCfgFile(`log_grpc`); err == nil {
		return logger.With(slog.NewTagField(`grpc`, tag))
	}

	return slog.NewLogger(tag)
}

//错误码到日志级别
func CodeToLogLevel(code codes.Code) zapcore.Level {
	switch code {
	case codes.OK:
		return zapcore.InfoLevel
	case codes.Canceled, codes.Unknown, codes.DeadlineExceeded:
		return zapcore.WarnLevel
	default:
		return zapcore.ErrorLevel
	}
}

func DefaultServerPayloadLogDecider(ctx context.Context, fullMethodName string, servingObject interface{}) bool {
	return true
}

func DefaultClientPayloadLoggingDecider(ctx context.Context, fullMethodName string) bool {
	return true
}

//服务端日志拦截器
func LogUnaryServerInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	//如果使用grpc_ctxtags中间件，则应放在第1位
	return grpc_zap.UnaryServerInterceptor(logger, grpc_zap.WithLevels(CodeToLogLevel))
}

//服务端日志拦截器(显示消息内容)
//payload日志拦截器需要放在日志拦截器后面，payload日志拦截器的decider实现应考虑性能
func LogPayloadUnaryServerInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return grpc_zap.PayloadUnaryServerInterceptor(logger, DefaultServerPayloadLogDecider)
}

//客户端日志拦截器(显示消息内容)
func LogUnaryClientInterceptor(logger *zap.Logger) grpc.UnaryClientInterceptor {
	return grpc_zap.UnaryClientInterceptor(logger, grpc_zap.WithLevels(CodeToLogLevel))
}

//客户端日志拦截器
func LogPayloadUnaryClientInterceptor(logger *zap.Logger) grpc.UnaryClientInterceptor {
	return grpc_zap.PayloadUnaryClientInterceptor(logger, DefaultClientPayloadLoggingDecider)
}

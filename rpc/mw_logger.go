package rpc

import (
	"context"
	"github.com/bingooh/b-go-util/slog"
	"github.com/bingooh/b-go-util/util"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// NewGRPCLogger 创建用于输出grpc日志的日志器，默认读取conf/log_grpc配置文件，如无则调用slog.NewLogger()
func NewGRPCLogger(tag string) *zap.Logger {
	if logger, err := slog.NewLoggerFromCfgFile(`log_grpc`); err == nil {
		return logger.With(slog.NewTagField(`grpc`, tag))
	}

	return slog.NewLogger(tag)
}

// CodeToLogLevel 错误码到日志级别
func CodeToLogLevel(code codes.Code) zapcore.Level {
	switch code {
	case codes.OK:
		return zapcore.DebugLevel
	case codes.Canceled, codes.Unknown, codes.DeadlineExceeded:
		return zapcore.WarnLevel
	default:
		return zapcore.ErrorLevel
	}
}

// 服务端日志拦截器
func LogUnaryServerInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	//如果使用grpc_ctxtags中间件，则应放在第1位
	return grpc_zap.UnaryServerInterceptor(logger, grpc_zap.WithLevels(CodeToLogLevel))
}

// 服务端日志拦截器(显示消息内容)
// payload日志拦截器需要放在日志拦截器后面，payload日志拦截器的decider实现应考虑性能
func LogPayloadUnaryServerInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return grpc_zap.PayloadUnaryServerInterceptor(logger, func(ctx context.Context, fullMethodName string, servingObject interface{}) bool {
		return true
	})
}

// 客户端日志拦截器(显示消息内容)
func LogUnaryClientInterceptor(logger *zap.Logger) grpc.UnaryClientInterceptor {
	return grpc_zap.UnaryClientInterceptor(logger, grpc_zap.WithLevels(CodeToLogLevel))
}

// 客户端日志拦截器
func LogPayloadUnaryClientInterceptor(logger *zap.Logger) grpc.UnaryClientInterceptor {
	return grpc_zap.PayloadUnaryClientInterceptor(logger, func(ctx context.Context, fullMethodName string) bool {
		return true
	})
}

type MWLogger struct {
	logger           *zap.Logger
	enableLogPayload bool
}

func NewMWLogger() *MWLogger {
	return NewMWLoggerWithTag(`grpc`)
}

func NewMWLoggerWithTag(tag string) *MWLogger {
	return &MWLogger{logger: NewGRPCLogger(tag)}
}

func (l *MWLogger) WithLogger(logger *zap.Logger) *MWLogger {
	util.AssertOk(logger != nil, `logger为空`)
	l.logger = logger
	return l
}

func (l *MWLogger) EnableLogPayload(enable bool) *MWLogger {
	l.enableLogPayload = enable
	return l
}

func (l *MWLogger) clientPayloadLoggingDecider(ctx context.Context, fullMethodName string) bool {
	return l.enableLogPayload
}

func (l *MWLogger) serverPayloadLogDecider(ctx context.Context, fullMethodName string, servingObject interface{}) bool {
	return l.enableLogPayload
}

func (l *MWLogger) NewUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return grpc_zap.UnaryClientInterceptor(l.logger, grpc_zap.WithLevels(CodeToLogLevel))
}

func (l *MWLogger) NewStreamClientInterceptor() grpc.StreamClientInterceptor {
	return grpc_zap.StreamClientInterceptor(l.logger, grpc_zap.WithLevels(CodeToLogLevel))
}

func (l *MWLogger) NewUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	//如果使用grpc_ctxtags中间件，则应放在第1位
	return grpc_zap.UnaryServerInterceptor(l.logger, grpc_zap.WithLevels(CodeToLogLevel))
}

func (l *MWLogger) NewStreamServerInterceptor() grpc.StreamServerInterceptor {
	return grpc_zap.StreamServerInterceptor(l.logger, grpc_zap.WithLevels(CodeToLogLevel))
}

func (l *MWLogger) NewPayloadUnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return grpc_zap.PayloadUnaryClientInterceptor(l.logger, l.clientPayloadLoggingDecider)
}

func (l *MWLogger) NewPayloadUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return grpc_zap.PayloadUnaryServerInterceptor(l.logger, l.serverPayloadLogDecider)
}

package http

import (
	"b-go-util/async"
	"b-go-util/util"
	"context"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type Server struct {
	logger *zap.Logger
	server *http.Server
}

func MustNewServer(addr string, handler http.Handler) *Server {
	//参数handler可以为空，此情况使用默认的http.DefaultServeMux
	server := &http.Server{Addr: addr, Handler: handler}
	return &Server{server: server, logger: newLogger(`server`)}
}

func MustStartServer(addr string, handler http.Handler) *Server {
	server := MustNewServer(addr, handler)
	server.Start()
	return server
}

func (s *Server) Run() error {
	return s.server.ListenAndServe()
}

func (s *Server) Start() {
	async.EnsureRun(func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(util.NewInternalError(err, `http服务器启动出错`))
		}
	})

	s.logger.Sugar().Infof(`http服务器已启动，监听地址[%v]`, s.server.Addr)
}

func (s *Server) Stop() {
	if s == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Warn(`http服务器关闭出错`, zap.Error(err))
		return
	}

	s.logger.Info(`http服务器已关闭`)
}

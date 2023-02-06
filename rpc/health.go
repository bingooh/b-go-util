package rpc

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"time"
)

func IsHealthStatusServing(status healthpb.HealthCheckResponse_ServingStatus) bool {
	return status == healthpb.HealthCheckResponse_SERVING
}

type HealthServer struct {
	*health.Server
}

func NewHealthServer(server *grpc.Server) *HealthServer {
	s := &HealthServer{Server: health.NewServer()} //health.NewServer()默认状态为SERVING
	s.SetStatusNotServing(``)
	if server != nil {
		healthpb.RegisterHealthServer(server, s)
	}
	return s
}

func (s *HealthServer) SetStatusServing(service string) {
	s.SetServingStatus(service, healthpb.HealthCheckResponse_SERVING)
}

func (s *HealthServer) SetStatusNotServing(service string) {
	s.SetServingStatus(service, healthpb.HealthCheckResponse_NOT_SERVING)
}

type HealthClient struct {
	healthpb.HealthClient
	conn *grpc.ClientConn
}

func NewHealthClient(conn *grpc.ClientConn) *HealthClient {
	return &HealthClient{HealthClient: healthpb.NewHealthClient(conn), conn: conn}
}

func MustNewHealthClient(server string) *HealthClient {
	conn := MustNewClientConnWithTimeout(server, 5*time.Second,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	return &HealthClient{HealthClient: healthpb.NewHealthClient(conn), conn: conn}
}

func (c *HealthClient) Close() error {
	if conn := c.conn; conn != nil {
		c.conn = nil
		return conn.Close()
	}

	return nil
}

func (c *HealthClient) CheckStatus(ctx context.Context, service string) (healthpb.HealthCheckResponse_ServingStatus, error) {
	if rsp, err := c.Check(ctx, &healthpb.HealthCheckRequest{Service: service}); err != nil {
		return healthpb.HealthCheckResponse_UNKNOWN, err
	} else {
		return rsp.GetStatus(), nil
	}
}

func (c *HealthClient) CheckStatusServing(ctx context.Context, service string) (bool, error) {
	if s, err := c.CheckStatus(ctx, service); err != nil {
		return false, err
	} else {
		return IsHealthStatusServing(s), nil
	}
}

func (c *HealthClient) WatchStatus(ctx context.Context, service string) (<-chan healthpb.HealthCheckResponse_ServingStatus, error) {
	stream, err := c.Watch(ctx, &healthpb.HealthCheckRequest{Service: service})
	if err != nil {
		return nil, err
	}

	ch := make(chan healthpb.HealthCheckResponse_ServingStatus, 1)
	go func() {
		defer close(ch)

		for {
			if rsp, err := stream.Recv(); err != nil {
				//util.Log(err, `health stream receive err`)
				return
			} else {
				ch <- rsp.Status
			}

		}
	}()

	return ch, nil
}

func (c *HealthClient) WatchStatusServing(ctx context.Context, service string) <-chan error {
	done := make(chan error, 1)
	go func() {
		defer close(done)
		done <- c.WaitStatusServing(ctx, service)
	}()

	return done
}

func (c *HealthClient) WaitStatusServing(ctx context.Context, service string) error {
	ch, err := c.WatchStatus(ctx, service)
	if err != nil {
		return err
	}

	for s := range ch {
		if IsHealthStatusServing(s) {
			return nil
		}
	}

	if ctx.Err() != nil {
		return nil
	}

	return status.Error(codes.Unavailable, `health stream closed`)
}

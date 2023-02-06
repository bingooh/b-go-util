package rpc

import (
	"context"
	"fmt"
	"github.com/bingooh/b-go-util/rpc"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"testing"
	"time"
)

func TestHealthServer(t *testing.T) {
	r := require.New(t)
	ctx := context.TODO()
	service := `` //空表示系统状态

	svr := grpc.NewServer()
	server := rpc.NewHealthServer(svr) //默认状态为NOT_SERVING
	rpc.MustStartServer(svr, port)
	defer svr.Stop() //不能优雅关闭，因为无法关闭health服务端流

	rollServerStatus := func() {
		start := healthpb.HealthCheckResponse_UNKNOWN
		end := healthpb.HealthCheckResponse_NOT_SERVING
		for status := start; status <= end; status++ {
			server.SetServingStatus(service, status)
			time.Sleep(1 * time.Second)
		}
	}

	client := rpc.MustNewHealthClient(port)

	checkStatus := func(expect healthpb.HealthCheckResponse_ServingStatus) {
		status, err := client.CheckStatus(ctx, service)
		r.NoError(err)
		r.Equal(expect, status)
	}

	checkStatus(healthpb.HealthCheckResponse_NOT_SERVING) //默认状态
	server.SetServingStatus(service, healthpb.HealthCheckResponse_UNKNOWN)
	checkStatus(healthpb.HealthCheckResponse_UNKNOWN)

	start := time.Now()
	time.AfterFunc(1*time.Second, func() {
		server.SetServingStatus(service, healthpb.HealthCheckResponse_SERVING)
	})
	r.NoError(client.WaitStatusServing(ctx, service)) //阻塞等待状态变为SERVING
	r.WithinDuration(time.Now(), start.Add(1*time.Second), 100*time.Millisecond)

	go rollServerStatus()

	//如果无法连接，返回err。如果连接断开，会导致关闭ch
	//传入ctx以关闭返回的管道
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	ch, err := client.WatchStatus(ctx, service)
	r.NoError(err)
	for status := range ch {
		fmt.Println(`status:`, status)
	}
	fmt.Println(`done`)

}

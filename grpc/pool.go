package grpc

import (
	"context"
	"fmt"
	"github.com/bingooh/b-go-util/util"
	gopool "github.com/processout/grpc-go-pool"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"time"
)

type Pool struct {
	option *ClientOption
	pool   *gopool.Pool
	logger zerolog.Logger
}

type ClientConn struct {
	*gopool.ClientConn
}

func connectionFactory(o *ClientOption) func() (*grpc.ClientConn, error) {
	return func() (*grpc.ClientConn, error) {
		opts := make([]grpc.DialOption, 0)

		if o.WithBlock {
			opts = append(opts, grpc.WithBlock())
		}

		if o.WithInsecure {
			opts = append(opts, grpc.WithInsecure())
		}

		return grpc.Dial(o.Server, opts...)
	}
}

func NewPool(option *ClientOption, logger zerolog.Logger) (*Pool, error) {
	if option == nil {
		return nil, util.NewIllegalArgError("option is nil")
	}

	if option.PoolOption == nil {
		return nil, util.NewIllegalArgError("pool option is nil")
	}

	o := option
	factory := connectionFactory(o)
	p, err := gopool.New(factory, o.MinSize, o.MaxSize, o.IdleTimeout, o.MaxLifeTimeout)
	if err != nil {
		return nil, fmt.Errorf("grpc pool init fail->%w", err)
	}

	return &Pool{option: option, pool: p, logger: logger}, nil
}

func MustNewPool(option *ClientOption, logger zerolog.Logger) *Pool {
	if pool, err := NewPool(option, logger); err != nil {
		panic(err)
	} else {
		return pool
	}
}

// 一直等待直到获取可用连接
func (p *Pool) Get() (*ClientConn, error) {
	return p.GetWithTimeout(p.option.GetConnTimeout)
}

func (p *Pool) GetWithContext(ctx context.Context) (*ClientConn, error) {
	conn, err := p.pool.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get client conn err->%w", err)
	}

	return &ClientConn{conn}, nil
}

func (p *Pool) GetWithTimeout(timeout time.Duration) (*ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return p.GetWithContext(ctx)
}

func (p *Pool) Put(conn *ClientConn) {
	if conn == nil {
		return
	}

	if err := conn.Close(); err != nil {
		conn.Unhealthy()
		p.logger.Warn().Err(err).Msg("close grpc client conn err")
	}
}

func (p *Pool) IsClosed() bool {
	return p.pool.IsClosed()
}

func (p *Pool) Close() {
	p.pool.Close()
}

func (p *Pool) Available() int {
	return p.pool.Available()
}

func (p *Pool) Capacity() int {
	return p.pool.Capacity()
}

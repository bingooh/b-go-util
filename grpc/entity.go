package grpc

import (
	"time"
)

type PoolOption struct {
	MinSize        int
	MaxSize        int
	IdleTimeout    time.Duration
	MaxLifeTimeout time.Duration
	GetConnTimeout time.Duration //获取连接超时时间
}

type ClientOption struct {
	*PoolOption
	Server       string
	WithBlock    bool
	WithInsecure bool
}

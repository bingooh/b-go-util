package rdb

import (
	"github.com/go-redis/redis/v8"
	"math"
	"strconv"
	"time"
)

func NewZ(score int64, member interface{}) *redis.Z {
	return &redis.Z{
		Score:  float64(score),
		Member: member,
	}
}

func NewZNow(member interface{}) *redis.Z {
	return NewZ(time.Now().Unix(), member)
}

func NewZSlice(score int64, members ...interface{}) (zs []*redis.Z) {
	for _, member := range members {
		zs = append(zs, NewZ(score, member))
	}

	return
}

func NewZNowSlice(members ...interface{}) []*redis.Z {
	return NewZSlice(time.Now().Unix(), members...)
}

type ZRangeOption struct {
	low, high               bool // 是否排除上下限边界值，默认包含
	min, max, offset, count int64
}

//min,max指分数，可设置为Math.MinInt64,math.MaxInt64
func NewZRangeOption(min, max int64) *ZRangeOption {
	return &ZRangeOption{min: min, max: max}
}

func (z *ZRangeOption) Limit(offset, count int64) *ZRangeOption {
	z.offset, z.count = offset, count
	return z
}

// 是否排除上下限边界值，默认包含
func (z *ZRangeOption) Exclude(low, high bool) *ZRangeOption {
	z.low, z.high = low, high
	return z
}

func (z *ZRangeOption) Build() *redis.ZRangeBy {
	return &redis.ZRangeBy{
		Min:    z.bound(z.min, z.low),
		Max:    z.bound(z.max, z.high),
		Offset: z.offset,
		Count:  z.count,
	}
}

func (z *ZRangeOption) bound(n int64, exclusive bool) string {
	switch n {
	case math.MaxInt64:
		return "+inf"
	case math.MinInt64:
		return "-inf"
	default:
		s := strconv.FormatInt(n, 10)
		if exclusive {
			s = "(" + s
		}
		return s
	}
}

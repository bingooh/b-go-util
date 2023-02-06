package rdb

import (
	"github.com/bingooh/b-go-util/slog"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

func newLogger(tag string) *zap.Logger {
	return slog.NewLogger(`rdb`, tag)
}

//HMGet().Scan()仅支持扫描Struct(属性名称对应Hash的field)，不支持扫描slice
//假设执行HMGet(field)返回cmd，调用cmd.Result()返回的结果数据类型：
// - 如果HMGet(field)的field对应的hash的field不存在，则对应的val为nil
// - 如果HMGet(field)的field对应的hash的field存在，则对应的val为string，可能为空字符串

func ForEachSliceItem(cmd *redis.SliceCmd, fn func(v string) error) error {
	rs, err := cmd.Result()
	if err != nil {
		return err
	}

	for _, v := range rs {
		s, _ := v.(string)
		if err := fn(s); err != nil {
			return err
		}
	}

	return nil
}

func ForEachStringMapItem(cmd *redis.StringStringMapCmd, fn func(k, v string) error) error {
	rs, err := cmd.Result()
	if err != nil {
		return err
	}

	for k, v := range rs {
		if err := fn(k, v); err != nil {
			return err
		}
	}

	return nil
}

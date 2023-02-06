package rdb

import (
	"context"
	"github.com/go-redis/redis/v8"
)

// 以下脚本，如果key不存在，则取零值，可匹配0，false,“ 注：redis将lua的nil转换为false
var (
	setEQScript  = redis.NewScript(`local v=redis.call("get", KEYS[1]) or '';if v == ARGV[1] then redis.call("set", KEYS[1], ARGV[2]);return 1 else return 0 end`)
	delEQScript  = redis.NewScript(`local v=redis.call("get", KEYS[1]) or '';if v == ARGV[1] then redis.call("del", KEYS[1]);return 1 else return 0 end`)
	incrEQScript = redis.NewScript(`local v=redis.call("get", KEYS[1]) or '';if v == ARGV[1] then redis.call("incrby", KEYS[1], ARGV[2]) end;return v`)

	setNEQScript  = redis.NewScript(`local v=redis.call("get", KEYS[1]) or '';if v ~= ARGV[1] then redis.call("set", KEYS[1], ARGV[2]);return 1 else return 0 end`)
	delNEQScript  = redis.NewScript(`local v=redis.call("get", KEYS[1]) or '';if v ~= ARGV[1] then redis.call("del", KEYS[1]);return 1 else return 0 end`)
	incrNEQScript = redis.NewScript(`local v=redis.call("get", KEYS[1]) or '';if v ~= ARGV[1] then redis.call("incrby", KEYS[1], ARGV[2]) end;return v`)

	hsetEQScript  = redis.NewScript(`local v=redis.call("hget", KEYS[1], KEYS[2]) or '';if v == ARGV[1] then redis.call("hset", KEYS[1], KEYS[2], ARGV[2]);return 1 else return 0 end`)
	hdelEQScript  = redis.NewScript(`local v=redis.call("hget", KEYS[1], KEYS[2]) or '';if v == ARGV[1] then redis.call("hdel", KEYS[1], KEYS[2]);return 1 else return 0 end`)
	hincrEQScript = redis.NewScript(`local v=redis.call("hget", KEYS[1], KEYS[2]) or '';if v == ARGV[1] then redis.call("hincrby", KEYS[1], KEYS[2], ARGV[2]);return 1 else return 0 end`)
)

// 如果值相等则设置
func SetEQ(ctx context.Context, client redis.Scripter, key string, expect interface{}, val interface{}) (bool, error) {
	return setEQScript.Run(ctx, client, []string{key}, expect, val).Bool()
}

// 如果值相等则删除
func DelEQ(ctx context.Context, client redis.Scripter, key string, expect interface{}) (bool, error) {
	return delEQScript.Run(ctx, client, []string{key}, expect).Bool()
}

// 如果值相等则新增，返回当前值
func IncrEQ(ctx context.Context, client redis.Scripter, key string, expect, val int64) (bool, int64, error) {
	v, err := incrEQScript.Run(ctx, client, []string{key}, expect, val).Int64() //v为更新前的值
	if err != nil || v != expect {
		return false, v, err
	}

	return true, v + val, nil
}

// 如果值不相等则设置
func SetNEQ(ctx context.Context, client redis.Scripter, key string, expect interface{}, val interface{}) (bool, error) {
	return setNEQScript.Run(ctx, client, []string{key}, expect, val).Bool()
}

// 如果值不相等则删除
func DelNEQ(ctx context.Context, client redis.Scripter, key string, expect interface{}) (bool, error) {
	return delNEQScript.Run(ctx, client, []string{key}, expect).Bool()
}

// 如果值不相等则新增，返回当前值
// 注意：如果val==0，则会造成返回bool误判
func IncrNEQ(ctx context.Context, client redis.Scripter, key string, expect, val int64) (bool, int64, error) {
	v, err := incrNEQScript.Run(ctx, client, []string{key}, expect, val).Int64() //v为更新前的值
	if err != nil || v == expect {
		return false, v, err
	}

	return true, v + val, nil
}

// 如果值相等则设置
func HSetEQ(ctx context.Context, client redis.Scripter, key, field string, expect interface{}, val interface{}) (bool, error) {
	return hsetEQScript.Run(ctx, client, []string{key, field}, expect, val).Bool()
}

// 如果值相等则删除
func HDelEQ(ctx context.Context, client redis.Scripter, key, field string, expect interface{}) (bool, error) {
	return hdelEQScript.Run(ctx, client, []string{key, field}, expect).Bool()
}

// 如果值相等则新增，成功返回新增后的值，失败返回0
func HIncrEQ(ctx context.Context, client redis.Scripter, key, field string, expect, val int64) (bool, int64, error) {
	ok, err := hincrEQScript.Run(ctx, client, []string{key, field}, expect, val).Bool()
	if err != nil || !ok {
		return false, 0, err
	}

	return true, expect + val, nil
}

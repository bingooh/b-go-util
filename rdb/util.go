package rdb

import (
	"b-go-util/_string"
	"b-go-util/slog"
	"bytes"
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

func newLogger(tag string) *zap.Logger {
	return slog.NewLogger(`rdb`, tag)
}

//HMGet().Scan()仅支持扫描Struct(属性名称对应Hash的field)，不支持扫描slice
//此方法支持扫描数组，但会忽略所有空值，即忽略nil和空字符串。参数rs必须为指针类型的slice
//假设执行HMGet(field)返回cmd，调用cmd.Result()返回的结果数据类型：
// - 如果HMGet(field)的field对应的hash的field不存在，则对应的val为nil
// - 如果HMGet(field)的field对应的hash的field存在，则对应的val为string，可能为空字符串
func ScanSlice(cmd *redis.SliceCmd, rs interface{}) error {
	vals, err := cmd.Result()
	if err != nil {
		return err
	}

	//拼接为json数组字符串，以便后面解析
	var b bytes.Buffer
	b.WriteString(`[`)

	hasWritten := false
	for _, val := range vals {
		if v, ok := val.(string); ok && !_string.Empty(v) {
			if hasWritten {
				b.WriteString(`,`)
			} else {
				hasWritten = true
			}

			b.WriteString(v)
		}
	}
	b.WriteString(`]`)

	return json.Unmarshal(b.Bytes(), rs)
}

func ScanStringMap(cmd *redis.StringStringMapCmd, rs interface{}) error {
	m, err := cmd.Result()
	if err != nil || len(m) == 0 {
		return err
	}

	//拼接为json数组字符串，以便后面解析
	var b bytes.Buffer
	b.WriteString(`[`)

	hasWritten := false
	for _, val := range m {
		if !_string.Empty(val) {
			if hasWritten {
				b.WriteString(`,`)
			} else {
				hasWritten = true
			}

			b.WriteString(val)
		}
	}
	b.WriteString(`]`)

	return json.Unmarshal(b.Bytes(), rs)
}

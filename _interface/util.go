package _interface

import "reflect"

func Of(items ...interface{}) []interface{} {
	return items
}

func ToIntSlice(items ...interface{}) []int {
	rs := make([]int, 0, len(items))
	for _, item := range items {
		if v, ok := item.(int); ok {
			rs = append(rs, v)
		}
	}

	return rs
}

func ToInt64Slice(items ...interface{}) []int64 {
	rs := make([]int64, 0, len(items))
	for _, item := range items {
		if v, ok := item.(int64); ok {
			rs = append(rs, v)
		}
	}

	return rs
}

func ToStringSlice(items ...interface{}) []string {
	rs := make([]string, 0, len(items))
	for _, item := range items {
		if v, ok := item.(string); ok {
			rs = append(rs, v)
		}
	}

	return rs
}

// 展平items为1维切片，此方法使用反射，暂仅支持1层嵌套。详见测试
func Flat(items ...interface{}) (rs []interface{}) {
	for _, item := range items {
		rv := reflect.ValueOf(item)

		if rv.Kind() != reflect.Slice {
			rs = append(rs, item)
			continue
		}

		for i := 0; i < rv.Len(); i++ {
			v := rv.Index(i).Interface()
			rs = append(rs, v)
		}
	}

	return
}

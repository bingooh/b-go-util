package _interface

import "reflect"

func ToInt64Slice(items ...interface{}) (rs []int64) {
	for _, item := range items {
		if v, ok := item.(int64); ok {
			rs = append(rs, v)
		}
	}

	return
}

func ToStringSlice(items ...interface{}) (rs []string) {
	for _, item := range items {
		if v, ok := item.(string); ok {
			rs = append(rs, v)
		}
	}

	return
}

//展平items为1维切片，此方法使用反射，暂仅支持1层嵌套。详见测试
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

package _int64

import "strconv"

type Int64Slice []int64

func (x Int64Slice) Len() int           { return len(x) }
func (x Int64Slice) Less(i, j int) bool { return x[i] < x[j] }
func (x Int64Slice) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }

func IfEls(ok bool, okVal, notOkVal int64) int64 {
	if ok {
		return okVal
	}

	return notOkVal
}

func ToIntSlice(items ...int64) []int {
	rs := make([]int, 0, len(items))
	for _, v := range items {
		rs = append(rs, int(v))
	}

	return rs
}

func ToStringSlice(items ...int64) []string {
	rs := make([]string, 0, len(items))
	for _, v := range items {
		rs = append(rs, strconv.FormatInt(v, 10))
	}

	return rs
}

func ToInterfaceSlice(items ...int64) []interface{} {
	rs := make([]interface{}, 0, len(items))
	for _, item := range items {
		rs = append(rs, item)
	}

	return rs
}

func OfIntSlice(items ...int) []int64 {
	rs := make([]int64, 0, len(items))
	for _, item := range items {
		rs = append(rs, int64(item))
	}

	return rs
}

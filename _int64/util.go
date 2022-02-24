package _int64

import "strconv"

func IfEls(ok bool, okVal, notOkVal int64) int64 {
	if ok {
		return okVal
	}

	return notOkVal
}

func ToStringSlice(items ...int64) []string {
	rs := make([]string, 0, len(items))
	for _, v := range items {
		rs = append(rs, strconv.FormatInt(v, 10))
	}

	return rs
}

func ToInterfaceSlice(items ...int64) []interface{} {
	list := make([]interface{}, 0, len(items))

	for _, item := range items {
		list = append(list, item)
	}

	return list
}

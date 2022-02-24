package _string

import (
	"regexp"
	"strconv"
	"strings"
)

var crlfRX = regexp.MustCompile(`\r?\n`)

func Empty(s string) bool {
	return strings.TrimSpace(s) == ""
}

//返回第1个非空字符串
func FirstNotEmpty(items ...string) string {
	for _, item := range items {
		if !Empty(item) {
			return item
		}
	}

	return ``
}

func IfEls(ok bool, okVal, notOkVal string) string {
	if ok {
		return okVal
	}

	return notOkVal
}

//移除所有换行符
func TrimCRLF(v string) string {
	return crlfRX.ReplaceAllString(v, ``)
}

func ToInt64Slice(items ...string) ([]int64, error) {
	rs := make([]int64, 0, len(items))

	for _, s := range items {
		if n, err := strconv.ParseInt(s, 10, 64); err != nil {
			return nil, err
		} else {
			rs = append(rs, n)
		}
	}

	return rs, nil
}

func ToInterfaceSlice(items ...string) []interface{} {
	list := make([]interface{}, 0, len(items))

	for _, item := range items {
		list = append(list, item)
	}

	return list
}

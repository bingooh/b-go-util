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

func ToBytes(s string) []byte {
	return []byte(s)
}

func Split(s, sep string) []string {
	if Empty(s) || s == sep {
		return nil
	}

	//如果s为空字符串或者仅包含分隔符，strings.Split()将返回1个包含空字符串元素的切片
	rs := strings.Split(s, sep)
	if s[len(s)-1:] == sep {
		return rs[:len(rs)-1] //去掉末尾空元素
	}

	return rs
}

// 返回第1个非空字符串
func FirstNotEmpty(items ...string) string {
	for _, item := range items {
		if !Empty(item) {
			return item
		}
	}

	return ``
}

func If(ok bool, okFn, notOkFn func() string) string {
	if ok {
		return okFn()
	}

	return notOkFn()
}

func IfEls(ok bool, okVal, notOkVal string) string {
	if ok {
		return okVal
	}

	return notOkVal
}

// 移除所有换行符
func TrimCRLF(v string) string {
	return crlfRX.ReplaceAllString(v, ``)
}

func ToIntSlice(items ...string) ([]int, error) {
	rs := make([]int, 0, len(items))
	for _, s := range items {
		if n, err := strconv.Atoi(s); err != nil {
			return nil, err
		} else {
			rs = append(rs, n)
		}
	}

	return rs, nil
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
	rs := make([]interface{}, 0, len(items))
	for _, item := range items {
		rs = append(rs, item)
	}

	return rs
}

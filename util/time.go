package util

import (
	"fmt"
	"strconv"
	"time"
)

const (
	PLAIN_DATE_FORMAT = `20060102`
	PLAIN_TIME_FORMAT = `20060102150405`
)

func FormatUnixTime(t time.Time) string {
	return strconv.FormatInt(t.Unix(), 10)
}

func FormatUnixNow() string {
	return FormatUnixTime(time.Now())
}

//时间戳毫秒(13位)
func TimeUnixMills(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

func TimeNowUnixMills() int64 {
	return TimeUnixMills(time.Now())
}

//解析Unix时间戳，参数v可以是string/int/int64/uint64，秒或毫秒(10/13位)
func ParseUnixTime(val interface{}) (time.Time, error) {
	var tss string
	switch v := val.(type) {
	case int:
		tss = strconv.Itoa(v)
	case int64:
		tss = strconv.FormatInt(v, 10)
	case uint64:
		tss = strconv.FormatInt(int64(v), 10)
	case string:
		tss = v
	default:
		return time.Time{}, fmt.Errorf("invalid unix timestamp data type '%T(%v)'", val, val)
	}

	n := len(tss)
	if n != 10 && n != 13 {
		return time.Time{}, fmt.Errorf("invalid unix timestamp '%v', it's length must be 10/13", val)
	}

	ts, err := strconv.ParseInt(tss, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid unix timestamp val '%v'", val)
	}

	if n == 13 {
		return time.UnixMilli(ts), nil
	}

	return time.Unix(ts, 0), nil
}

func FormatPlainDate(t time.Time) string {
	return t.Format(PLAIN_DATE_FORMAT)
}

func FormatPlainTime(t time.Time) string {
	return t.Format(PLAIN_TIME_FORMAT)
}

func ParsePlainDate(val string) (time.Time, error) {
	return time.ParseInLocation(PLAIN_DATE_FORMAT, val, time.Local)
}

func ParsePlainTime(val string) (time.Time, error) {
	return time.ParseInLocation(PLAIN_TIME_FORMAT, val, time.Local)
}

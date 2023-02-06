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

func Tomorrow() time.Time {
	return time.Now().Add(24 * time.Hour)
}

func FormatUnixTime(t time.Time) string {
	return strconv.FormatInt(t.Unix(), 10)
}

func FormatUnixNow() string {
	return FormatUnixTime(time.Now())
}

// 解析Unix时间戳，参数v可以是string/int/int64/uint64，秒或毫秒(10/13位)
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
	//t.In(time.UTC).Format(PLAIN_DATE_FORMAT)//指定0时区
	return t.Format(PLAIN_DATE_FORMAT)
}

func FormatPlainTime(t time.Time) string {
	return t.Format(PLAIN_TIME_FORMAT)
}

func FormatPlainUnixDate(sec int64) string {
	return time.Unix(sec, 0).Format(PLAIN_DATE_FORMAT)
}

func FormatPlainUnixTime(sec int64) string {
	return time.Unix(sec, 0).Format(PLAIN_TIME_FORMAT)
}

func ParsePlainDate(val string) (time.Time, error) {
	return time.ParseInLocation(PLAIN_DATE_FORMAT, val, time.Local)
}

func ParsePlainTime(val string) (time.Time, error) {
	return time.ParseInLocation(PLAIN_TIME_FORMAT, val, time.Local)
}

func TruncateToDay(v time.Time) time.Time {
	//_,offset:=now.Zone()//总是使用本地时区截断，以下减去时区偏移量以得到明天零时
	//tm:=now.Truncate(24*time.Hour).Add(0-time.Duration(offset)*time.Second)
	return time.Date(v.Year(), v.Month(), v.Day(), 0, 0, 0, 0, v.Location())
}

func TruncateToDayEnd(v time.Time) time.Time {
	return time.Date(v.Year(), v.Month(), v.Day(), 23, 59, 59, 0, v.Location())
}

// DurationToTomorrow 距离明天零时的剩余时长
func DurationToTomorrow() time.Duration {
	now := time.Now()
	return TruncateToDay(now.Add(24 * time.Hour)).Sub(now)
}

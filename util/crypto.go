package util

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/bingooh/b-go-util/_string"
	"io"
	"strconv"
	"strings"
	"time"
)

//获取plain的md5值，并转换为hex
func HexMd5(plain string) string {
	h := md5.New()

	if _, err := io.WriteString(h, plain); err != nil {
		fmt.Println("md5 err: ", err)
	}

	return hex.EncodeToString(h.Sum(nil))
}

//获取plain哈希值，算法hmac_sha512
func HS512(plain, key string) []byte {
	mac := hmac.New(sha512.New, []byte(key))
	mac.Write([]byte(plain))
	return mac.Sum(nil)
}

//创建token，格式为：header.ts.sign
//	header: (可选)每个值使用hex编码，多个值用点号分隔
//	ts    : 时间戳(秒数)，表示token的创建日期
//	sign  : 签名，算法为hmac_sha512(header.ts,key)->hex->to_lower_case
//
//举例:
// 	key   : abc
//  header: 1.2
//	ts    : 1617013547
//	token : 31.32.1617013547.e94ab5002db7ad2019576739f63044805cd87da52667f6bae48a39591e2b2932bb2eb0db4e6096d9bf95ffa4eabf5c9c70b08d1e4bc6389751e24aa68ee8b6ac
func NewToken(key string, ts time.Time, headers ...interface{}) string {
	var b strings.Builder
	for _, v := range headers {
		b.WriteString(hex.EncodeToString([]byte(fmt.Sprintf(`%v`, v))))
		b.WriteString(`.`)
	}
	b.WriteString(strconv.FormatInt(ts.Unix(), 10))

	sign := hex.EncodeToString(HS512(b.String(), key))
	b.WriteString(`.`)
	b.WriteString(sign)

	return b.String()
}

func ParseToken(token string) (headers []string, ts time.Time, sign string, err error) {
	if _string.Empty(token) {
		return nil, time.Time{}, ``, errors.New(`token为空`)
	}

	vs := strings.Split(token, `.`)
	n := len(vs)
	if n <= 1 {
		return nil, time.Time{}, ``, errors.New(`token无效`)
	}

	sign = vs[n-1]
	tsInt64, err := strconv.ParseInt(vs[n-2], 10, 64)
	if err != nil {
		return nil, time.Time{}, ``, errors.New(`token时间戳无效`)
	}
	ts = time.Unix(tsInt64, 0)

	for _, v := range vs[0 : n-2] {
		d, err := hex.DecodeString(v)
		if err != nil {
			return nil, time.Time{}, ``, fmt.Errorf(`token头信息解析出错[header=%v]`, v)
		}

		headers = append(headers, string(d))
	}

	return
}

//校验token是否有效，如果参数tokenTTL大于0，则token的时间戳超过+-tokenTTL视为已过期
func CheckToken(key, token string, tokenTTL time.Duration) error {
	if _string.Empty(key) {
		return errors.New(`key为空`)
	}

	vs := strings.Split(token, `.`)
	n := len(vs)
	if n <= 1 {
		return errors.New(`token无效`)
	}

	if tokenTTL > 0 {
		tsInt64, err := strconv.ParseInt(vs[n-2], 10, 64)
		if err != nil {
			return errors.New(`token时间戳无效`)
		}

		diff := time.Since(time.Unix(tsInt64, 0))
		if diff < 0 {
			diff = -diff
		}

		if diff > tokenTTL {
			return errors.New(`token已失效`)
		}
	}

	sign := vs[n-1]
	plain := token[:len(token)-len(sign)-1]
	expect := hex.EncodeToString(HS512(plain, key))
	if sign != expect {
		return errors.New(`token签名无效`)
	}

	return nil
}

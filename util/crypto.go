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

// 获取plain的md5值，并转换为hex
func HexMd5(plain string) string {
	h := md5.New()

	if _, err := io.WriteString(h, plain); err != nil {
		fmt.Println("md5 err: ", err)
	}

	return hex.EncodeToString(h.Sum(nil))
}

// 获取plain哈希值，算法hmac_sha512
func HS512(plain, key string) []byte {
	mac := hmac.New(sha512.New, []byte(key))
	mac.Write([]byte(plain))
	return mac.Sum(nil)
}

type Token struct {
	Headers   []string
	CreatedAt time.Time
	Sign      string
}

func NewToken(headers ...string) *Token {
	return &Token{CreatedAt: time.Now(), Headers: headers}
}

func (t *Token) AddHeader(headers ...interface{}) *Token {
	for _, header := range headers {
		switch h := header.(type) {
		case string:
			t.Headers = append(t.Headers, h)
		case []byte:
			t.Headers = append(t.Headers, string(h))
		default:
			t.Headers = append(t.Headers, fmt.Sprintf(`%v`, h))
		}
	}
	return t
}

func (t *Token) Val() string {
	var b strings.Builder

	for _, h := range t.Headers {
		b.WriteString(h)
		b.WriteString(`.`)
	}

	b.WriteString(strconv.FormatInt(t.CreatedAt.Unix(), 10))
	b.WriteString(`.`)
	b.WriteString(t.Sign)

	return b.String()
}

func (t *Token) String() string {
	return t.Val()
}

func (t *Token) Encode(key string) (string, error) {
	return EncodeToken(key, t)
}

func (t *Token) MustEncode(key string) string {
	return MustEncodeToken(key, t)
}

// EncodeToken 编码token，返回1个新token
// 编码格式：header.createdAt.sign
//
//	header    : 头数据(可选)，每个值使用hex编码，多个值用点号分隔
//	createdAt : 时间戳(秒数)，表示token的创建日期
//	sign      : 签名，算法为hmac_sha512(header.createdAt,key)->hex->to_lower_case
//
// 举例:
//
//		key       : abc
//	 header    : 1.2
//		createdAt : 1617013547
//		token     : 31.32.1617013547.e94ab5002db7ad2019576739f63044805cd87da52667f6bae48a39591e2b2932bb2eb0db4e6096d9bf95ffa4eabf5c9c70b08d1e4bc6389751e24aa68ee8b6ac
func EncodeToken(key string, token *Token) (string, error) {
	if _string.Empty(key) {
		return ``, errors.New(`key为空`)
	}

	if token == nil {
		return ``, errors.New(`token为空`)
	}

	var b strings.Builder

	for _, h := range token.Headers {
		v := hex.EncodeToString([]byte(h))

		b.WriteString(v)
		b.WriteString(`.`)
	}

	b.WriteString(strconv.FormatInt(token.CreatedAt.Unix(), 10))
	sign := hex.EncodeToString(HS512(b.String(), key))

	b.WriteString(`.`)
	b.WriteString(sign)

	return b.String(), nil
}

func MustEncodeToken(key string, token *Token) string {
	v, err := EncodeToken(key, token)
	AssertNilErr(err)
	return v
}

func ParseToken(token string) (t *Token, err error) {
	if _string.Empty(token) {
		return t, errors.New(`token为空`)
	}

	vs := strings.Split(token, `.`)
	n := len(vs)
	if n <= 1 {
		return t, errors.New(`token格式错误`)
	}

	t = &Token{}
	t.Sign = vs[n-1]
	tsInt64, err := strconv.ParseInt(vs[n-2], 10, 64)
	if err != nil {
		return t, errors.New(`token时间戳无效`)
	}
	t.CreatedAt = time.Unix(tsInt64, 0)

	for _, v := range vs[0 : n-2] {
		d, err := hex.DecodeString(v)
		if err != nil {
			return t, fmt.Errorf(`token头信息解析出错[header=%v]`, v)
		}

		t.Headers = append(t.Headers, string(d))
	}

	return
}

// ParseAndCheckToken 解析后检验token是否有效，校验通过返回解析后的token对象
// 如果参数ttl大于0，则token的时间戳超过+-ttl视为已过期
func ParseAndCheckToken(key string, token string, ttl time.Duration) (t *Token, err error) {
	if _string.Empty(key) {
		return nil, errors.New(`key为空`)
	}

	t, err = ParseToken(token)
	if err != nil {
		return nil, err
	}

	if ttl > 0 {
		diff := time.Since(t.CreatedAt)
		if diff < 0 {
			diff = -diff
		}

		if diff > ttl {
			return nil, errors.New(`token已失效`)
		}
	}

	plain := token[:strings.LastIndex(token, `.`)]
	expect := hex.EncodeToString(HS512(plain, key))
	if t.Sign != expect {
		return nil, errors.New(`token签名无效`)
	}

	return t, nil
}

func CheckToken(key string, token string, ttl time.Duration) error {
	_, err := ParseAndCheckToken(key, token, ttl)
	return err
}

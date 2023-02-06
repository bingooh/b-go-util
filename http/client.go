package http

import (
	"github.com/bingooh/b-go-util/_string"
	"github.com/bingooh/b-go-util/util"
	"github.com/go-resty/resty/v2"
)

type ClientOption struct {
	HostURL             string //服务端URL
	Debug               bool   //是否启用调试模式
	AuthToken           string //验证token
	AuthScheme          string //验证方式，默认bearer
	BasicAuthName       string //basic验证用户名
	BasicAuthPasswd     string //basic验证密码
	UserAgent           string //user agent
	UseDefaultUserAgent bool   //是否设置默认user agent
}

func (o *ClientOption) MustNormalize() *ClientOption {
	util.AssertOk(o != nil, `option为空`)

	return o
}

func NewClient(hostURL string, debug bool) *resty.Client {
	return resty.New().
		SetLogger(newLogger(`client`).Sugar()).
		SetDebug(debug).SetHostURL(hostURL)
}

func MustNewClient(option *ClientOption) *resty.Client {
	o := option.MustNormalize()
	c := NewClient(o.HostURL, o.Debug)

	if !_string.Empty(o.BasicAuthName) {
		c.SetBasicAuth(o.BasicAuthName, o.BasicAuthPasswd)
	}

	if !_string.Empty(o.AuthToken) {
		c.SetAuthToken(o.AuthToken)
	}

	if !_string.Empty(o.AuthScheme) {
		c.SetAuthScheme(o.AuthScheme)
	}

	if !_string.Empty(o.UserAgent) {
		c.SetHeader(`User-Agent`, o.UserAgent)
	} else if o.UseDefaultUserAgent {
		c.SetHeader(`User-Agent`, UserAgent)
	}

	return c
}

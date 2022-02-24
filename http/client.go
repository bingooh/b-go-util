package http

import (
	"b-go-util/_string"
	"b-go-util/util"
	"github.com/go-resty/resty/v2"
)

type ClientOption struct {
	HostURL         string //服务端URL
	Debug           bool   //是否启用调试模式
	AuthToken       string //验证token
	AuthScheme      string //验证方式，默认bearer
	BasicAuthName   string //basic验证用户名
	BasicAuthPasswd string //basic验证密码
}

func (o *ClientOption) MustNormalize() *ClientOption {
	util.AssertOk(o != nil, `option为空`)

	return o
}

func MustNewClient(option *ClientOption) *resty.Client {
	o := option.MustNormalize()

	c := resty.New().
		SetLogger(newLogger(`client`).Sugar()).
		SetDebug(o.Debug).
		SetHostURL(o.HostURL)

	if !_string.Empty(o.BasicAuthName) {
		c.SetBasicAuth(o.BasicAuthName, o.BasicAuthPasswd)
	}

	if !_string.Empty(o.AuthToken) {
		c.SetAuthToken(o.AuthToken)
	}

	if !_string.Empty(o.AuthScheme) {
		c.SetAuthScheme(o.AuthScheme)
	}

	return c
}

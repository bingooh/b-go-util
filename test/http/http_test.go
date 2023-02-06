package http

import (
	"errors"
	"fmt"
	"github.com/bingooh/b-go-util/conf"
	"github.com/bingooh/b-go-util/http"
	"github.com/bingooh/b-go-util/util"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
)

type Option struct {
	Client  *http.ClientOption
	Server  *http.ServerOption
	Session *http.MWSessionOption
}

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func mustNewOptionFromCfgFile() *Option {
	o := &Option{}
	conf.MustLoad(o, `http`)

	o.Server.MustNormalize()
	o.Server.ResetGinGlobalCfg()

	return o
}

func newClient() *resty.Client {
	option := mustNewOptionFromCfgFile().Client
	return http.MustNewClient(option)
}

func TestServer(t *testing.T) {
	option := mustNewOptionFromCfgFile()

	r := gin.New()
	r.Use(http.MWLogger())
	r.Use(http.MWPanicLogger())
	r.Use(http.NewMWErrorHandler().Handle) //默认从错误码解析前3位作为http响应码

	r.Use(http.MWSession(option.Session))

	r.NoRoute(http.NoRouteHandler())

	r.GET(`/session1`, func(c *gin.Context) {
		ss := sessions.Default(c)

		v, _ := ss.Get(`v1`).(string)
		v += c.Query(`v1`)

		ss.Set(`v1`, v)
		util.AssertNilErr(ss.Save())

		c.JSON(200, gin.H{`v1`: v})
	})

	r.GET(`/session2`, func(c *gin.Context) {
		ss := http.GetSession(c)

		v := ss.StringOrElse(`v2`, ``)
		v += c.Query(`v2`)

		util.AssertNilErr(ss.Set(`v2`, v))
		c.JSON(200, gin.H{`v2`: v})
	})

	r.GET(`/err`, func(c *gin.Context) {
		v := c.Query(`v`)
		switch v {
		case `ok`:
			c.JSON(200, `ok`)
		case `e1`:
			c.Error(errors.New(v)) //500
		case `e2`:
			c.Error(&gin.Error{
				Err:  errors.New(v),
				Type: gin.ErrorTypePublic, //500，其他类型都属于此情况
			})
		case `e3`:
			c.Error(&gin.Error{
				Err:  errors.New(v),
				Type: gin.ErrorTypeBind, //400,请求参数校验失败
			})
		case `e4`:
			c.Error(&gin.Error{
				Err:  util.NewIllegalArgError(`参数错误`), //400，即使Type不是gin.ErrorTypeBind
				Type: gin.ErrorTypePublic,
			})
		case `e5`:
			c.Error(util.NewIllegalArgError(`参数错误2`)) //400
		case `e6`:
			c.Error(http.NewError(429, util.ErrCodeTooOften, `请求太频繁`)) //429
		case `e7`:
			c.Error(errors.New(v))
			c.Abort() //200，表示放弃，需自行发送响应，如不发送则默认响应200
		case `panic`:
			panic(`故意崩溃`) //500
		}
	})

	//测试验证错误
	r.POST(`/validate`, func(c *gin.Context) {
		//user重新定义User字段，添加校验逻辑
		user := &struct {
			Name string `json:"name" binding:"required"`
			Age  int    `json:"age"`
		}{}

		rules := []validation.Rule{
			validation.Required.Error(`年龄不能为空`),
			validation.Min(1).ErrorObject(validation.NewError(`400900`, `年龄不能小于1`)), //可指定错误码和错误消息
			validation.Max(120),
		}

		//仅校验user.name
		if err := c.ShouldBindJSON(user); err != nil {
			c.Error(err)
			return
		}

		//校验user.age,但必须先从请求里读取并解析user，可考虑将go-ozzo用于service层的校验
		ageRules := validation.Field(&user.Age, rules...)
		if err := validation.ValidateStruct(user, ageRules); err != nil {
			c.Error(err)
			return
		}

		//使用自定义错误码
		if user.Age == 99 {
			c.Error(util.NewBizError(400901, `年龄不能为[99]`)) //会取前3位作为http响应码
			return
		}

		if user.Age == 88 {
			c.Error(util.NewBizError(902, `年龄不能为[88]`)) //错误码至少为4位，否则不会取前3位作为http响应码
			return
		}

		c.JSON(200, user)
	})

	server := http.MustNewServer(option.Server.ListenAddress, r)
	server.Run()
}

// 需启动Server
func TestSession(t *testing.T) {
	r := require.New(t)
	client := newClient()

	send := func(tag string) {
		expect := ``
		url := `session` + tag
		key := `v` + tag

		for i := 0; i < 10; i++ {
			s := strconv.Itoa(i)
			expect += s

			rs := make(map[string]string)

			rsp, err := client.R().
				SetQueryParam(key, s).
				SetResult(&rs).
				Get(url)
			r.NoError(err)

			rs2 := rsp.Result().(*map[string]string)
			r.Equal(&rs, rs2)

			actual := rs[key]
			r.Equal(expect, actual)
		}
	}

	send(`1`)
	send(`2`)
}

// 需启动Server
func TestErrHandler(t *testing.T) {
	r := require.New(t)
	client := newClient()

	send := func(v string, expectStatus int) {
		rsp, err := client.R().SetQueryParam(`v`, v).Get(`err`)
		r.NoError(err)

		fmt.Println(v, rsp.Status(), rsp.String())
		r.EqualValues(expectStatus, rsp.StatusCode())
	}

	send(`e1`, 500)
	send(`e2`, 500)
	send(`e3`, 400)
	send(`e4`, 400)
	send(`e5`, 400)
	send(`e6`, 429)
	send(`e7`, 200)
	send(`panic`, 500)

	//handler没有发送响应，默认返回200
	send(`nil`, 200)
}

// 需启动Server
func TestValidateErrHandler(t *testing.T) {
	r := require.New(t)
	client := newClient()

	send := func(tag string, user *User, expectStatus, expectCode int) {
		httpErr := &http.Error{}
		rsp, err := client.R().SetBody(user).SetError(httpErr).Post(`validate`)
		r.NoError(err)

		fmt.Println(tag, rsp.Status(), rsp.String())
		r.EqualValues(expectStatus, rsp.StatusCode())
		r.EqualValues(expectCode, httpErr.Code())
	}

	send(`NoName`, &User{}, 400, -1)
	send(`NoAge`, &User{Name: `b`}, 400, -1)
	send(`InvalidAge1`, &User{Name: `b`, Age: -1}, 400, 400900)
	send(`InvalidAge2`, &User{Name: `b`, Age: 200}, 400, -1)
	send(`Age99`, &User{Name: `b`, Age: 99}, 400, 400901)
	send(`Age88`, &User{Name: `b`, Age: 88}, 500, 902)
	send(`OK`, &User{Name: `b`, Age: 10}, 200, 0)
}

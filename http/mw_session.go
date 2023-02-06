package http

import (
	"github.com/bingooh/b-go-util/_string"
	"github.com/bingooh/b-go-util/util"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	gsessions "github.com/gorilla/sessions"
	"strconv"
)

type MWSessionRedisStoreOption struct {
	Addr     string   //redis地址
	Password string   //redis密码
	DB       int      //redis数据库实例
	Prefix   string   //session键名称前缀，默认为session
	Keys     []string //session加密密钥，需设置1个或偶数个，建议使用16/32/64字符
}

func (o *MWSessionRedisStoreOption) MustNormalize() *MWSessionRedisStoreOption {
	util.AssertOk(o != nil, `option为空`)
	util.AssertOk(!_string.Empty(o.Addr), `Addr为空`)

	return o
}

func (o *MWSessionOption) CookieKeysAsBytesSlice() [][]byte {
	rs := make([][]byte, 0, len(o.CookieKeys))
	for _, key := range o.CookieKeys {
		rs = append(rs, []byte(key))
	}

	return rs
}

type MWSessionOption struct {
	CookieName string                     //session cookie键名称
	CookieKeys []string                   //session cookie加密密钥。需设置1个或偶数个，建议使用16/32/64字符
	Cookie     *sessions.Options          //session cookie配置(可选)
	RedisStore *MWSessionRedisStoreOption //可选，默认使用CookieStore
}

func (o *MWSessionOption) MustNormalize() *MWSessionOption {
	util.AssertOk(o != nil, `option为空`)
	util.AssertOk(!_string.Empty(o.CookieName), `CookieName为空`)
	util.AssertOk(len(o.CookieKeys) > 0, `CookieKeys为空`)

	return o
}

// session中间件
func MWSession(option *MWSessionOption) gin.HandlerFunc {
	o := option.MustNormalize()

	var err error
	var store sessions.Store

	keys := o.CookieKeysAsBytesSlice()
	if o.RedisStore == nil {
		store = cookie.NewStore(keys...)
	} else {
		os := o.RedisStore.MustNormalize()
		store, err = redis.NewStoreWithDB(10, `tcp`, os.Addr, os.Password, strconv.Itoa(os.DB), keys...)
		util.AssertNilErr(err, `redis session store创建出错`)

		if !_string.Empty(os.Prefix) {
			util.AssertNilErr(redis.SetKeyPrefix(store, os.Prefix), `设置session键名前缀出错`)
		}
	}

	if o.Cookie != nil {
		store.Options(*o.Cookie)
	}

	return sessions.Sessions(o.CookieName, store)
}

// 自动保存session
func MWSaveSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.IsAborted() {
			return
		}

		if err := sessions.Default(c).Save(); err != nil {
			c.Error(util.NewInternalError(err, `会话保存失败`))
		}
	}
}

// 会话
type Session interface {
	GSession() sessions.Session
	Set(key, val interface{}) error              //设置并立刻保存
	Del(key interface{}) error                   //删除并立刻保存
	Clear() error                                //清空并立刻保存
	Values() (map[interface{}]interface{}, bool) //获取全部值
	Get(key interface{}) interface{}
	Int64(key interface{}) (int64, bool)
	String(key interface{}) (string, bool)
	MustInt64(key interface{}) int64
	MustString(key interface{}) string
	Int64OrElse(key interface{}, val int64) int64
	StringOrElse(key interface{}, val string) string
}

// 实现Session接口
type baseSession struct {
	gs sessions.Session
}

func (s *baseSession) save() error {
	return s.gs.Save()
}

func (s *baseSession) GSession() sessions.Session {
	return s.gs
}

func (s *baseSession) Set(key interface{}, val interface{}) error {
	s.gs.Set(key, val)
	return s.save()
}

func (s *baseSession) Del(key interface{}) error {
	s.gs.Delete(key)
	return s.save()
}

func (s *baseSession) Clear() error {
	s.gs.Clear()
	return s.save()
}

func (s *baseSession) Values() (map[interface{}]interface{}, bool) {
	gs, ok := s.gs.(interface {
		Session() *gsessions.Session
	})

	if !ok {
		return nil, false
	}

	return gs.Session().Values, true
}

func (s *baseSession) Get(key interface{}) interface{} {
	return s.gs.Get(key)
}

func (s *baseSession) Int64(key interface{}) (int64, bool) {
	v, ok := s.Get(key).(int64)
	return v, ok
}

func (s *baseSession) String(key interface{}) (string, bool) {
	v, ok := s.Get(key).(string)
	return v, ok
}

func (s *baseSession) MustInt64(key interface{}) int64 {
	if v, ok := s.Int64(key); ok {
		return v
	}

	v := s.Get(key)
	panic(util.NewAssertFailError(`session值不是int64[key=%v,value=%v(%T)]`, key, v, v))
}

func (s *baseSession) MustString(key interface{}) string {
	if v, ok := s.String(key); ok {
		return v
	}

	v := s.Get(key)
	panic(util.NewAssertFailError(`session值不是string[key=%v,value=%v(%T)]`, key, v, v))
}

func (s *baseSession) Int64OrElse(key interface{}, val int64) int64 {
	if v, ok := s.Int64(key); ok {
		return v
	}

	return val
}

func (s *baseSession) StringOrElse(key interface{}, val string) string {
	if v, ok := s.String(key); ok {
		return v
	}

	return val
}

// 获取Session，此方法依赖中间件MWSession()
func GetSession(c *gin.Context) Session {
	if v, exist := c.Get(KeySession); exist {
		s, ok := v.(Session)
		util.AssertOk(ok && s != nil, `session不存在或数据类型不匹配[key=%v,type=%T]`, KeySession, v)
		return s
	}

	s := &baseSession{gs: sessions.Default(c)}
	c.Set(KeySession, s)
	return s
}

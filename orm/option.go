package orm

import (
	"github.com/bingooh/b-go-util/util"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"time"
)

//数据库连接配置，详情参考sql标准库的db.SetXX()方法
type ConnOption struct {
	MaxIdleConns    int           //最大空闲连接数
	MaxOpenConns    int           //最大打开连接数
	ConnMaxIdleTime time.Duration //连接最大空闲时间
	ConnMaxLifeTime time.Duration //连接最大生存时间
}

//驱动配置优先级:MySQL=>PgSQL
type Option struct {
	MySQL *mysql.Config    //mysql驱动配置
	PgSQL *postgres.Config //pgsql驱动配置

	Conn *ConnOption //数据库连接配置

	GORM           *gorm.Config           //gorm配置
	Log            LoggerOption           //日志配置
	NamingStrategy *schema.NamingStrategy //数据库表等命名规则
}

func (o *Option) MustNormalize() *Option {
	util.AssertOk(o != nil, `option为空`)
	util.AssertOk(o.MySQL != nil || o.PgSQL != nil, `数据库驱动配置全部为空`)

	if o.GORM == nil {
		o.GORM = &gorm.Config{}
	}

	if o.NamingStrategy != nil {
		o.GORM.NamingStrategy = o.NamingStrategy
	}

	return o
}

func (o *Option) NewDialer() gorm.Dialector {
	if o.MySQL != nil {
		return mysql.New(*o.MySQL)
	}

	if o.PgSQL != nil {
		return postgres.New(*o.PgSQL)
	}

	return nil
}

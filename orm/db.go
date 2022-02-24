package orm

import (
	"context"
	"github.com/bingooh/b-go-util/conf"
	"github.com/bingooh/b-go-util/util"
	"gorm.io/gorm"
)

var defaultDB *gorm.DB //需自行初始化

func ResetDefaultDB(db *gorm.DB) {
	util.AssertOk(db != nil, `db为空`)
	db.Logger.Info(context.Background(), `设置全局默认DB`)
	defaultDB = db
}

func CloseDefaultDB() error {
	return CloseDB(defaultDB)
}

func MustGetDefaultDB() *gorm.DB {
	util.AssertOk(defaultDB != nil, `defaultDB为空`)
	return defaultDB
}

func MustInitDefaultDBFromDefaultCfgFile() *gorm.DB {
	ResetDefaultDB(MustNewDBFromDefaultCfgFile())
	return defaultDB
}

func MustNewDB(option *Option) *gorm.DB {
	option.MustNormalize()

	g := option.GORM
	g.Logger = newDBLogger(option.Log)

	db, err := gorm.Open(option.NewDialer(), option.GORM)
	util.AssertNilErr(err, `创建数据库对象出错`)

	if co := option.Conn; co != nil {
		rawDB, err := db.DB()
		util.AssertNilErr(err, `获取底层数据库对象出错`)

		rawDB.SetMaxIdleConns(co.MaxIdleConns)
		rawDB.SetMaxOpenConns(co.MaxOpenConns)
		rawDB.SetConnMaxIdleTime(co.ConnMaxIdleTime)
		rawDB.SetConnMaxLifetime(co.ConnMaxLifeTime)
	}

	return db
}

func MustNewDBFromCfgFile(fileName string) *gorm.DB {
	option := &Option{}
	conf.MustScanConfFile(option, fileName)
	return MustNewDB(option)
}

//读取默认配置文件db.toml创建数据库实例
func MustNewDBFromDefaultCfgFile() *gorm.DB {
	return MustNewDBFromCfgFile(`db`)
}

//关闭数据库
//gorm内部使用数据库连接池，仍然建议关闭数据库
//注意：查询返回的rows应遍历完或关闭，否则此查询将一直占用1个数据库连接，可能导致连接耗尽
func CloseDB(db *gorm.DB) error {
	if db == nil {
		return nil
	}

	rawDB, err := db.DB()
	if err != nil {
		return err
	}

	return rawDB.Close()
}

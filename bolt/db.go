package bolt

import (
	"github.com/bingooh/b-go-util/_string"
	"github.com/bingooh/b-go-util/util"
	bolt "go.etcd.io/bbolt"
	"os"
	"path/filepath"
	"time"
)

type Option struct {
	DbFilePath string
	bolt.Options
}

// MustNewDb 创建bolt.DB，使用完后应调用db.Close()关闭
// 1个数据库文件同时仅能由1个数据库实例访问(使用文件锁)，否则后续实例将等待锁或超时
// boltdb适用于读多写少，内部存储使用B+树。每秒大概可写入1000条记录
func MustNewDb(option *Option) *bolt.DB {
	util.AssertOk(option != nil, "option is nil")
	util.AssertOk(!_string.Empty(option.DbFilePath), "option.DbFilePath is empty")

	if option.Timeout <= 0 {
		option.Timeout = 1 * time.Minute //避免长时间等待获取文件锁
	}

	//必须创建好目录
	dir := filepath.Dir(option.DbFilePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		util.Panic(err, `can't create db file dir[%v]`, dir)
	}

	db, err := bolt.Open(option.DbFilePath, 0600, &option.Options)
	if err != nil {
		util.Panic(err, `bolt db open err`)
	}

	return db
}

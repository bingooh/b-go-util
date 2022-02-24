package bolt

import (
	"b-go-util/_string"
	"b-go-util/util"
	"fmt"
	bolt "go.etcd.io/bbolt"
	"os"
	"path/filepath"
	"time"
)

type Option struct {
	DbFilePath string
	Timeout    time.Duration
}

//创建bolt.DB，使用完后应调用db.Close()关闭
//1个数据库文件同时仅能由1个数据库实例访问(使用文件锁)，否则后续实例将等待锁或超时
func MustNewDb(option *Option) *bolt.DB {
	util.AssertOk(option != nil, "option is nil")
	util.AssertOk(!_string.Empty(option.DbFilePath), "option.DbFilePath is empty")

	if option.Timeout <= 0 {
		option.Timeout = 1 * time.Minute //避免长时间等待获取文件锁
	}

	boltOption := &bolt.Options{
		Timeout: option.Timeout,
	}

	//必须创建好目录
	dir := filepath.Dir(option.DbFilePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		panic(fmt.Errorf("can't create db file dir: %v->%w", dir, err))
	}

	db, err := bolt.Open(option.DbFilePath, 0600, boltOption)
	if err != nil {
		panic(fmt.Errorf("bolt db open err->%w", err))
	}

	return db
}

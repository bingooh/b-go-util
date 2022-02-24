package util

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

// 第1个参数必须是字符串
func sprintf(args ...interface{}) string {
	n := len(args)

	if n == 0 {
		return ""
	}

	format, ok := args[0].(string)
	if !ok {
		panic("args[0] is not string")
	}

	if n == 1 {
		return format
	}

	return fmt.Sprintf(format, args[1:]...)
}

// 参数args格式：err或format,args...或err,format,args...
func Sprintf(args ...interface{}) string {
	n := len(args)
	if n == 0 {
		return ""
	}

	cause, ok := args[0].(error)
	if !ok {
		return sprintf(args...)
	}

	if n == 1 {
		if cause != nil {
			return cause.Error()
		}

		return ""
	}

	msg := sprintf(args[1:]...)
	return fmt.Sprintf(`%v->%v`, msg, cause)
}

//输出到标准日志，参数格式同Sprintf
func Log(args ...interface{}) {
	log.Println(Sprintf(args...))
}

func AbsFilePath(fp string) (string, error) {
	sep := string(os.PathSeparator)
	return filepath.Abs(strings.ReplaceAll(fp, sep, "/"))
}

//如果返回false，则保证对应路径不存在。如果返回true，则不能保证真的存在
func IsFilePathExist(fp string) bool {
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		return false
	}

	//如果要判断文件是否真的存在，需要读写文件
	return true
}

//获取文件最近修改时间，如果操作系统禁用文件时间，则总是返回0
func GetFileLastModTime(fp string) time.Time {
	if info, err := os.Stat(fp); err == nil {
		return info.ModTime()
	}

	return time.Time{}
}

func ReadFile(filePath string) ([]byte, error) {
	bs, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("文件读取出错[path=%v,err=%w]", filePath, err)
	}

	return bs, nil
}

func ReadFileAsString(filePath string) (string, error) {
	bs, err := ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return string(bs), nil
}

func WriteFile(filePath string, content []byte) (string, error) {
	fp, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf(`[%v]转换为绝对文件路径出错:%w`, filePath, err)
	}

	if err := os.MkdirAll(filepath.Dir(fp), 0777); err != nil {
		return "", fmt.Errorf(`创建文件[%v]所在目录出错:%w`, filePath, err)
	}

	if err := ioutil.WriteFile(fp, content, 0666); err != nil {
		return "", fmt.Errorf(`写入文件出错:%w`, err)
	}

	return fp, nil
}

func WriteFileAsString(filePath, content string) (string, error) {
	return WriteFile(filePath, []byte(content))
}

func NewAtomicValue(v interface{}) *atomic.Value {
	val := new(atomic.Value)
	val.Store(v)
	return val
}

func OnExit(fn func(err error)) {
	var err error = nil
	if r := recover(); r != nil {
		if e, ok := r.(error); ok {
			err = e
		} else {
			err = fmt.Errorf("%v", r)
		}
	}

	fn(err)
}

// 此方法有点反模式，因为需要调用者关闭返回的管道
// 如果收到信号后结束进程，也不会有太多影响 signal.Stop(c);close(c)
func ListenQuitSignalNotify() chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT)

	return c
}

func ListenQuitSignal() <-chan struct{} {
	done := make(chan struct{})

	go func() {
		defer close(done)

		c := ListenQuitSignalNotify()
		<-c
		signal.Stop(c)
		close(c)
	}()

	return done
}

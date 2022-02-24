package conf

import (
	"fmt"
	"github.com/bingooh/b-go-util/_string"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const GO_APP_CFG_DIR = "GO_APP_CFG_DIR"

//获取配置文件目录，优先级dirs>conf.GetWorkingDir()/conf>$GO_APP_CFG_DIR>当前目录/conf>当前目录
//返回结果会添加$GO_APP_CFG_DIR,当前目录/conf,当前目录
func GetConfDirs(dirs ...string) []string {
	if dir, ok := GetWorkingDir(); ok {
		dirs = append(dirs, filepath.Join(dir, "conf"))
	}

	if dir := os.Getenv(GO_APP_CFG_DIR); !_string.Empty(dir) {
		dirs = append(dirs, dir)
	}

	return append(dirs, `./conf`, `.`)
}

//获取配置目录下已存在文件的绝对路径, filename可以为相对路径
//注意：Golang不能百分百判断文件是否真的存在，除非读取文件
func GetExistConfFilePath(filename string, dirs ...string) string {
	sep := string(os.PathSeparator)
	for _, dir := range GetConfDirs(dirs...) {
		fp, err := filepath.Abs(strings.ReplaceAll(dir+"/"+filename, sep, "/"))
		if err != nil {
			continue
		}

		//可以准确判断指定路径不存在，但不能判断是否真的存在
		if info, err := os.Stat(fp); os.IsNotExist(err) || info.IsDir() {
			continue
		}

		return fp
	}

	return ``
}

// 读取配置文件，filename不应该包含扩展名
func ReadConfFile(filename string, dirs ...string) (*viper.Viper, error) {
	v := viper.New()

	if filepath.IsAbs(filename) {
		dir := filepath.Dir(filename)
		dirs = append([]string{dir}, dirs...)
	}

	if ext := filepath.Ext(filename); ext != "" {
		name := filepath.Base(filename)
		filename = name[0 : len(name)-len(ext)]
	}

	v.SetConfigName(filename)

	for _, dir := range GetConfDirs(dirs...) {
		v.AddConfigPath(dir)
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	return v, nil
}

func ScanConfFile(dest interface{}, filename string, dirs ...string) error {
	v, err := ReadConfFile(filename, dirs...)
	if err != nil {
		return err
	}

	if err = v.Unmarshal(dest); err != nil {
		return err
	}

	return nil
}

func MustScanConfFile(dest interface{}, filename string, dirs ...string) {
	if err := ScanConfFile(dest, filename, dirs...); err != nil {
		log.Panicf("读取配置文件出错：%s", err)
	}
}

//读取文件内容，filename如为绝对路径则忽略dirs
func ReadFile(filename string, dirs ...string) ([]byte, error) {
	fp := filename
	if !filepath.IsAbs(fp) {
		fp = GetExistConfFilePath(filename, dirs...)
	}

	if bs, err := ioutil.ReadFile(fp); err != nil {
		return nil, fmt.Errorf(`读取文件出错: filepath=%v,err=%v`, fp, err)
	} else {
		return bs, nil
	}
}

func MustReadFile(filename string, dirs ...string) []byte {
	if bs, err := ReadFile(filename, dirs...); err != nil {
		panic(err)
	} else {
		return bs
	}
}

func MustReadFileAsString(filename string, dirs ...string) string {
	return string(MustReadFile(filename, dirs...))
}

//获取Debug配置，优先级:conf.IsDebugEnable()>env(环境变量)>cfg(配置文件)>dv(默认值)
//配置文件内容：Debug=true/false
func IsDebug(env, cfg string, dv bool) bool {
	if debug, ok := IsDebugEnable(); ok {
		return debug
	}

	if !_string.Empty(env) {
		if v := strings.ToLower(os.Getenv(env)); v != "" {
			if debug, err := strconv.ParseBool(v); err == nil {
				return debug
			}
		}
	}

	if !_string.Empty(cfg) {
		o := &struct {
			Debug bool
		}{dv}

		if err := ScanConfFile(o, cfg); err == nil {
			return o.Debug
		} else {
			fmt.Printf("读取Debug配置文件出错：%s\n", err)
		}
	}

	return dv
}

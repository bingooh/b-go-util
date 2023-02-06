package conf

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	BGO_CONF_ENV = `BGO_CONF_ENV`
	BGO_CONF_DIR = `BGO_CONF_DIR`
)

var Debug = false //启用将打印加载配置日志

// 匹配环境变量${name:default_value}
var envRx = regexp.MustCompile(`^\${([0-9A-Za-z_-]+)(:.*)*}$`)

func MustSetConfEnv(env string) {
	if err := os.Setenv(BGO_CONF_ENV, env); err != nil {
		log.Panic(err)
	}
}

func MustSetConfDir(dir string) {
	if err := os.Setenv(BGO_CONF_DIR, dir); err != nil {
		log.Panic(err)
	}
}

// Load 加载配置，参数dst必须为指针类型
// 参数file指定配置文件路径，不支持指定文件扩展名，优先级从上到下：
//   - 绝对路径或相对路径，如：./cfg/app
//   - 环境变量BGO_CONF_DIR的值
//   - 默认值：./conf
//
// 如果设置环境变量BGO_CONF_ENV，则文件名称会添加此值作为后缀
// 如：BGO_CONF_ENV=dev，file=app。则实际文件名称为app.dev
//
// 配置项值支持使用`$$`引用环境变量值，环境变量名称区分大小写
func Load(dst interface{}, file string) error {
	if dst == nil {
		return errors.New(`dst is nil`)
	}

	if strings.TrimSpace(file) == `` {
		return errors.New(`file path is empty`)
	}

	//viper不支持后缀名
	filename := filepath.Base(file)
	hasDirPath := filename != file
	if ext := filepath.Ext(filename); ext != `` {
		filename = filename[:len(filename)-len(ext)]
	}

	if v := os.Getenv(BGO_CONF_ENV); v != `` {
		filename += `.` + v
	}

	var dir string
	if hasDirPath {
		dir = filepath.Dir(file)
	} else if v := os.Getenv(BGO_CONF_DIR); v != `` {
		dir = v
	} else {
		dir = `./conf`
	}

	absDir, err := filepath.Abs(strings.ReplaceAll(dir, string(os.PathSeparator), "/"))
	if err != nil {
		return fmt.Errorf(`convert to abs path err[%v]`, dir)
	}

	v := viper.New()
	v.AddConfigPath(absDir)
	v.SetConfigName(filename)

	if err := v.ReadInConfig(); err != nil {
		return err
	}

	for key, val := range v.AllSettings() {
		bindConfEnv(v, key, val)
	}

	if err := v.Unmarshal(dst); err != nil {
		return err
	}

	if Debug {
		log.Printf("load cfg file[%v]\n", filepath.Join(absDir, filename))
	}

	return nil
}

func MustLoad(dst interface{}, file string) {
	if err := Load(dst, file); err != nil {
		log.Panicf("load cfg file err: %v\n", err)
	}
}

func parseEnv(v string) (envName, envDefaultValue string) {
	rs := envRx.FindStringSubmatch(v)
	if len(rs) != 3 {
		return
	}

	if len(rs[2]) == 0 {
		return rs[1], rs[2]
	}

	//去掉冒号
	return rs[1], rs[2][1:]
}

func bindConfEnv(vp *viper.Viper, key string, val interface{}) {
	switch v := val.(type) {
	case map[string]interface{}:
		for k1, v1 := range v {
			bindConfEnv(vp, key+`.`+k1, v1)
		}
	case string:
		envName, envDefaultValue := parseEnv(v)
		if envName == `` {
			return
		}

		//如果环境变量不存在，则设置为默认值
		//如果调用vp.SetDefault()，则配置项值仍然使用配置文件指定的值，以下直接设置
		if _, ok := os.LookupEnv(envName); !ok {
			vp.Set(key, envDefaultValue)
			return
		}

		if err := vp.BindEnv(key, envName); err != nil {
			log.Printf(`conf env variable bind fail[key=%v,env=%v]->%s`, key, envName, err)
		}
	}

}

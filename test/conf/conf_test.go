package conf

import (
	"github.com/bingooh/b-go-util/conf"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

type AppConf struct {
	Env   string `json:"env"`
	Age   int    `json:"age"`
	Tags  []int  `json:"tags"`
	Group struct {
		Name string `json:"name"`
	} `json:"group"`
}

func TestConf(t *testing.T) {
	r := require.New(t)

	conf.Debug = true //输出配置文件路径日志

	//配置文件：.conf/app
	c1 := &AppConf{}
	conf.MustLoad(c1, `app`)
	r.Equal(`default`, c1.Env)
	r.Equal(1, c1.Age)
	r.Equal(`g1`, c1.Group.Name)
	r.EqualValues([]int{1}, c1.Tags)

	//配置文件：.conf/app.dev
	c2 := &AppConf{}
	conf.MustSetConfEnv(`dev`)
	r.NoError(os.Setenv(`my_age`, `2`))
	r.NoError(os.Setenv(`my_tags`, `2,2`))
	conf.MustLoad(c2, `app`)
	r.Equal(`dev`, c2.Env)
	r.Equal(2, c2.Age)
	r.Equal(`g2`, c2.Group.Name)
	r.EqualValues([]int{2, 2}, c2.Tags)

	//配置文件：.conf/sub/app
	conf.MustSetConfEnv(``)
	conf.MustSetConfDir(`./conf/sub`)
	c3 := &AppConf{}
	conf.MustLoad(c3, `app`)
	r.Equal(`sub`, c3.Env)
	r.Equal(3, c3.Age)
	r.Equal(`g3`, c3.Group.Name)

	//直接指定文件路径
	c4 := &AppConf{}
	conf.MustLoad(c4, `./conf/app`)
	r.EqualValues(c1, c4)

	wd, err := os.Getwd()
	r.NoError(err)
	c5 := &AppConf{}
	conf.MustLoad(c5, filepath.Join(wd, `./conf/sub/app`)) //绝对路径
	r.EqualValues(c3, c5)
}

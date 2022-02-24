package util

import (
	"b-go-util/util"
	"fmt"
	"github.com/stretchr/testify/require"
	"strconv"
	"testing"
	"time"
)

func TestToken(t *testing.T) {
	r := require.New(t)

	key := `abc`
	h1 := 1
	h2 := `2.3`
	ts := time.Now()
	token := util.NewToken(key, ts, h1, h2)
	r.Error(util.CheckToken(`111`, token, 1*time.Minute))
	r.NoError(util.CheckToken(key, token, 1*time.Minute))
	fmt.Println(token)

	headers, ts1, _, err := util.ParseToken(token)
	r.NoError(err)
	r.Equal(ts.Unix(), ts1.Unix())
	r.EqualValues(strconv.Itoa(h1), headers[0]) //解析结果为字符串
	r.EqualValues(h2, headers[1])

	//等待3秒，然后校验时设置tokenTTL为2秒，校验结果为token超时失效
	time.Sleep(3 * time.Second)
	r.Error(util.CheckToken(key, token, 2*time.Second))
	r.NoError(util.CheckToken(key, token, 0)) //tokenTTL设置为0则不检查是否过期
}

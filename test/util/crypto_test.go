package util

import (
	"encoding/hex"
	"fmt"
	"github.com/bingooh/b-go-util/util"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func TestToken(t *testing.T) {
	r := require.New(t)

	key := `abc`
	h1 := `1`
	h2 := `2.3`
	ts := time.Now()

	t1 := util.NewToken(h1, h2)
	t1.CreatedAt = ts

	token1 := t1.MustEncode(key)
	r.Error(util.CheckToken(`111`, token1, 1*time.Minute))
	r.NoError(util.CheckToken(key, token1, 1*time.Minute))
	fmt.Println(token1)

	t2, err := util.ParseToken(token1)
	r.NoError(err)
	r.True(strings.HasSuffix(token1, t2.Sign))
	r.Equal(ts.Unix(), t2.CreatedAt.Unix())
	r.EqualValues(h1, t2.Headers[0])
	r.EqualValues(h2, t2.Headers[1])
	r.NoError(util.CheckToken(key, t2.MustEncode(key), 1*time.Minute))
	fmt.Println(t2.Val()) //未编码

	//等待3秒，然后校验时设置tokenTTL为2秒，校验结果为token超时失效
	time.Sleep(3 * time.Second)
	r.Error(util.CheckToken(key, token1, 2*time.Second))
	r.NoError(util.CheckToken(key, token1, 0)) //tokenTTL设置为0则不检查是否过期
}

func TestAesCbc(t *testing.T) {
	r := require.New(t)

	key := []byte(`1111111111111111`)
	iv := []byte(`2222222222222222`)
	plain := []byte(`hello`)
	expect := `f5e78b5be17acba949ccdd92b5c875ee`
	expectBytes, err := hex.DecodeString(expect)
	r.NoError(err)

	rs1, err := util.EncryptAesCbc(plain, key, iv)
	r.NoError(err)
	r.Equal(rs1, expectBytes)

	rs2, err := util.DecryptAesCbc(rs1, key, iv)
	r.NoError(err)
	r.Equal(plain, rs2)

	h := util.NewAesCbcHelper(string(key))
	rs3, err := h.EncryptToHex(string(plain))

	rs4, err := h.DecryptFromHex(rs3)
	r.NoError(err)
	r.Equal(string(plain), rs4)
}

func TestAesGcm(t *testing.T) {
	r := require.New(t)

	key := []byte(`1111111111111111`)
	nonce := []byte(`333333333333`)
	plain := []byte(`hello`)
	expect := `6d1c7a32654e75dab7de8ee4a8caf486faa8fda060`

	rs1, err := util.EncryptAesGcm(plain, key, nonce)
	r.NoError(err)
	r.Equal(expect, hex.EncodeToString(rs1))

	rs2, err := util.DecryptAesGcm(rs1, key, nonce)
	r.NoError(err)
	r.Equal(plain, rs2)

	h := util.NewAesGcmHelper(string(key))
	rs3, err := h.EncryptToHex(string(plain))

	rs4, err := h.DecryptFromHex(rs3)
	r.NoError(err)
	r.Equal(string(plain), rs4)
}

package util

import (
	srand "crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"log"
	"math/rand"
	"strings"
	"time"
)

const (
	NUM       = "0123456789"
	ALPHA_NUM = "0123456789abcdefghijklmnopqrstuvwxyz"
)

func NewRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

// 取值范围[min,max)
func RandInt(min, max int) int {
	return NewRand().Intn(max-min) + min
}

// 取值范围[min,max)
func RandInt64(min, max int64) int64 {
	return NewRand().Int63n(max-min) + min
}

func RandDuration(min, max int) time.Duration {
	return time.Duration(RandInt(min, max))
}

func RandNum(size int) string {
	return randBy(NUM, size)
}

func RandAlphaNum(size int) string {
	return randBy(ALPHA_NUM, size)
}

func RandSecure(size int) string {
	bs := make([]byte, size)
	if _, err := srand.Read(bs); err != nil {
		log.Println(err, "RandSecure err, will use insecure rand")
		return RandAlphaNum(size)
	}

	return string(bs)
}

func RandSecureHex(size int) string {
	return hex.EncodeToString([]byte(RandSecure(size / 2)))
}

func RandSecureBase64(size int) string {
	s := base64.StdEncoding.EncodeToString([]byte(RandSecure(size)))
	if len(s) >= size {
		return s[:size]
	}

	return s + RandAlphaNum(size-len(s))
}

func randBy(seed string, size int) string {
	if n := len(seed); n < size {
		seed = strings.Repeat(seed, size/n) + seed[:size%n]
	}

	src := []byte(seed)
	NewRand().Shuffle(len(seed), func(i, j int) {
		src[i], src[j] = src[j], src[i]
	})

	return string(src[:size])
}

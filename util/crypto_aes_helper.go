package util

import (
	"crypto/aes"
	"encoding/hex"
	"fmt"
)

type aesCipher interface {
	Encrypt(key, plain []byte) (rs []byte, err error)
	Decrypt(key, encrypted []byte) (rs []byte, err error)
}

type aesCbcCipher struct{}

// Encrypt 加密结果格式为：16位随机IV+密文
func (c *aesCbcCipher) Encrypt(key, plain []byte) (rs []byte, err error) {
	iv := []byte(RandSecure(aes.BlockSize))

	rs, err = EncryptAesCbc(plain, key, iv)
	if err == nil {
		rs = append(iv, rs...)
	}

	return
}

func (c *aesCbcCipher) Decrypt(key, encrypted []byte) (rs []byte, err error) {
	n := len(encrypted)
	if n < aes.BlockSize {
		return nil, fmt.Errorf(`invalid encrypted size[expect>=%v,actual=%v]`, aes.BlockSize, n)
	}

	if n == aes.BlockSize {
		return nil, nil
	}

	iv := encrypted[:aes.BlockSize]
	encrypted = encrypted[aes.BlockSize:]
	return DecryptAesCbc(encrypted, key, iv)
}

type aesGcmCipher struct{}

// 加密结果格式为：12位随机nonce+密文
func (c *aesGcmCipher) Encrypt(key, plain []byte) (rs []byte, err error) {
	nonce := []byte(RandSecure(gcmNonceSize))

	rs, err = EncryptAesGcm(plain, key, nonce)
	if err == nil {
		rs = append(nonce, rs...)
	}

	return
}

func (c *aesGcmCipher) Decrypt(key, encrypted []byte) (rs []byte, err error) {
	n := len(encrypted)
	if n < gcmNonceSize {
		return nil, fmt.Errorf(`invalid encrypted size[expect>=%v,actual=%v]`, gcmNonceSize, n)
	}

	if n == gcmNonceSize {
		return nil, nil
	}

	nonce := encrypted[:gcmNonceSize]
	encrypted = encrypted[gcmNonceSize:]
	return DecryptAesGcm(encrypted, key, nonce)
}

type AesHelper struct {
	key    []byte //密钥
	cipher aesCipher
}

func newHelper(key string, cipher aesCipher) *AesHelper {
	return &AesHelper{key: []byte(key), cipher: cipher}
}

func NewAesCbcHelper(key string) *AesHelper {
	return newHelper(key, &aesCbcCipher{})
}

func NewAesGcmHelper(key string) *AesHelper {
	return newHelper(key, &aesGcmCipher{})
}

func (h *AesHelper) SetKey(key string) {
	h.key = []byte(key)
}

// Encrypt 加密结果格式为：16位随机IV+密文
func (h *AesHelper) Encrypt(plain []byte) (rs []byte, err error) {
	return h.cipher.Encrypt(h.key, plain)
}

// Decrypt 解密，encrypted格式为：16位随机IV+密文
func (h *AesHelper) Decrypt(encrypted []byte) (plain []byte, err error) {
	return h.cipher.Decrypt(h.key, encrypted)
}

func (h *AesHelper) EncryptString(plain string) (string, error) {
	rs, err := h.Encrypt([]byte(plain))
	return string(rs), err
}

func (h *AesHelper) DecryptString(encrypted string) (string, error) {
	rs, err := h.Decrypt([]byte(encrypted))
	return string(rs), err
}

func (h *AesHelper) EncryptToHex(plain string) (string, error) {
	if rs, err := h.Encrypt([]byte(plain)); err != nil {
		return ``, err
	} else {
		return hex.EncodeToString(rs), nil
	}
}

func (h *AesHelper) DecryptFromHex(encrypted string) (string, error) {
	if data, err := hex.DecodeString(encrypted); err != nil {
		return ``, err
	} else {
		return h.DecryptString(string(data))
	}
}

package util

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
)

const (
	gcmTagSize   = 16
	gcmNonceSize = 12
)

func Pkcs7Padding(data []byte, blockSize int) []byte {
	n := len(data)
	padding := blockSize - n%blockSize
	padded := bytes.Repeat([]byte{byte(padding)}, padding)

	rs := make([]byte, n+padding)
	copy(rs[:n], data)
	copy(rs[n:], padded)
	return rs
}

func Pkcs7UnPadding(data []byte) []byte {
	if n := len(data); n > 0 {
		padding := int(data[n-1])
		return data[:(n - padding)]
	}

	return data
}

// EncryptAesCbc key长度必须是16,24,32，分别对应aes128, aes192, aes256，iv长度必须为16
func EncryptAesCbc(plain, key, iv []byte) (rs []byte, err error) {
	defer OnExit(func(e error) {
		if e != nil {
			rs, err = nil, e
		}
	})

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	//相同的密钥和明文+不同的IV可产生不同的密文
	//初始向量IV长度必须与blockSize相同，不同长度的key对应的blockSize都为aes.BlockSize
	blockSize := block.BlockSize()
	if len(iv) != blockSize {
		return nil, fmt.Errorf("invalid iv length[expect=%v,actual=%v]", blockSize, len(iv))
	}

	//填充明文，其长度必须是blockSize的倍数
	plain = Pkcs7Padding(plain, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, iv)
	blockMode.CryptBlocks(plain, plain) //此方法可能panic
	return plain, nil
}

func DecryptAesCbc(encrypted, key, iv []byte) (rs []byte, err error) {
	defer OnExit(func(e error) {
		if e != nil {
			rs, err = nil, e
		}
	})

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	if len(iv) != blockSize {
		return nil, fmt.Errorf("invalid iv length[expect=%v,actual=%v]", blockSize, len(iv))
	}

	//密文长度必须是blockSize的倍数
	if n := len(encrypted); n < blockSize || n%blockSize != 0 {
		return nil, fmt.Errorf("invalid encrypted length[expect=%vx,actual=%v]", blockSize, n)
	}

	blockMode := cipher.NewCBCDecrypter(block, iv)
	blockMode.CryptBlocks(encrypted, encrypted) //此方法可能panic
	return Pkcs7UnPadding(encrypted), nil
}

// EncryptAesGcm key长度必须是16,24,32，分别对应aes128, aes192, aes256。
// NonceSize为12，AuthTagSize为16。所以nonce长度必须为12，rs最后16字节为AuthTag
// 其他语言加密结果可能不包含AuthTag，或者默认AuthTagSize不等于16
func EncryptAesGcm(plain, key, nonce []byte) (rs []byte, err error) {
	defer OnExit(func(e error) {
		if e != nil {
			rs, err = nil, e
		}
	})

	if len(nonce) != gcmNonceSize {
		return nil, fmt.Errorf(`invalid nonce size[expect=%v,actual=%v]`, gcmNonceSize, len(nonce))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	//可以调用其他方法指定AuthTagSize,NonceSize
	//head.NonceSize()可获取当前的NonceSize
	head, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	//设置additionalData不会影响加密结果
	return head.Seal(nil, nonce, plain, nil), nil
}

// DecryptAesGcm nonce长度必须为12，encrypted最后16位必须为AuthTag
func DecryptAesGcm(encrypted, key, nonce []byte) (rs []byte, err error) {
	defer OnExit(func(e error) {
		if e != nil {
			rs, err = nil, e
		}
	})

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	head, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return head.Open(nil, nonce, encrypted, nil)
}

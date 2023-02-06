package util

import "golang.org/x/text/encoding/simplifiedchinese"

func DecodeGBK(v string) (string, error) {
	return simplifiedchinese.GBK.NewDecoder().String(v)
}

func DecodeGBKBytes(v []byte) ([]byte, error) {
	return simplifiedchinese.GBK.NewDecoder().Bytes(v)
}

func DecodeGB18030(v string) (string, error) {
	return simplifiedchinese.GB18030.NewDecoder().String(v)
}

func DecodeGB18030Bytes(v []byte) ([]byte, error) {
	return simplifiedchinese.GB18030.NewDecoder().Bytes(v)
}

func EncodeGBK(v string) (string, error) {
	return simplifiedchinese.GBK.NewEncoder().String(v)
}

func EncodeGBKBytes(v []byte) ([]byte, error) {
	return simplifiedchinese.GBK.NewEncoder().Bytes(v)
}

func EncodeGB18030(v string) (string, error) {
	return simplifiedchinese.GB18030.NewEncoder().String(v)
}

func EncodeGB18030Bytes(v []byte) ([]byte, error) {
	return simplifiedchinese.GB18030.NewEncoder().Bytes(v)
}

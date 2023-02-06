package util

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AbsFilePath 获取文件绝对路径
// DEPRECATED: filepath.Abs instead.
func AbsFilePath(fp string) (string, error) {
	sep := string(os.PathSeparator)
	return filepath.Abs(strings.ReplaceAll(fp, sep, "/"))
}

// IsFilePathExist 如果返回false，则保证对应路径不存在。如果返回true，则不能保证真的存在
func IsFilePathExist(fp string) bool {
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		return false
	}

	//如果要判断文件是否真的存在，需要读写文件
	return true
}

// GetFileLastModTime 获取文件最近修改时间，如果操作系统禁用文件修改时间，则总是返回0
func GetFileLastModTime(fp string) time.Time {
	if info, err := os.Stat(fp); err == nil {
		return info.ModTime()
	}

	return time.Time{}
}

// MustMkdirAll 创建目录并返回目录绝对路径
func MustMkdirAll(dirPath string, subDirs ...string) string {
	if len(subDirs) > 0 {
		dirPath = filepath.Join(append([]string{dirPath}, subDirs...)...)
	}

	fp, err := filepath.Abs(dirPath)
	AssertNilErr(err, `目录绝对路径获取出错[dir=%v]`, dirPath)

	AssertNilErr(os.MkdirAll(fp, os.ModePerm), `目录创建出错[dir=%v]`, fp)
	return fp
}

func ReadFile(filePath string) ([]byte, error) {
	bs, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("文件读取出错[path=%v]->%w", filePath, err)
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

func ReadFileAsStringLines(filepath string) ([]string, error) {
	rs, err := ReadFileAsString(filepath)
	if err != nil {
		return nil, err
	}

	items := strings.Split(rs, "\n")
	if i := len(items) - 1; i >= 0 && items[i] == "" {
		return items[:i], nil
	}

	return items, nil
}

func WriteFile(filePath string, content []byte) (string, error) {
	fp, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf(`文件绝对路径获取出错[path=%v]->%w`, filePath, err)
	}

	if err := os.MkdirAll(filepath.Dir(fp), os.ModePerm); err != nil {
		return "", fmt.Errorf(`文件所在目录创建出错[path=%v]->%w`, fp, err)
	}

	if err := ioutil.WriteFile(fp, content, 0644); err != nil {
		return "", fmt.Errorf(`文件写入出错[path=%v]->%w`, fp, err)
	}

	return fp, nil
}

func WriteFileAsString(filePath, content string) (string, error) {
	return WriteFile(filePath, []byte(content))
}

func OpenFile(filePath string, flag int) (*os.File, error) {
	fp, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf(`文件绝对路径获取出错[path=%v]->%w`, filePath, err)
	}

	return os.OpenFile(fp, os.O_CREATE|flag, 0644)
}

func MustOpenFile(filePath string, flag int) *os.File {
	file, err := OpenFile(filePath, flag)
	AssertNilErr(err, `文件打开出错`)
	return file
}

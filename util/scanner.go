package util

import (
	"bufio"
	"bytes"
	"io"
	"os"
)

func NewSplitFn(sep byte) bufio.SplitFunc {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF {
			if len(data) > 0 {
				//确保能取到分隔符后的字符
				return 0, data, bufio.ErrFinalToken
			}

			return 0, nil, io.EOF
		}

		if i := bytes.IndexByte(data, sep); i > -1 {
			return i + 1, data[:i], nil
		}

		return 0, nil, nil
	}
}

func NewSplitScanner(r io.Reader, sep byte) *bufio.Scanner {
	s := bufio.NewScanner(r)
	s.Split(NewSplitFn(sep))

	return s
}

func NewFileLineScanner(fpath string) (*bufio.Scanner, *os.File, error) {
	file, err := os.Open(fpath)
	if err != nil {
		return nil, nil, err
	}

	return bufio.NewScanner(file), file, nil
}

func ForEachFileLine(fpath string, fn func(i int, line string) bool) (err error) {
	scanner, file, err := NewFileLineScanner(fpath)
	if err != nil {
		return err
	}

	defer func() {
		if e := file.Close(); e != nil {
			Log(e, `file close err[path=%v]`, fpath)
		}
	}()

	i := -1
	for scanner.Scan() {
		i++
		if !fn(i, scanner.Text()) {
			return
		}
	}

	return scanner.Err()
}

// ForEachFileLineBytes 遍历每行数据
// 注意：回调函数传入的line将在遍历下1条记录时被覆盖，如需复用需先复制
func ForEachFileLineBytes(fpath string, fn func(i int, line []byte) bool) (err error) {
	scanner, file, err := NewFileLineScanner(fpath)
	if err != nil {
		return err
	}

	defer func() {
		if e := file.Close(); e != nil {
			Log(e, `file close err[path=%v]`, fpath)
		}
	}()

	i := -1
	for scanner.Scan() {
		i++
		if !fn(i, scanner.Bytes()) {
			return
		}
	}

	return scanner.Err()
}

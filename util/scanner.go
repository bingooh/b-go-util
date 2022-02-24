package util

import (
	"bufio"
	"bytes"
	"io"
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

package util

import (
	"fmt"
	"log"
)

// CopyBytes 复制字节，参数dst可选，如果传入可用作缓存
func CopyBytes(src, dst []byte) []byte {
	if len(src) == 0 {
		return src
	}
	return append(dst[:0], src...)
}

func toErr(v interface{}) error {
	if v == nil {
		return nil
	}

	if e, ok := v.(error); ok {
		return e
	}

	return fmt.Errorf(`%v`, v)
}

func OnExit(fn func(err error)) {
	fn(toErr(recover()))
}

func OnPanic(fn func(err error)) {
	if err := toErr(recover()); err != nil {
		fn(err)
	}
}

func Recover() {
	if r := recover(); r != nil {
		log.Println(`recover:`, r)
	}
}

func Panic(args ...interface{}) {
	if len(args) == 1 {
		if err, ok := args[0].(error); ok {
			panic(err)
		}
	}

	panic(Sprintf(args...))
}

func DoBatch(size, step int, fn func(i, start, end int) error) error {
	if size <= 0 || step <= 0 {
		return nil
	}

	if step >= size {
		return fn(0, 0, size)
	}

	for i, start := 0, 0; start < size; start += step {
		end := start + step
		if end > size {
			end = size
		}

		if err := fn(i, start, end); err != nil {
			return err
		}

		i++
	}

	return nil
}

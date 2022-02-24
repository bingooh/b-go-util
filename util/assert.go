package util

import "b-go-util/_string"

func AssertNilErr(err error, args ...interface{}) {
	if err == nil {
		return
	}

	if len(args) == 0 {
		panic(err)
	}

	args = append([]interface{}{err}, args...)
	panic(NewAssertFailError(args...))
}

func AssertOk(ok bool, args ...interface{}) {
	if !ok {
		panic(NewAssertFailError(args...))
	}
}

func AssertNotEmpty(val string, args ...interface{}) {
	AssertOk(!_string.Empty(val), args...)
}

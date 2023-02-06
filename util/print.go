package util

import (
	"errors"
	"fmt"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"log"
)

var EnglishPrinter = NewEnglishPrinter()

// NewEnglishPrinter 可格式化显示千分位
func NewEnglishPrinter() *message.Printer {
	return message.NewPrinter(language.English)
}

// 第1个参数必须是字符串
func sprintf(args ...interface{}) string {
	n := len(args)

	if n == 0 {
		return ""
	}

	format, ok := args[0].(string)
	if !ok {
		panic("args[0] is not string")
	}

	if n == 1 {
		return format
	}

	return fmt.Sprintf(format, args[1:]...)
}

// Sprintf 参数args格式：err或format,args...或err,format,args...
func Sprintf(args ...interface{}) string {
	n := len(args)
	if n == 0 {
		return ""
	}

	cause, ok := args[0].(error)
	if !ok {
		return sprintf(args...)
	}

	if n == 1 {
		if cause != nil {
			return cause.Error()
		}

		return ""
	}

	msg := sprintf(args[1:]...)
	return fmt.Sprintf(`%v->%v`, msg, cause)
}

func Errorf(args ...interface{}) error {
	if n := len(args); n > 0 {
		if cause, ok := args[0].(error); ok {
			if n == 1 {
				return cause
			}

			return fmt.Errorf(`%v->%w`, sprintf(args[1:]...), cause)
		}

	}

	return errors.New(sprintf(args...))
}

// Log 输出到标准日志，参数格式同Sprintf
func Log(args ...interface{}) {
	log.Println(Sprintf(args...))
}

package util

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

var quitSignals = []os.Signal{syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT}

func ListenQuitSignal() <-chan os.Signal {
	c := make(chan os.Signal, 1)
	signal.Notify(c, quitSignals...)

	go func() {
		defer close(c)

		<-c
		signal.Stop(c)
	}()

	return c
}

// ListenQuitSignalFromStdIn 监听来自stdin的退出信号
// 通过stdio传递退出信号，发送退出信号参考 WriteSignal()
// windows进程不支持退出信号,以下方案适用于启动windows子进程
func ListenQuitSignalFromStdIn() <-chan os.Signal {
	c := make(chan os.Signal, 1)

	go func() {
		defer close(c)

		for {
			sig := ``
			_, err := fmt.Scanln(&sig)
			if err == io.EOF {
				//是否应该写入1个signal
				return
			}
			if err != nil {
				Log(err, `read signal from stdin err`)
				continue
			}

			v, err := strconv.Atoi(sig)
			if err != nil {
				Log(err, `parse signal from stdin err`)
				continue
			}

			s := syscall.Signal(v)
			for _, qs := range quitSignals {
				if s == qs {
					c <- s
					return
				}
			}
		}
	}()

	return c
}

func WriteSignal(w io.Writer, sig syscall.Signal) error {
	_, err := fmt.Fprintln(w, strconv.Itoa(int(sig))) //1个signal占1行
	return err
}

func WriteQuitSignal(w io.Writer) error {
	return WriteSignal(w, syscall.SIGTERM)
}

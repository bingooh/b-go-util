package async

import (
	"context"
	"github.com/bingooh/b-go-util/util"
)

// DoAll 串行执行直到遇到第1个失败任务，返回第1个错误
func DoAll(fns ...func() error) error {
	for _, fn := range fns {
		if err := fn(); err != nil {
			return err
		}
	}

	return nil
}

// DoCancelableAll 串行执行直到遇到第1个失败任务或ctx取消，返回第1个错误
func DoCancelableAll(ctx context.Context, fns ...func() error) error {
	for _, fn := range fns {
		rs := <-RunCancelableTask(ctx, ToErrTask(fn))
		if err := rs.Error(); err != nil {
			return err
		}
	}

	return nil
}

// DoAny 串行执行直到遇到第1个成功任务，如果全部失败返回最后1个错误
func DoAny(fns ...func() error) error {
	var cause error
	for _, fn := range fns {
		if cause = fn(); cause == nil {
			return nil
		}
	}

	return cause
}

// DoCancelableAny 串行执行直到遇到第1个成功任务或ctx取消，如果全部失败返回最后1个错误
func DoCancelableAny(ctx context.Context, fns ...func() error) error {
	var cause error

	for _, fn := range fns {
		rs := <-RunCancelableTask(ctx, ToErrTask(fn))
		if rs.Error() == nil {
			return nil
		}

		cause = rs.Error()
	}

	return cause
}

// RunAll 并行执行直到遇到第1个失败任务，返回第1个错误
func RunAll(fns ...func() error) error {
	return RunCancelableAll(context.Background(), fns...)
}

// RunCancelableAll 并行执行直到遇到第1个失败任务或ctx取消，返回第1个错误
func RunCancelableAll(ctx context.Context, fns ...func() error) error {
	cx, cancel := context.WithCancel(ctx)
	defer cancel()

	cause := util.NewAtomicError()
	g := NewWaitGroup()
	for _, fn := range fns {
		if cause.Value() != nil {
			break
		}

		fn := fn
		g.Run(func() {
			if cause.Value() != nil {
				return
			}

			rs := <-RunCancelableTask(cx, ToErrTask(fn))
			if err := rs.Error(); err != nil {
				if cause.SetIfAbsent(err) {
					cancel()
				}
			}
		})
	}
	g.Wait()

	return cause.Value()
}

// RunAny 并行执行直到遇到第1个成功任务，如果全部失败返回最后1个错误
func RunAny(fns ...func() error) error {
	return RunCancelableAny(context.Background(), fns...)
}

// RunCancelableAny 并行执行直到遇到第1个成功任务或ctx取消，如果全部失败返回最后1个错误
func RunCancelableAny(ctx context.Context, fns ...func() error) error {
	cx, cancel := context.WithCancel(ctx)
	defer cancel()

	cause := util.NewAtomicError()
	isOk := util.NewAtomicBool(false)

	g := NewWaitGroup()
	for _, fn := range fns {
		if isOk.True() {
			break
		}

		fn := fn
		g.Run(func() {
			if isOk.True() {
				return
			}

			rs := <-RunCancelableTask(cx, ToErrTask(fn))
			if isOk.True() {
				return
			}

			if err := rs.Error(); err != nil {
				cause.Set(err)
				return
			}

			if isOk.CASwap(false) {
				cancel()
			}
		})
	}
	g.Wait()

	if isOk.True() {
		return nil
	}

	return cause.Value()
}

package async

//任务接口
type Task interface {
	Run() Result
}

// 任务函数，实现了Task接口
type TaskFn func() Result
type VoidTaskFn func()
type ErrTaskFn func() error
type ValTaskFn func() (interface{}, error)

func (f TaskFn) Run() Result {
	return f()
}

func (f VoidTaskFn) Run() Result {
	f()
	return NewResult(nil, nil)
}

func (f ErrTaskFn) Run() Result {
	return NewResult(nil, f())
}

func (f ValTaskFn) Run() Result {
	return NewResult(f())
}

func ToTask(fn TaskFn) Task {
	return fn
}

func ToVoidTask(fn func()) Task {
	return VoidTaskFn(fn)
}

func ToErrTask(fn func() error) Task {
	return ErrTaskFn(fn)
}

func ToValTask(fn func() (interface{}, error)) Task {
	return ValTaskFn(fn)
}

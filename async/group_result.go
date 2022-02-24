package async

import "sync"

//任务组执行结果
type GroupResult struct {
	lock sync.Mutex

	err          error
	firstOkIdx   int
	firstDoneIdx int
	resultMap    ResultMap

	canceled bool
	timeout  bool
}

func newGroupResult() *GroupResult {
	return &GroupResult{firstOkIdx: -1, firstDoneIdx: -1, resultMap: make(ResultMap)}
}

//保存任务组等待超时/取消结果
func (r *GroupResult) cancel(c Context) {
	if !c.Done() || r.timeout || r.canceled {
		return
	}

	r.lock.Lock()
	defer r.lock.Unlock()

	if r.err == nil {
		r.err = c.Error()
	}
	r.timeout = c.Timeout()
	r.canceled = c.Canceled()
}

//保存任务执行结果
func (r *GroupResult) put(i int, result Result) {
	r.lock.Lock()
	defer r.lock.Unlock()

	//如果group等待超时则不再接受新的执行结果，但保留已有结果
	if r.timeout || r.canceled {
		return
	}

	if r.firstDoneIdx == -1 {
		r.firstDoneIdx = i
	}

	hasErr := result.HasError()
	if r.firstOkIdx == -1 && !hasErr {
		r.firstOkIdx = i
	}

	if r.err == nil && hasErr {
		r.err = result.Error()
	}

	r.resultMap[i] = result
}

//第1个失败任务返回的错误
func (r *GroupResult) Error() error {
	return r.err
}

func (r *GroupResult) HasError() bool {
	return r.err != nil
}

//等待任务组执行结果是否取消
func (r *GroupResult) Canceled() bool {
	return r.canceled
}

//等待任务组执行结果是否超时
func (r *GroupResult) Timeout() bool {
	return r.timeout
}

//第1个成功完成任务的索引值
func (r *GroupResult) FirstOkIdx() int {
	return r.firstOkIdx
}

//第1个完成任务(不管是否成功)的索引值
func (r *GroupResult) FirstDoneIdx() int {
	return r.firstDoneIdx
}

//第1个成功完成任务的结果
func (r *GroupResult) FirstOk() Result {
	if r.firstOkIdx == -1 {
		return nil
	}

	return r.resultMap[r.firstOkIdx]
}

//第1个完成任务(不管是否成功)的结果
func (r *GroupResult) FirstDone() Result {
	if r.firstDoneIdx == -1 {
		return nil
	}

	return r.resultMap[r.firstDoneIdx]
}

//获取索引值对应的任务执行结果，如果无返回nil
func (r *GroupResult) Get(idx int) Result {
	return r.resultMap[idx]
}

//任务结果总数
func (r *GroupResult) Size() int {
	return len(r.resultMap)
}

//任务执行结果Map，key为任务索引值，即任务添加时获取的序列号
func (r *GroupResult) ResultMap() ResultMap {
	m := make(ResultMap, len(r.resultMap))
	for k, result := range r.resultMap {
		m[k] = result
	}

	return m
}

//任务执行结果List，按照任务索引值，即添加顺序排序
func (r *GroupResult) ResultList() ResultList {
	list := make(ResultList, len(r.resultMap))

	for i, r := range r.resultMap {
		list[i] = r
	}

	return list
}

package async

import (
	"fmt"
	"github.com/bingooh/b-go-util/async"
	"time"
)

//延时任务，沉睡i秒后任务结束
func job(i int) {
	fmt.Printf("task %v start\n", i)
	time.Sleep(time.Duration(i) * time.Second)
	fmt.Printf("task %v done\n", i)
}

//创建延时任务，沉睡i秒后任务结束
func newJob(i int) func() {
	return func() {
		job(i)
	}
}

//创建延时任务，沉睡i秒后任务结束
func newTask(i int) async.Task {
	return async.ToValTask(func() (v interface{}, e error) {
		job(i)
		return i, nil
	})
}

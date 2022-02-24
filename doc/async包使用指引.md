### 主要方法和类型
- `DoXX(task func())`       在当前协程执行任务
- `RunXX(task func())`      在后台协程执行任务
- `RunXXTask(task Task)`    在后台协程执行任务
- `RunIntervalHelper`       定时任务执行帮助类
- `Runner`                  后台任务执行帮助类
- `Pool`                    协程池，限制执行任务可使用的协程数
- `Group`                   任务组，执行一组任务并保存其执行结果
- `Task/Context`            任务接口和任务执行上下文
- `Result/GroupResult`      任务和任务组执行结果

### 限制和缺陷
- `Pool/Group`
    - 支持：  当前协程添加任务->等待任务执行结果
    - 不支持：协程1添加任务->协程2等待任务执行结果
    - 不支持：当前任务执行时，向当前`Pool/Group`里添加新的任务
    
### Task/Result/Context
- 任务定义包括(后续说明不再做区分)：
    - `task fun()`   不需返回结果的任务，可作为`DoXX()/RunXX()`参数
    - `Task/TaskFn`  需要返回结果的任务，可作为`RunXXTask()`参数
- `Task/TaskFn` 定义需返回结果的任务接口和任务函数
    - 更多将普通函数转换为`Task`的帮助方法见源码    
- `Result` 保存任务执行结果
    - `Value()/Error()`       任务返回的值和错误
    - `Canceled()/Timeout()`  任务是否取消或超时，一般根据任务返回的错误类型判断
    - `Int()/MustInt()....`   将任务返回的值转换为对应的数据类型，直接使用Go的类型转换
- `Context` 定义任务执行上下文，封装`context.Context`,提供更多帮助方法
    - `Done()/Canceled()/Timeout()` 任务是否完成，取消，超时
    - `Abort()/Aborted()`           主动取消任务/任务是否主动取消
    - `Error()`                     任务返回的错误
    - `Count()`                     任务循环执行次数

```go
	//执行不需返回结果的任务
	async.Run(func() {
		fmt.Println(`done`)
	})

	//执行需要返回结果的任务
	result:=<-async.RunTask(async.TaskFn(func() async.Result {
		return async.NewResult("1",nil)
	}))

	require.NoError(t,result.Error())
	fmt.Println(result.Error()) // nil
	fmt.Println(result.Value()) // "1"

	//result.Value()返回的数据类型是interface{}
	//以下调用result.String()将结果转换为字符串
	s,err:=result.String()
	fmt.Println(err) // nil
	fmt.Println(s)   // "1"

	//result.Value()的值为字符串"1"，不能转换为int
	//转换类型时仅使用v.(int)
	_,err=result.Int()
	fmt.Println(err) // type cast err

    //任务循环执行3次后主动取消
    ctx:=context.Background()
	<-async.RunUtilCancel(ctx, func(c async.Context) {
        //如果是ctx取消，将执行以下代码，即c.Done()==true
        //如果调用c.Abort()主动取消，会立刻终止循环，不会执行以下代码
		if c.Done() {
			fmt.Println("task done")
			return
		}

		//c.Count()从1开始计数
		fmt.Println("task do: ", c.Count())

		if c.Count() == 3 {
			c.Abort()
		}
	})

```

### `DoXX()/RunXX()/RunXXTask()`
- `DoXX(task func())`     
    - 在当前协程执行任务，阻塞当前协程直到任务执行结束
    - `DoUntilCancel()`    循环执行任务直到取消
    - `DoUntilTimeout()`   循环执行任务直到超时
    - `DoCancelableTask()` 执行任务直到取消或任务执行完成
    - `DoTimeLimitTask()`  执行任务直到超时或任务执行完成
- `RunXX(task func())`    
    - 在后台协程执行任务，返回`<-chan struct{}`，任务执行完成管道关闭
    - `Run()`                    后台执行任务
    - `RunCancelable()`          后台执行任务直到取消或任务执行完成
    - `RunTimeLimit()`           后台执行任务直到超时或任务执行完成    
    - `RunUntilCancel()`         后台循环执行任务直到取消
    - `RunUntilTimeout()`        后台循环执行任务直到超时
    - `RunCancelableInterval()`  后台定时循环执行任务直到取消
    - `RunTimeLimitInterval()`   后台定时循环执行任务直到超时
- `RunXXTask(task Task)`  
    - 在后台协程执行任务，返回`<-chan Result`，任务执行完成管道关闭
    - `RunTask()`           后台执行任务
    - `RunCancelableTask()` 后台执行任务直到取消或任务执行完成
    - `RunTimeLimitTask()`  后台执行任务直到超时或任务执行完成    
- 注意：外部传入的`context.Context`取消后，***后台任务仍然继续执行直到完成***

```go
	//后台执行任务，任务完成后返回的管道关闭
	done:=async.Run(func() {
		fmt.Println(`Hi`)
	})
	<-done //等待任务执行完成

	//执行任务，1秒后超时ctx取消导致返回的管道关闭，即任务被取消
	//但是后台协程仍然继续执行任务直到完成，即无法中断正在执行的协程
	//如果希望ctx取消时后台任务立刻终止，则需要自行监听ctx.Done()
	ctx,_:=context.WithTimeout(context.Background(),1*time.Second)
	<-async.RunCancelable(ctx, func() {
		count:=3
		for count>0{
			fmt.Println(`task:`,count)
			time.Sleep(1*time.Second)
			count--
		}
	})
	
	//执行任务并获取返回结果
	result:=<-async.RunTask(async.ToTask(func() async.Result {
		return async.NewResult(`1`,nil)
	}))
	fmt.Println(result.Value()) // "1"
```    

### Runner
- `Runner` 适用于执行长期运行的任务，提供以下方法
    - `Start()/MustStart()`  开始执行任务
    - `Stop()`               停止执行任务
    - `Wait()`               等待任务执行完成
    - `Running()`            是否正在执行任务
- `BgTask/BgTaskFn` 定义Runner执行的任务(函数)
    - 方法签名：`Run(ctx context.Context) (<-chan struct{}, error)`
    - `Run()`不要阻塞当前调用协程，应仅做任务初始化，然后启用后台协程执行真正的业务处理
    - 任务初始化出错应返回error
    - 任务执行完成后应关闭返回的管道
    - 任务初始化时应监听ctx.Done(),在ctx取消后关闭返回的管道

```go
	//创建1个runner执行1个长期运行的定时任务：每1秒打印1条消息
	runner:=async.NewRunner(async.BgTaskFn(func(ctx context.Context) (<-chan struct{}, error) {
        
        //async.RunCancelableInterval()满足BgTask实现要求：
        // - 不阻塞当前协程，启用1个后台协程执行定时任务
        // - 监听传入的ctx.Done()，在其取消后结束任务执行，并关闭返回的管道
        // - 没有初始化错误，直接返回nil
        // 综上：一般会配合使用async.RunXX()实现BgTask，否则代码量仍然较多
		return async.RunCancelableInterval(ctx,1*time.Second, func(c async.Context) {
			fmt.Println(`task:`,c.Count())
		}),nil
	}))

	//启动runner，并在5秒后关闭
	runner.MustStart()
	time.AfterFunc(5*time.Second,runner.Stop)

	//等待runner执行完成(任务执行完成或关闭)
	runner.Wait()
	fmt.Println(`runner stopped`)
```

### Pool
- 协程池限制执行任务时使用的协程数
- 协程池提供`RunXX()/RunXXTask()`
    - 调用此方法可将任务添加到池里，然后由池里的协程执行
    - 如果池没有空闲的协程，则任务将暂存到内部的任务队列里
    - 如果任务队列已满，则将阻塞调用协程，直到任务添加成功
- 关闭协程池
    - 协程池使用完毕后应关闭，以退出内部创建的协程
    - 调用协程池关闭方法后，将丢弃任务队列待处理的任务，且禁止提交新任务    
```go
	//协程池，用于限制执行任务的协程数，进而减少系统资源消耗
	//以下声明最多添加6个任务：4 worker+2 task in queue
	pool := async.NewWorkerPool(2, 4, 2)

	for i:=1;i<=6;i++{
		i:=i
		pool.Run(func() {
			time.Sleep(time.Duration(i)*time.Second)
			fmt.Println(`done:`,i)
		})
	}

	pool.Wait()  //等待全部任务执行完成
	pool.Close() //关闭池
``` 

### Group
- 任务组，执行一组任务并保存其执行结果。提供以下方法：
    - `RunXX()`        添加任务
    - `WaitXX()`       等待任务执行完成，或者取消/超时
    - `XXTaskCount()`  当前全部/待完成/已完成的任务数
    - `WithPool()`     设置使用的协程池，限制执行任务可用协程数
- `GroupResult` 任务组执行结果，每个任务按添加顺序获取1个递增序列号作为索引值
    - `ResultList()` 任务执行结果List，按照任务添加顺序排序
    - `ResultMap()`  任务执行结果Map，key为任务索引值，即任务添加时获取的序列号
    - `Get(idx)`     获取索引值对应的任务执行结果
    - `Error()`      第1个失败任务返回的错误
    - `Size()`       任务结果总数
    - `FirstOk()/FistOkIdx()`     第1个成功任务执行结果/索引值
    - `FirstDone()/FistDoneIdx()` 第1个完成任务执行结果/索引值，不管其是否有返回错误
- `ResultList/ResultMap`  用于保存多个任务结果，提供部分帮助方法
    - 类型定义：`type ResultList []Result`
    - 类型定义：`type ResultMap  map[int]Result`
    - `Each(fn)`     遍历执行结果
    - `ForEach(fn)`  遍历执行结果，如果`fn`返回`false`则不再遍历下1条数据
    
```go
	//创建1个新的任务，此任务在沉睡i秒后结束
	task:= func(i int)async.TaskFn{
		return async.TaskFn(func() async.Result {
			time.Sleep(time.Duration(i)*time.Second)
			return async.NewResult(i,nil)
		})
	}

	//添加2个任务，分别沉睡1,2秒后结束
	g:=async.NewGroup()
	g.RunTask(task(1))
	g.RunTask(task(2))

	//等待任务执行完成并获取结果
	result:=g.Wait()
	fmt.Println(result.Size()) // 2个任务

	//第1个完成/成功完成的任务索引值都是0
	fmt.Println(result.FirstDoneIdx()) // 0
	fmt.Println(result.FirstOkIdx())   // 0

	//获取第1个任务的执行结果
	fmt.Println(result.Get(0).Value()) // 1

	//遍历所有任务执行结果，key为任务索引值
	result.ResultMap().Each(func(key int, r async.Result) {
		fmt.Printf("task %v result: %v \n", key, r.Value())
	})

	//添加2个任务，沉睡1,3秒后结束
	//新创建1个组，以避免已有结果的影响
	g=async.NewGroup()
	g.RunTask(task(1))
	g.RunTask(task(3))

	//等待任务执行完成，2秒后超时不再等待
	//超时后，后续完成的任务结果不会保存到组里
	result=g.WaitOrTimeout(2*time.Second)

	//仅包含第1个任务的结果，第2个任务因等待超时未完成
	fmt.Println(result.Size())    // 1
	fmt.Println(result.Timeout()) // true，表示等待超时

	//等待超时后，添加到组里的任务仍然继续执行
	//以下再等待2秒，任务2执行完成，但是执行结果仍然不会添加到组里
	//实际上，2次调用g.WaitOrTimeout()返回的是相同的结果
	result=g.WaitOrTimeout(2*time.Second)
	fmt.Println(result.Size())    // 1
	fmt.Println(result.Timeout()) // true
```
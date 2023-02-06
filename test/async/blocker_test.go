package async

import (
	"github.com/bingooh/b-go-util/async"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"
)

func TestBlocker(t *testing.T) {
	r := require.New(t)

	b1 := async.NewBlockerOf(1)
	r.True(b1.HasValue())

	b2 := async.NewBlockerOf(nil)
	r.True(b2.HasValue())

	b3 := async.NewBlocker()
	r.False(b3.HasValue())

	b3.Put(1)
	r.True(b3.HasValue())
	r.EqualValues(1, b3.Get()) //此时Get()不会阻塞

	r.Equal(1, b3.Remove())
	r.False(b3.HasValue())
	r.Nil(b3.Peek()) //Peek()不会阻塞

	r.Nil(b3.Remove())
	r.False(b3.HasValue())

	b3.Put(nil) //放入空值仍将解除阻塞
	r.True(b3.HasValue())
	r.Nil(b3.Remove())

	start := time.Now()
	async.EnsureRun(func() {
		time.Sleep(3 * time.Second)
		b3.Put(1)
	})

	r.Equal(1, b3.Get()) //将被阻塞
	r.WithinDuration(time.Now(), start.Add(3*time.Second), 100*time.Millisecond)
}

func TestBlockGroup(t *testing.T) {
	r := require.New(t)

	start := time.Now()
	resetStart := func() {
		start = time.Now()
	}

	period := 2 * time.Second
	assertWithinPeriod := func(period time.Duration) {
		r.WithinDuration(time.Now(), start.Add(period), 100*time.Millisecond)
	}

	g := async.NewBlockGroup()
	delayPut := func(key string, val interface{}) {
		time.AfterFunc(period, func() {
			g.Put(key, val)
		})
	}

	k1 := `k1`
	r.False(g.Has(k1))
	r.Nil(g.Peek(k1)) //不会阻塞

	resetStart()
	delayPut(k1, 1)
	r.EqualValues(1, g.Get(k1)) //将被阻塞
	r.True(g.Has(k1))
	assertWithinPeriod(period)

	resetStart()
	g.Put(k1, nil)
	r.Nil(g.Peek(k1))
	r.Nil(g.Get(k1)) //因为key已存在，不会被阻塞
	assertWithinPeriod(100 * time.Millisecond)

	g.Put(k1, 2)
	r.EqualValues(2, g.Remove(k1))
	r.False(g.Has(k1))

	resetStart()
	c := async.DoTimeLimitTask(period, func() { g.Get(k1) }) //将超时
	r.Error(c.Error())
	assertWithinPeriod(period)

	r.Nil(g.Remove(k1))
	r.False(g.PutIfExist(k1, 1))
	g.Put(k1, 1)
	r.True(g.PutIfExist(k1, 2))
	r.EqualValues(2, g.Get(k1))

	n := 1000
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		i := i
		async.EnsureRun(func() {
			defer wg.Done()

			switch {
			case i%33 == 0:
				g.Put(k1, i)
			case i%5 == 0:
				g.Remove(k1)
			default:
				g.Get(k1)
			}
		})
	}

	time.Sleep(5 * time.Second)
	g.Put(k1, 1) //避免死锁，即最后1步必须存放1个值，否则前面g.Get()将被阻塞

	wg.Wait()
}

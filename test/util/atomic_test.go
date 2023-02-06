package util

import (
	"errors"
	"fmt"
	"github.com/bingooh/b-go-util/util"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestTmp(t *testing.T) {
	rs1 := []int{1, 2}
	rs2 := make([]int, len(rs1))
	copy(rs2, rs1)
	fmt.Println(rs2)
}

func TestAtomicError(t *testing.T) {
	r := require.New(t)

	a1 := util.NewAtomicError()
	r.Nil(a1.Value())

	a2 := util.NewAtomicError().Set(nil)
	r.Nil(a2.Value())

	e3 := errors.New(`e3`)
	a3 := util.NewAtomicError().Set(e3)
	r.Equal(e3, a3.Value())

	a3.Set(nil)
	r.Nil(a3.Value())

	e4 := errors.New(`e4`)
	e5 := errors.New(`e5`)
	r.True(a3.SetIfAbsent(e4))
	r.Equal(e4, a3.Value())
	r.False(a3.SetIfAbsent(e4))
	r.Equal(e4, a3.Value())
	r.False(a3.CASwap(e5, e4))
	r.Equal(e4, a3.Value())
	r.True(a3.CASwap(e4, e5))
	r.Equal(e5, a3.Value())

	r.True(a3.CASwap(e5, nil))
	r.Nil(a3.Value())
}

func TestAtomicTime(t *testing.T) {
	r := require.New(t)

	zero := time.Time{}
	now := time.Now()
	end := now.Add(1 * time.Second)

	a1 := util.NewAtomicTime()
	r.Zero(a1.Value())
	r.Equal(zero, a1.Value())

	a2 := util.NewAtomicTime().Set(now)
	r.Equal(now, a2.Value())
	r.False(a2.CASwap(end, now))
	r.Equal(now, a2.Value())
	r.True(a2.CASwap(now, end))
	r.Equal(end, a2.Value())

	a2.Set(zero)
	r.Equal(zero, a2.Value())
}

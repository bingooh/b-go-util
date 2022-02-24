package util

import (
	"github.com/bingooh/b-go-util/_interface"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInterfaceFlat(t *testing.T) {
	r := require.New(t)
	r.True(true)

	assertEquals := func(actual []interface{}, expect ...interface{}) {
		r.Equal(len(expect), len(actual))
		r.EqualValues(expect, actual)

		/*		for i, v := range expect {
				r.EqualValues(v,actual[i])
			}*/
	}

	flat := _interface.Flat
	assertEquals(flat(1), 1)
	assertEquals(flat(1, 2), 1, 2)
	assertEquals(flat([]int{1, 2}), 1, 2)
	assertEquals(flat(1, []int{2, 3}, 4, `a`, nil, 0), 1, 2, 3, 4, `a`, nil, 0)
	assertEquals(flat(1, []interface{}{2, `a`}, 3), 1, 2, `a`, 3)
	assertEquals(flat(1, []interface{}{2, []interface{}{3}}, 4), 1, 2, []interface{}{3}, 4) //仅支持1层嵌套
}

package util

import (
	"github.com/bingooh/b-go-util/_interface"
	"github.com/bingooh/b-go-util/_reflect"
	"github.com/stretchr/testify/require"
	"testing"
)

type User struct {
	ID      int     `json:"id"`
	Name    *string `json:"name"`
	IsSuper bool    `json:"is_super"`
}

type UserNoTag struct {
	ID      int
	Name    *string
	IsSuper bool
}

func TestInterfaceFlat(t *testing.T) {
	r := require.New(t)

	assertEquals := func(actual []interface{}, expect ...interface{}) {
		r.Equal(len(expect), len(actual))
		r.EqualValues(expect, actual)
	}

	flat := _interface.Flat

	assertEquals(flat())
	assertEquals(flat(nil), nil) //nil会被添加仅切片
	assertEquals(flat(1), 1)
	assertEquals(flat(1, 2), 1, 2)
	assertEquals(flat([]int{1, 2}), 1, 2)
	assertEquals(flat(1, []int{2, 3}, 4, `a`, nil, 0), 1, 2, 3, 4, `a`, nil, 0)
	assertEquals(flat(1, []interface{}{2, `a`}, 3), 1, 2, `a`, 3)
	assertEquals(flat(1, []interface{}{2, []interface{}{3}}, 4), 1, 2, []interface{}{3}, 4) //仅支持1层嵌套
}

func TestReflect(t *testing.T) {
	r := require.New(t)

	var a interface{} = (*int)(nil)
	r.True(_reflect.IsNil(a))
	r.False(_reflect.IsPrimitive(a))
	r.False(_reflect.IsPrimitive(nil))

	var i int
	var j *int
	r.True(_reflect.IsPrimitive(i))
	r.False(_reflect.IsPrimitive(j))

	name := `b`
	emptyName := ``
	user := &User{ID: 1, Name: &name}
	userNoTag := &UserNoTag{ID: 1, Name: nil}

	v, ok := _reflect.PluckStructFieldValue(nil, `name`, false)
	r.False(ok)
	r.Nil(v)

	v, ok = _reflect.PluckStructFieldValue(user, `id`, false) //匹配json标签值
	r.True(ok)
	r.Equal(1, v)

	v, ok = _reflect.PluckStructFieldValue(user, `ID`, false) //匹配字段名称，区分大小写
	r.True(ok)
	r.Equal(1, v)

	v, ok = _reflect.PluckStructFieldValue(userNoTag, `id`, false)
	r.False(ok) //匹配不到，因为userNoTag没有json标签
	r.Nil(v)

	v, ok = _reflect.PluckStructFieldValue(userNoTag, `ID`, false)
	r.True(ok)
	r.Equal(1, v)

	//匹配基本数据指针类型
	v, ok = _reflect.PluckStructFieldValue(user, `name`, false)
	r.True(ok)
	r.Equal(name, v)

	//userNoTag.Name值为nil，不会被匹配到
	v, ok = _reflect.PluckStructFieldValue(userNoTag, `Name`, false)
	r.False(ok)
	r.Nil(v)

	userNoTag.Name = &emptyName
	v, ok = _reflect.PluckStructFieldValue(userNoTag, `Name`, false)
	r.True(ok)
	r.Equal(``, v)

	//不忽略零值
	v, ok = _reflect.PluckStructFieldValue(user, `is_super`, false)
	r.True(ok)
	r.Equal(false, v)

	//忽略零值
	v, ok = _reflect.PluckStructFieldValue(user, `is_super`, true)
	r.False(ok)
	r.Nil(v)

	users := []*User{
		{ID: 1, IsSuper: true},
		{ID: 2, IsSuper: false},
		{ID: 3, IsSuper: true},
	}

	rs := _reflect.PluckStructFieldValues(`id`, false, false, nil)
	r.EqualValues(0, len(rs))

	rs = _reflect.PluckStructFieldValues(`id`, false, false, users)
	r.EqualValues([]interface{}{1, 2, 3}, rs)

	rs = _reflect.PluckStructFieldValues(`is_super`, true, false, users)
	r.EqualValues([]interface{}{true, true}, rs)

	rs = _reflect.PluckStructFieldValues(`is_super`, false, true, users)
	r.EqualValues([]interface{}{true, false}, rs)

	m1 := make(map[int]bool)
	err := _reflect.ConvertStructListToMap(`id`, `is_super`, m1, users)
	r.NoError(err)
	r.EqualValues(3, len(m1))

	m2 := make(map[int]bool)
	err = _reflect.ConvertStructListToMap(`id`, `name`, m2, users)
	r.NoError(err)
	r.EqualValues(0, len(m2)) //map类型不匹配，value类型为string

	m3 := _reflect.ConvertStructToMap(user, `id`, `name`, `is_super`)
	r.EqualValues(3, len(m3))
	r.EqualValues(false, m3[`is_super`])

	m4 := make(map[int]interface{})
	err = _reflect.ConvertStructListToMap(`id`, `is_super`, m4, users)
	r.NoError(err)
	r.EqualValues(3, len(m4))

	m5 := make(map[int]*User)
	err = _reflect.ConvertStructListToMap(`id`, ``, m5, users)
	r.NoError(err)
	r.EqualValues(3, len(m5))
}

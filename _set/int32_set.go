package _set

import "fmt"

type Int32Set struct {
	m map[int32]struct{}
}

func NewInt32Set(items ...int32) *Int32Set {
	nm := make(map[int32]struct{}, len(items))
	for _, v := range items {
		nm[v] = struct{}{}
	}

	return &Int32Set{m: nm}
}

func (h *Int32Set) Size() int {
	if h == nil {
		return 0
	}
	return len(h.m)
}

func (h *Int32Set) Empty() bool {
	if h == nil {
		return true
	}
	return len(h.m) == 0
}

func (h *Int32Set) Contains(item int32) bool {
	_, ok := h.m[item]
	return ok
}

func (h *Int32Set) ContainsAll(items ...int32) bool {
	for _, item := range items {
		if _, ok := h.m[item]; !ok {
			return false
		}
	}

	return true
}

func (h *Int32Set) ContainsSet(other *Int32Set) bool {
	if other == nil || other.Size() == 0 {
		return true
	}

	for k, _ := range other.m {
		if _, ok := h.m[k]; !ok {
			return false
		}
	}

	return true
}

func (h *Int32Set) ForEach(fn func(item int32) bool) {
	for k, _ := range h.m {
		if !fn(k) {
			return
		}
	}
}

func (h *Int32Set) Add(item int32) {
	h.m[item] = struct{}{}
}

func (h *Int32Set) AddAll(items ...int32) {
	for _, item := range items {
		h.m[item] = struct{}{}
	}
}

func (h *Int32Set) Remove(item int32) bool {
	if _, ok := h.m[item]; !ok {
		return false
	}

	delete(h.m, item)
	return true
}

func (h *Int32Set) RemoveAll(items ...int32) {
	for _, item := range items {
		delete(h.m, item)
	}
}

func (h *Int32Set) Clear() {
	h.m = make(map[int32]struct{})
}

func (h *Int32Set) Clone() *Int32Set {
	m := make(map[int32]struct{}, len(h.m))

	for k, v := range h.m {
		m[k] = v
	}

	return &Int32Set{m: m}
}

func (h *Int32Set) ToSlice() []int32 {
	list := make([]int32, 0, len(h.m))

	for k, _ := range h.m {
		list = append(list, k)
	}

	return list
}

func (h *Int32Set) String() string {
	return fmt.Sprintf("%v", h.ToSlice())
}

// 并集
func (h *Int32Set) Union(other *Int32Set) *Int32Set {
	if other == nil || len(other.m) == 0 {
		return h
	}

	for k, v := range other.m {
		h.m[k] = v
	}

	return h
}

// 差集
func (h *Int32Set) Diff(other *Int32Set) *Int32Set {
	if other == nil || len(other.m) == 0 {
		return h
	}

	for k, _ := range other.m {
		delete(h.m, k)
	}

	return h
}

// 交集
func (h *Int32Set) Intersect(other *Int32Set) *Int32Set {
	if other == nil || len(other.m) == 0 {
		h.m = make(map[int32]struct{})
		return h
	}

	m := make(map[int32]struct{})
	for k, v := range h.m {
		if _, ok := other.m[k]; ok {
			m[k] = v
		}
	}

	h.m = m
	return h
}

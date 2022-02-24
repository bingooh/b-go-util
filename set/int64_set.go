package util

import "fmt"

type Int64Set struct {
	m map[int64]struct{}
}

func NewInt64Set(items ...int64) *Int64Set {
	nm := make(map[int64]struct{}, len(items))
	for _, v := range items {
		nm[v] = struct{}{}
	}

	return &Int64Set{m: nm}
}

func (h *Int64Set) Size() int {
	if h == nil {
		return 0
	}
	return len(h.m)
}

func (h *Int64Set) Empty() bool {
	if h == nil {
		return true
	}
	return len(h.m) == 0
}

func (h *Int64Set) Contains(item int64) bool {
	_, ok := h.m[item]
	return ok
}

func (h *Int64Set) Add(item int64) {
	h.m[item] = struct{}{}
}

func (h *Int64Set) AddAll(items ...int64) {
	for _, item := range items {
		h.m[item] = struct{}{}
	}
}

func (h *Int64Set) Remove(item int64) bool {
	if _, ok := h.m[item]; !ok {
		return false
	}

	delete(h.m, item)
	return true
}

func (h *Int64Set) RemoveAll(items ...int64) {
	for _, item := range items {
		delete(h.m, item)
	}
}

func (h *Int64Set) Clear() {
	h.m = make(map[int64]struct{})
}

func (h *Int64Set) Clone() *Int64Set {
	m := make(map[int64]struct{}, len(h.m))

	for k, v := range h.m {
		m[k] = v
	}

	return &Int64Set{m: m}
}

func (h *Int64Set) ToSlice() []int64 {
	list := make([]int64, 0, len(h.m))

	for k, _ := range h.m {
		list = append(list, k)
	}

	return list
}

func (h *Int64Set) String() string {
	return fmt.Sprintf("%v", h.ToSlice())
}

// 并集
func (h *Int64Set) Union(other *Int64Set) *Int64Set {
	if other == nil || len(other.m) == 0 {
		return h
	}

	for k, v := range other.m {
		h.m[k] = v
	}

	return h
}

// 差集
func (h *Int64Set) Diff(other *Int64Set) *Int64Set {
	if other == nil || len(other.m) == 0 {
		return h
	}

	for k, _ := range other.m {
		delete(h.m, k)
	}

	return h
}

// 交集
func (h *Int64Set) Intersect(other *Int64Set) *Int64Set {
	if other == nil || len(other.m) == 0 {
		h.m = make(map[int64]struct{})
		return h
	}

	m := make(map[int64]struct{})
	for k, v := range h.m {
		if _, ok := other.m[k]; ok {
			m[k] = v
		}
	}

	h.m = m
	return h
}

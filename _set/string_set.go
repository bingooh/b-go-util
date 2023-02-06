package _set

import "fmt"

type StringSet struct {
	m map[string]struct{}
}

func NewStringSet(items ...string) *StringSet {
	nm := make(map[string]struct{}, len(items))
	for _, v := range items {
		nm[v] = struct{}{}
	}

	return &StringSet{m: nm}
}

func (h *StringSet) Size() int {
	if h == nil {
		return 0
	}
	return len(h.m)
}

func (h *StringSet) Empty() bool {
	if h == nil {
		return true
	}
	return len(h.m) == 0
}

func (h *StringSet) Contains(item string) bool {
	_, ok := h.m[item]
	return ok
}

func (h *StringSet) ContainsAll(items ...string) bool {
	for _, item := range items {
		if _, ok := h.m[item]; !ok {
			return false
		}
	}

	return true
}

func (h *StringSet) ContainsSet(other *StringSet) bool {
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

func (h *StringSet) Equal(other *StringSet) bool {
	if h.Size() != other.Size() {
		return false
	}

	if h.Empty() && other.Empty() {
		return true
	}

	for k, _ := range h.m {
		if _, ok := other.m[k]; !ok {
			return false
		}
	}

	return true
}

func (h *StringSet) ForEach(fn func(item string) bool) {
	for k, _ := range h.m {
		if !fn(k) {
			return
		}
	}
}

func (h *StringSet) Add(item string) {
	h.m[item] = struct{}{}
}

func (h *StringSet) AddAll(items ...string) {
	for _, item := range items {
		h.m[item] = struct{}{}
	}
}

func (h *StringSet) Remove(item string) bool {
	if _, ok := h.m[item]; !ok {
		return false
	}

	delete(h.m, item)
	return true
}

func (h *StringSet) RemoveAll(items ...string) {
	for _, item := range items {
		delete(h.m, item)
	}
}

func (h *StringSet) Clear() {
	h.m = make(map[string]struct{})
}

func (h *StringSet) Clone() *StringSet {
	m := make(map[string]struct{}, len(h.m))

	for k, v := range h.m {
		m[k] = v
	}

	return &StringSet{m: m}
}

func (h *StringSet) Pop(n int) *StringSet {
	if n < 1 {
		return NewStringSet()
	}

	if h.Size() <= n {
		s := &StringSet{m: h.m}
		h.m = make(map[string]struct{})
		return s
	}

	i := 0
	m := make(map[string]struct{}, n)
	for k, v := range h.m {
		m[k] = v
		delete(h.m, k)

		i++
		if i == n {
			break
		}
	}

	return &StringSet{m: m}
}

func (h *StringSet) ToSlice() []string {
	list := make([]string, 0, len(h.m))

	for k, _ := range h.m {
		list = append(list, k)
	}

	return list
}

func (h *StringSet) String() string {
	return fmt.Sprintf("%v", h.ToSlice())
}

// 并集
func (h *StringSet) Union(other *StringSet) *StringSet {
	if other == nil || len(other.m) == 0 {
		return h
	}

	for k, v := range other.m {
		h.m[k] = v
	}

	return h
}

// 差集
func (h *StringSet) Diff(other *StringSet) *StringSet {
	if other == nil || len(other.m) == 0 {
		return h
	}

	for k, _ := range other.m {
		delete(h.m, k)
	}

	return h
}

func (h *StringSet) NewDiff(other *StringSet) *StringSet {
	if other == nil || len(other.m) == 0 {
		return h.Clone()
	}

	s := NewStringSet()
	for k, _ := range h.m {
		if _, ok := other.m[k]; !ok {
			s.Add(k)
		}
	}

	return s
}

// 交集
func (h *StringSet) Intersect(other *StringSet) *StringSet {
	if other == nil || len(other.m) == 0 {
		h.m = make(map[string]struct{})
		return h
	}

	m := make(map[string]struct{})
	for k, v := range h.m {
		if _, ok := other.m[k]; ok {
			m[k] = v
		}
	}

	h.m = m
	return h
}

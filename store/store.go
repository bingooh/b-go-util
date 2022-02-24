package store

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
)

var ErrKeyNotExist = errors.New("key not exist")

type Store interface {
	Keys() []string
	HasKey(key string) bool
	Get(key string) (interface{}, bool)
	GetVal(key string) interface{}
	GetOrDefault(key string, df interface{}) interface{}
	GetString(key string) (string, error)
	GetBool(key string) (bool, error)
	GetInt(key string) (int, error)
	GetInt64(key string) (int64, error)
	GetStringOrDefault(key, df string) string
	GetBoolOrDefault(key string, df bool) bool
	GetIntOrDefault(key string, df int) int
	GetInt64OrDefault(key string, df int64) int64
	Put(key string, value interface{}) interface{}
	Remove(key string) interface{}
	Clear()
}

type MemoryStore struct {
	m map[string]interface{}
	l sync.RWMutex
}

func NewMemoryStore(size int) Store {
	return &MemoryStore{
		m: make(map[string]interface{}, size),
	}
}

func NewMemoryStoreOf(m map[string]interface{}) Store {
	nm := make(map[string]interface{}, len(m))
	for k, v := range m {
		nm[k] = v
	}

	return &MemoryStore{m: nm}
}

func (s *MemoryStore) newCastErr(expect string, val interface{}) error {
	return fmt.Errorf("can't cast '(%T)%v' to %s", val, val, expect)
}

func (s *MemoryStore) Keys() []string {
	s.l.RLock()
	defer s.l.RUnlock()

	keys := make([]string, 0, len(s.m))
	for k := range s.m {
		keys = append(keys, k)
	}

	return keys
}

func (s *MemoryStore) HasKey(key string) bool {
	_, ok := s.Get(key)
	return ok
}

func (s *MemoryStore) Get(key string) (interface{}, bool) {
	s.l.RLock()
	defer s.l.RUnlock()

	val, ok := s.m[key]
	return val, ok
}

func (s *MemoryStore) GetVal(key string) interface{} {
	v, _ := s.Get(key)
	return v
}

func (s *MemoryStore) GetString(key string) (string, error) {
	s.l.RLock()
	defer s.l.RUnlock()

	v, ok := s.m[key]
	if !ok {
		return "", ErrKeyNotExist
	}

	switch val := v.(type) {
	case string:
		return val, nil
	default:
		return fmt.Sprintf(`%v`, val), nil
	}
}

func (s *MemoryStore) GetBool(key string) (bool, error) {
	s.l.RLock()
	defer s.l.RUnlock()

	v, ok := s.m[key]
	if !ok {
		return false, ErrKeyNotExist
	}

	switch val := v.(type) {
	case bool:
		return val, nil
	case string:
		return strconv.ParseBool(val)
	}

	return false, s.newCastErr("bool", v)
}

func (s *MemoryStore) GetInt(key string) (int, error) {
	s.l.RLock()
	defer s.l.RUnlock()

	v, ok := s.m[key]
	if !ok {
		return 0, ErrKeyNotExist
	}

	switch val := v.(type) {
	case string:
		return strconv.Atoi(val)
	case int:
		return val, nil
	case int8:
		return int(val), nil
	case int16:
		return int(val), nil
	case int32:
		return int(val), nil
	case int64:
		return int(val), nil
	default:
		return 0, s.newCastErr("int", v)
	}
}

func (s *MemoryStore) GetInt64(key string) (int64, error) {
	s.l.RLock()
	defer s.l.RUnlock()

	v, ok := s.m[key]
	if !ok {
		return 0, ErrKeyNotExist
	}

	switch val := v.(type) {
	case string:
		return strconv.ParseInt(val, 10, 64)
	case int64:
		return val, nil
	case int:
		return int64(val), nil
	case int8:
		return int64(val), nil
	case int16:
		return int64(val), nil
	case int32:
		return int64(val), nil
	default:
		return 0, s.newCastErr("int64", v)
	}
}

func (s *MemoryStore) GetOrDefault(key string, df interface{}) interface{} {
	s.l.RLock()
	defer s.l.RUnlock()

	if v, ok := s.m[key]; ok {
		return v
	}

	return df
}

func (s *MemoryStore) GetStringOrDefault(key, df string) string {
	if v, err := s.GetString(key); err == nil {
		return v
	}

	return df
}

func (s *MemoryStore) GetBoolOrDefault(key string, df bool) bool {
	if v, err := s.GetBool(key); err == nil {
		return v
	}

	return df
}

func (s *MemoryStore) GetIntOrDefault(key string, df int) int {
	if v, err := s.GetInt(key); err == nil {
		return v
	}

	return df
}

func (s *MemoryStore) GetInt64OrDefault(key string, df int64) int64 {
	if v, err := s.GetInt64(key); err == nil {
		return v
	}

	return df
}

func (s *MemoryStore) Put(key string, value interface{}) interface{} {
	s.l.Lock()
	defer s.l.Unlock()

	v := s.m[key]
	s.m[key] = value

	return v
}

func (s *MemoryStore) Remove(key string) interface{} {
	s.l.Lock()
	defer s.l.Unlock()

	v := s.m[key]
	delete(s.m, key)

	return v
}

func (s *MemoryStore) Clear() {
	s.l.Lock()
	defer s.l.Unlock()

	s.m = make(map[string]interface{})
}

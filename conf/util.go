package conf

import "sync"

const (
	wd    = "wd"
	debug = "debug"
)

var holder = newSyncMap()

type syncMap struct {
	lock sync.Mutex
	m    map[string]interface{}
}

func newSyncMap() *syncMap {
	return &syncMap{m: make(map[string]interface{})}
}

func (sm *syncMap) Get(key string) (interface{}, bool) {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	val, ok := sm.m[key]
	return val, ok
}

func (sm *syncMap) GetBool(key string) (bool, bool) {
	if val, ok := sm.Get(key); ok {
		if v, ok := val.(bool); ok {
			return v, true
		}
	}

	return false, false
}

func (sm *syncMap) GetString(key string) (string, bool) {
	if val, ok := sm.Get(key); ok {
		if v, ok := val.(string); ok {
			return v, true
		}
	}

	return "", false
}

func (sm *syncMap) Put(key string, val interface{}) {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	sm.m[key] = val
}

func SetWorkingDir(dir string) {
	holder.Put(wd, dir)
}

func GetWorkingDir() (string, bool) {
	return holder.GetString(wd)
}

func EnableDebug(val bool) {
	holder.Put(debug, val)
}

func IsDebugEnable() (bool, bool) {
	return holder.GetBool(debug)
}

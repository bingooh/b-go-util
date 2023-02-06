package util

import "fmt"

// Provider 可用于动态提供JSON解析用的val(指针)
type Provider struct {
	suppliers map[interface{}]func() interface{}
}

func NewProvider() *Provider {
	return &Provider{
		suppliers: make(map[interface{}]func() interface{}),
	}
}

// MustRegister key区分数据类型，int(1)!=int64(1)
func (p *Provider) MustRegister(supplier func() interface{}, keys ...interface{}) *Provider {
	AssertOk(len(keys) > 0, `keys为空`)
	AssertOk(supplier != nil, `supplier为空`)

	for _, key := range keys {
		if _, ok := p.suppliers[key]; ok {
			panic(fmt.Errorf(`provider已注册[key=%v]`, key))
		}

		p.suppliers[key] = supplier
	}

	return p
}

func (p *Provider) HasSupplier(key interface{}) bool {
	_, ok := p.suppliers[key]
	return ok
}

func (p *Provider) Get(key interface{}) interface{} {
	if fn, ok := p.suppliers[key]; ok {
		return fn()
	}

	return nil
}

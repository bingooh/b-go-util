package util

import (
	"github.com/valyala/bytebufferpool"
	"sync"
)

// BytesBuffer 不固定大小字节缓存，使用完应调用Close()
type BytesBuffer struct {
	*bytebufferpool.ByteBuffer
}

func NewBytesBuffer() *BytesBuffer {
	return &BytesBuffer{
		ByteBuffer: bytebufferpool.Get(),
	}
}

func (b *BytesBuffer) Close() {
	if b != nil && b.ByteBuffer != nil {
		bytebufferpool.Put(b.ByteBuffer)
		b.ByteBuffer = nil
	}
}

// DefaultFixedSizeBytesBufferPool 每个缓存大小默认为4KB
var DefaultFixedSizeBytesBufferPool = MustNewFixedSizeBytesBufferPool(4096)

// FixedSizeBytesBufferPool 固定大小字节缓存池
type FixedSizeBytesBufferPool struct {
	pool *sync.Pool
	size int //缓存大小
}

func MustNewFixedSizeBytesBufferPool(size int) *FixedSizeBytesBufferPool {
	AssertOk(size > 0, `size小于等于0`)

	p := &FixedSizeBytesBufferPool{size: size}
	p.pool = &sync.Pool{New: p.newBuffer}

	return p
}

func (p *FixedSizeBytesBufferPool) newBuffer() interface{} {
	return make([]byte, p.size)
}

// Get 返回的[]byte的len==cap==size，不要使用append添加数据
// 返回的[]byte可能包含上次写入的数据，需自行根据本次写入的数据长度获取内容
func (p *FixedSizeBytesBufferPool) Get() []byte {
	return p.pool.Get().([]byte)
}

func (p *FixedSizeBytesBufferPool) Put(v []byte) {
	//仅回收固定长度的[]byte
	if len(v) == p.size && cap(v) == p.size {
		p.pool.Put(v)
	}
}

package pgparty

import (
	"bytes"
	"sync"
)

var bufferPool = sync.Pool{}

func GetBuffer() *bytes.Buffer {
	b := bufferPool.Get()
	if b != nil {
		vb := b.(*bytes.Buffer)
		return vb
	}
	return &bytes.Buffer{}
}

func PutBuffer(b *bytes.Buffer) {
	if b == nil {
		return
	}
	b.Reset()
	bufferPool.Put(b)
}

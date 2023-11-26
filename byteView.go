package simpleCache

import "bytes"

type ByteView struct {
	byteView []byte
}

func (b *ByteView) Len() int {
	return len(b.byteView)
}

func (b *ByteView) String() string {
	return string(b.byteView)
}

func (b *ByteView) ByteSlice() []byte {
	return cloneByte(b.byteView)
}

func cloneByte(b []byte) []byte {
	clone := bytes.Clone(b)
	return clone
}

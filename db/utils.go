package db

import "encoding/binary"

func htoleu64(val uint64) (res []byte) {
	res = make([]byte, 8)
	binary.LittleEndian.PutUint64(res, val)
	return
}

func letohu64(le []byte) uint64 {
	return binary.LittleEndian.Uint64(le)
}

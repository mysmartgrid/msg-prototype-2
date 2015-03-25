package db

import (
	"github.com/boltdb/bolt"
)

type sensor struct {
	b  *bolt.Bucket
	id string
}

func (s *sensor) init(name string, dbId uint64) {
	s.b.Put(nameKey, []byte(name))
	s.b.Put(dbIdKey, htoleu64(dbId))
}

func (s *sensor) Id() string {
	return s.id
}

func (s *sensor) dbId() uint64 {
	return letohu64(s.b.Get(dbIdKey))
}

func (s *sensor) Name() string {
	return string(s.b.Get(nameKey))
}

func (s *sensor) SetName(name string) {
	s.b.Put(nameKey, []byte(name))
}

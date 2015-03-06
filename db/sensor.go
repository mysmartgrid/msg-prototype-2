package db

import (
	"github.com/boltdb/bolt"
)

type sensor struct {
	b  *bolt.Bucket
	id string
}

func (s *sensor) init(name string) {
	s.b.Put(nameKey, []byte(name))
}

func (s *sensor) Id() string {
	return s.id
}

func (s *sensor) Name() string {
	return string(s.b.Get(nameKey))
}

func (s *sensor) SetName(name string) {
	s.b.Put(nameKey, []byte(name))
}

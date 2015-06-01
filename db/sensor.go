package db

import (
	"github.com/boltdb/bolt"
)

type sensor struct {
	b  *bolt.Bucket
	id string
}

var (
	sensor_name = []byte("name")
	sensor_id   = []byte("dbId")
)

func (s *sensor) init(name string, dbId uint64) {
	s.b.Put(sensor_name, []byte(name))
	s.b.Put(sensor_id, htoleu64(dbId))
}

func (s *sensor) Id() string {
	return s.id
}

func (s *sensor) dbId() uint64 {
	return letohu64(s.b.Get(sensor_id))
}

func (s *sensor) Name() string {
	return string(s.b.Get(sensor_name))
}

func (s *sensor) SetName(name string) {
	s.b.Put(sensor_name, []byte(name))
}

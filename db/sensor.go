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
	sensor_port = []byte("port")
	sensor_unit = []byte("unit")
)

func (s *sensor) init(name string, dbId uint64, unit string, port int32) {
	s.b.Put(sensor_name, []byte(name))
	s.b.Put(sensor_id, htoleu64(dbId))
	s.b.Put(sensor_unit, []byte(unit))
	s.b.Put(sensor_port, htoleu64(uint64(port)))
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

func (s *sensor) SetName(name string) error {
	return s.b.Put(sensor_name, []byte(name))
}

func (s *sensor) Port() int32 {
	if val := s.b.Get(sensor_port); val != nil {
		return int32(letohu64(val))
	}
	return -1
}

func (s *sensor) Unit() string {
	if val := s.b.Get(sensor_unit); val != nil {
		return string(val)
	}
	return ""
}

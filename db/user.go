package db

import (
	"github.com/boltdb/bolt"
	"time"
)

type user struct {
	tx *tx
	b  *bolt.Bucket
	id string
}

func (u *user) init(dbId uint64) {
	u.b.CreateBucketIfNotExists(dbUserDevicesKey)
	u.b.Put(dbIdKey, htoleu64(dbId))
}

func (u *user) AddDevice(id string, key []byte) (Device, error) {
	idBytes := []byte(id)
	if len(idBytes) == 0 || len(idBytes) >= bolt.MaxKeySize {
		return nil, InvalidId
	}

	b := u.b.Bucket(dbUserDevicesKey)
	db, err := b.CreateBucket(idBytes)
	if err != nil {
		return nil, IdExists
	}
	seq, err := b.NextSequence()
	if err != nil {
		return nil, err
	}

	result := &device{db, u, id}
	result.init(key, id, seq)
	return result, nil
}

func (u *user) RemoveDevice(id string) error {
	idBytes := []byte(id)
	if len(idBytes) == 0 || len(idBytes) >= bolt.MaxKeySize {
		return InvalidId
	}

	b := u.b.Bucket(dbUserDevicesKey)
	if b.Bucket(idBytes) == nil {
		return InvalidId
	}
	return b.DeleteBucket(idBytes)
}

func (d *user) Device(id string) Device {
	b := d.b.Bucket(dbUserDevicesKey).Bucket([]byte(id))
	if b != nil {
		return &device{b, d, id}
	}
	return nil
}

func (d *user) Devices() map[string]Device {
	result := make(map[string]Device)
	b := d.b.Bucket(dbUserDevicesKey)
	b.ForEach(func(k, v []byte) error {
		result[string(k)] = &device{b.Bucket(k), d, string(k)}
		return nil
	})
	return result
}

func (u *user) Id() string {
	return u.id
}

func (u *user) dbId() uint64 {
	return letohu64(u.b.Get(dbIdKey))
}

func (u *user) LoadReadings(since time.Time, sensors map[Device][]Sensor) (map[Device]map[Sensor][]Value, error) {
	return u.tx.loadReadings(since, u, sensors)
}

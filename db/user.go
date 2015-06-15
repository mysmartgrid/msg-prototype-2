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

var (
	user_id      = []byte("dbId")
	user_devices = []byte("devices")
)

func (u *user) init(dbId uint64) {
	u.b.CreateBucketIfNotExists(user_devices)
	u.b.Put(user_id, htoleu64(dbId))
}

func (u *user) AddDevice(id string, key []byte) (Device, error) {
	idBytes := []byte(id)
	if len(idBytes) == 0 || len(idBytes) >= bolt.MaxKeySize {
		return nil, InvalidId
	}

	b := u.b.Bucket(user_devices)
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

	dev := u.Device(id)
	if dev == nil {
		return InvalidId
	}

	for id, _ := range dev.Sensors() {
		if err := dev.RemoveSensor(id); err != nil {
			return err
		}
	}

	b := u.b.Bucket(user_devices)
	return b.DeleteBucket(idBytes)
}

func (d *user) Device(id string) Device {
	b := d.b.Bucket(user_devices).Bucket([]byte(id))
	if b != nil {
		return &device{b, d, id}
	}
	return nil
}

func (d *user) Devices() map[string]Device {
	result := make(map[string]Device)
	b := d.b.Bucket(user_devices)
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
	return letohu64(u.b.Get(user_id))
}

func (u *user) LoadReadings(since time.Time, sensors map[Device][]Sensor) (map[Device]map[Sensor][]Value, error) {
	return u.tx.loadReadings(since, u, sensors)
}

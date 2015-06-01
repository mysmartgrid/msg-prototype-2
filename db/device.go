package db

import (
	"github.com/boltdb/bolt"
)

type device struct {
	b    *bolt.Bucket
	user *user
	id   string
}

var (
	device_name    = []byte("name")
	device_id      = []byte("dbId")
	device_key     = []byte("key")
	device_sensors = []byte("sensors")
)

func (d *device) init(key []byte, name string, dbId uint64) {
	d.b.CreateBucketIfNotExists(device_sensors)
	d.b.Put(device_key, key)
	d.b.Put(device_name, []byte(name))
	d.b.Put(device_id, htoleu64(dbId))
}

func (d *device) AddSensor(id string) (Sensor, error) {
	idBytes := []byte(id)
	if len(idBytes) == 0 || len(idBytes) >= bolt.MaxKeySize {
		return nil, InvalidId
	}

	b := d.b.Bucket(device_sensors)
	sb, err := b.CreateBucket(idBytes)
	if err != nil {
		return nil, IdExists
	}
	seq, err := b.NextSequence()
	if err != nil {
		return nil, err
	}

	result := &sensor{sb, id}
	result.init(id, seq)

	d.user.tx.db.bufferAdd <- bufferKey{d.user.dbId(), d.dbId(), result.dbId()}

	return result, nil
}

func (d *device) Sensor(id string) Sensor {
	b := d.b.Bucket(device_sensors).Bucket([]byte(id))
	if b != nil {
		return &sensor{b, id}
	}
	return nil
}

func (d *device) Sensors() map[string]Sensor {
	result := make(map[string]Sensor)
	b := d.b.Bucket(device_sensors)
	b.ForEach(func(k, v []byte) error {
		result[string(k)] = &sensor{b.Bucket(k), string(k)}
		return nil
	})
	return result
}

func (d *device) RemoveSensor(id string) error {
	sens := d.Sensor(id)
	if sens == nil {
		return InvalidId
	}
	sid := sens.dbId()
	d.b.Bucket(device_sensors).DeleteBucket([]byte(id))
	return d.user.tx.removeSeriesFor(d.user.dbId(), d.dbId(), sid)
}

func (d *device) Key() []byte {
	return d.b.Get(device_key)
}

func (d *device) Id() string {
	return d.id
}

func (d *device) dbId() uint64 {
	return letohu64(d.b.Get(device_id))
}

func (d *device) Name() string {
	return string(d.b.Get(device_name))
}

func (d *device) SetName(name string) {
	d.b.Put(device_name, []byte(name))
}

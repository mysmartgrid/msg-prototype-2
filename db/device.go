package db

import (
	"github.com/boltdb/bolt"
)

type device struct {
	b    *bolt.Bucket
	user *user
	id   string
}

func (d *device) init(key []byte, name string) {
	d.b.CreateBucketIfNotExists(dbUserDeviceSensorsKey)
	d.b.Put(dbUserDeviceKeyKey, key)
	d.b.Put(nameKey, []byte(name))
}

func (d *device) AddSensor(id string) (Sensor, error) {
	idBytes := []byte(id)
	if len(idBytes) == 0 || len(idBytes) >= bolt.MaxKeySize {
		return nil, InvalidId
	}

	b := d.b.Bucket(dbUserDeviceSensorsKey)
	sb, err := b.CreateBucket(idBytes)
	if err != nil {
		return nil, IdExists
	}

	result := &sensor{sb, id}
	result.init(id)

	d.user.tx.db.bufferAdd <- bufferKey{d.user.Id(), d.id, result.Id()}

	return result, nil
}

func (d *device) Sensor(id string) Sensor {
	b := d.b.Bucket(dbUserDeviceSensorsKey).Bucket([]byte(id))
	if b != nil {
		return &sensor{b, id}
	}
	return nil
}

func (d *device) Sensors() map[string]Sensor {
	result := make(map[string]Sensor)
	b := d.b.Bucket(dbUserDeviceSensorsKey)
	b.ForEach(func(k, v []byte) error {
		result[string(k)] = &sensor{b.Bucket(k), string(k)}
		return nil
	})
	return result
}

func (d *device) RemoveSensor(id string) error {
	if err := d.b.Bucket(dbUserDeviceSensorsKey).DeleteBucket([]byte(id)); err != nil {
		return InvalidId
	}
	return d.user.tx.removeSeriesFor(d.user.Id(), d.Id(), id)
}

func (d *device) Key() []byte {
	return d.b.Get(dbUserDeviceKeyKey)
}

func (d *device) Id() string {
	return d.id
}

func (d *device) Name() string {
	return string(d.b.Get(nameKey))
}

func (d *device) SetName(name string) {
	d.b.Put(nameKey, []byte(name))
}

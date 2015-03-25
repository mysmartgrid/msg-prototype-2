package db

import (
	"github.com/boltdb/bolt"
	"time"
)

type tx struct {
	db *db
	*bolt.Tx
}

func (tx *tx) AddUser(id string) (User, error) {
	idBytes := []byte(id)
	if len(idBytes) == 0 || len(idBytes) >= bolt.MaxKeySize {
		return nil, InvalidId
	}

	b := tx.Bucket(dbUsersKey)
	ub, err := b.CreateBucket(idBytes)
	if err != nil {
		return nil, IdExists
	}
	seq, err := b.NextSequence()
	if err != nil {
		return nil, err
	}

	result := &user{tx, ub, id}
	result.init(seq)
	return result, nil
}

func (tx *tx) User(id string) User {
	b := tx.Bucket(dbUsersKey)
	ub := b.Bucket([]byte(id))
	if ub != nil {
		return &user{tx, ub, id}
	}
	return nil
}

func (tx *tx) Users() map[string]User {
	result := make(map[string]User)
	b := tx.Bucket(dbUsersKey)
	b.ForEach(func(k, v []byte) error {
		result[string(k)] = &user{tx, b.Bucket(k), string(k)}
		return nil
	})
	return result
}

func (tx *tx) AddDevice(id string, key []byte) error {
	if tx.Bucket(registeredDevicesKey).Bucket([]byte(id)) != nil {
		return IdExists
	}
	db, err := tx.Bucket(registeredDevicesKey).CreateBucket([]byte(id))
	if err != nil {
		return err
	}
	(&registeredDevice{db, id}).init(key)
	return nil
}

func (tx *tx) Device(devId string) RegisteredDevice {
	if db := tx.Bucket(registeredDevicesKey).Bucket([]byte(devId)); db != nil {
		return &registeredDevice{db, devId}
	}
	return nil
}

func (tx *tx) Devices() map[string]RegisteredDevice {
	result := make(map[string]RegisteredDevice)
	b := tx.Bucket(registeredDevicesKey)
	b.ForEach(func(k, v []byte) error {
		result[string(k)] = &registeredDevice{b.Bucket(k), string(k)}
		return nil
	})
	return result
}

func (tx *tx) loadReadings(since time.Time, user User, sensors map[Device][]Sensor) (map[Device]map[Sensor][]Value, error) {
	keys := make([]bufferKey, 0)
	dmap := make(map[uint64]Device)
	smap := make(map[Device]map[uint64]Sensor)
	for device, sensors := range sensors {
		dmap[device.dbId()] = device
		smap[device] = make(map[uint64]Sensor)
		for _, sensor := range sensors {
			smap[device][sensor.dbId()] = sensor
			keys = append(keys, bufferKey{user.dbId(), device.dbId(), sensor.dbId()})
		}
	}

	queryResult, err := tx.db.influx.loadValues(since, keys)
	if err != nil {
		return nil, err
	}

	result := make(map[Device]map[Sensor][]Value)
	for key, values := range queryResult {
		dev := dmap[key.device]
		sensor := smap[dev][key.sensor]

		if result[dev] == nil {
			result[dev] = make(map[Sensor][]Value)
		}
		result[dev][sensor] = values
	}

	return result, nil
}

func (tx *tx) removeSeriesFor(user, device, sensor uint64) error {
	tx.db.bufferKill <- bufferKey{user, device, sensor}
	return tx.db.influx.removeSeriesFor(user, device, sensor)
}

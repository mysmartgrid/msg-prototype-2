package regdev

import (
	"github.com/boltdb/bolt"
)

type tx struct {
	db *db
	*bolt.Tx
}

func (tx *tx) AddDevice(id string, key []byte) error {
	if tx.Bucket(db_registeredDevices).Bucket([]byte(id)) != nil {
		return IdExists
	}
	db, err := tx.Bucket(db_registeredDevices).CreateBucket([]byte(id))
	if err != nil {
		return err
	}
	(&registeredDevice{db, id}).init(key)
	return nil
}

func (tx *tx) Device(devId string) RegisteredDevice {
	if db := tx.Bucket(db_registeredDevices).Bucket([]byte(devId)); db != nil {
		return &registeredDevice{db, devId}
	}
	return nil
}

func (tx *tx) Devices() map[string]RegisteredDevice {
	result := make(map[string]RegisteredDevice)
	b := tx.Bucket(db_registeredDevices)
	b.ForEach(func(k, v []byte) error {
		result[string(k)] = &registeredDevice{b.Bucket(k), string(k)}
		return nil
	})
	return result
}

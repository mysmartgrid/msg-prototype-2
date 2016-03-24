package regdev

import (
	"github.com/boltdb/bolt"
)

type tx struct {
	db *db
	*bolt.Tx
}

func (tx *tx) AddDevice(id string, key []byte) error {
	if tx.Bucket(dbRegisteredDevices).Bucket([]byte(id)) != nil {
		return ErrIDExists
	}
	db, err := tx.Bucket(dbRegisteredDevices).CreateBucket([]byte(id))
	if err != nil {
		return err
	}
	(&registeredDevice{db, id}).init(key)
	return nil
}

func (tx *tx) Device(devID string) RegisteredDevice {
	if db := tx.Bucket(dbRegisteredDevices).Bucket([]byte(devID)); db != nil {
		return &registeredDevice{db, devID}
	}
	return nil
}

func (tx *tx) Devices() map[string]RegisteredDevice {
	result := make(map[string]RegisteredDevice)
	b := tx.Bucket(dbRegisteredDevices)
	b.ForEach(func(k, v []byte) error {
		result[string(k)] = &registeredDevice{b.Bucket(k), string(k)}
		return nil
	})
	return result
}

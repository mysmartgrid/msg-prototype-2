package oldapidb

import (
	"github.com/boltdb/bolt"
)

type tx struct {
	db *db
	*bolt.Tx
}

func (tx *tx) AddSensor(id string, key []byte) error {
	if tx.Bucket(dbRegisteredSensors).Bucket([]byte(id)) != nil {
		return ErrIDExists
	}
	db, err := tx.Bucket(dbRegisteredSensors).CreateBucket([]byte(id))
	if err != nil {
		return err
	}
	(&registeredSensor{db, id}).init(key)
	return nil
}

func (tx *tx) Sensor(sensorID string) RegisteredSensor {
	if db := tx.Bucket(dbRegisteredSensors).Bucket([]byte(sensorID)); db != nil {
		return &registeredSensor{db, sensorID}
	}
	return nil
}

func (tx *tx) Sensors() map[string]RegisteredSensor {
	result := make(map[string]RegisteredSensor)
	b := tx.Bucket(dbRegisteredSensors)
	b.ForEach(func(k, v []byte) error {
		result[string(k)] = &registeredSensor{b.Bucket(k), string(k)}
		return nil
	})
	return result
}

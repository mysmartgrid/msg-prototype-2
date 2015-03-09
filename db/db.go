package db

import (
	"errors"
	"github.com/boltdb/bolt"
	"log"
	"time"
)

const (
	bufferSize = 10000
)

var (
	InvalidId     = errors.New("id invalid")
	IdExists      = errors.New("id exists")
	AlreadyLinked = errors.New("already linked")

	keyKey                 = []byte("key")
	nameKey                = []byte("name")
	userKey                = []byte("user")
	registeredDevicesKey   = []byte("registeredDevices")
	dbUsersKey             = []byte("users")
	dbUserDevicesKey       = []byte("devices")
	dbUserDeviceKeyKey     = []byte("key")
	dbUserDeviceSensorsKey = []byte("sensors")
)

type db struct {
	store *bolt.DB

	influx influxHandler

	bufferedValues     map[bufferKey][]Value
	bufferedValueCount uint32

	bufferInput chan bufferValue
	bufferAdd   chan bufferKey
	bufferKill  chan bufferKey
}

type bufferKey struct {
	user, device, sensor string
}

type bufferValue struct {
	key   bufferKey
	value Value
}

func (db *db) flushBuffer() {
	if db.bufferedValueCount == 0 {
		return
	}

	err := db.influx.saveValuesAndClear(db.bufferedValues)
	if err != nil {
		panic(err.Error())
	}

	db.bufferedValueCount = 0
}

func (d *db) manageBuffer() {
	ticker := time.NewTicker(1 * time.Second)
	defer func() {
		ticker.Stop()
		d.flushBuffer()
	}()

	for {
		select {
		case bval, ok := <-d.bufferInput:
			if !ok {
				return
			}
			slice, found := d.bufferedValues[bval.key]
			if !found {
				log.Printf("adding value to bad key %v", bval.key)
				continue
			}
			d.bufferedValues[bval.key] = append(slice, bval.value)
			d.bufferedValueCount++

			if d.bufferedValueCount >= bufferSize {
				d.flushBuffer()
			}

		case key := <-d.bufferKill:
			delete(d.bufferedValues, key)

		case key := <-d.bufferAdd:
			d.bufferedValues[key] = make([]Value, 0, 4)

		case <-ticker.C:
			d.flushBuffer()
		}
	}
}

func OpenDb(path, influxAddr, influxDb, influxUser, influxPass string) (Db, error) {
	store, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	store.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists(dbUsersKey)
		tx.CreateBucketIfNotExists(registeredDevicesKey)
		return nil
	})

	result := &db{
		store:          store,
		influx:         influxHandler{influxAddr, influxDb, influxUser, influxPass},
		bufferedValues: make(map[bufferKey][]Value),
		bufferInput:    make(chan bufferValue),
		bufferKill:     make(chan bufferKey),
		bufferAdd:      make(chan bufferKey),
	}

	go result.manageBuffer()

	result.View(func(tx Tx) error {
		for _, user := range tx.Users() {
			for _, dev := range user.Devices() {
				for _, sensor := range dev.Sensors() {
					result.bufferAdd <- bufferKey{user.Id(), dev.Id(), sensor.Id()}
				}
			}
		}
		return nil
	})

	return result, nil
}

func (d *db) Close() {
	close(d.bufferInput)
	d.store.Close()
}

func (d *db) View(fn func(Tx) error) error {
	return d.store.View(func(btx *bolt.Tx) error {
		return fn(&tx{d, btx})
	})
}

func (d *db) Update(fn func(Tx) error) error {
	return d.store.Update(func(btx *bolt.Tx) error {
		return fn(&tx{d, btx})
	})
}

func (d *db) AddReading(user User, device Device, sensor Sensor, time time.Time, value float64) error {
	d.bufferInput <- bufferValue{bufferKey{user.Id(), device.Id(), sensor.Id()}, Value{time, value}}
	return nil
}

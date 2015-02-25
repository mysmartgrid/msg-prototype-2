package msgp

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"net/http"
	"time"
)

const (
	bufferSize = 10000
)

type DbTx interface {
	AddUser(name string) (User, error)
	User(name string) User
	Users() map[string]User
}

type Db interface {
	Close()

	Update(func(DbTx) error) error
	View(func(DbTx) error) error

	AddReading(user, device, sensor string, time time.Time, value float64) error
}

type User interface {
	AddDevice(name string, key []byte) (Device, error)
	Device(name string) Device
	Devices() map[string]Device
}

type Device interface {
	AddSensor(name string) (Sensor, error)
	Sensor(name string) Sensor
	Sensors() map[string]Sensor

	Key() []byte
}

type Sensor interface {
}

type Value struct {
	Time  time.Time
	Value float64
}

var (
	InvalidName = errors.New("name invalid")
	NameExists  = errors.New("name exists")

	dbUsersKey             = []byte("users")
	dbUserDevicesKey       = []byte("devices")
	dbUserDeviceKeyKey     = []byte("key")
	dbUserDeviceSensorsKey = []byte("sensors")
)

type db struct {
	store *bolt.DB

	influxAddr, influxDb, influxUser, influxPass string

	bufferedValues     map[string][]Value
	bufferedValueCount uint32

	bufferInput chan bufferValue
}

type bufferValue struct {
	sensor string
	value  Value
}

func (db *db) flushBuffer() {
	if db.bufferedValueCount == 0 {
		return
	}

	var buf bytes.Buffer

	buf.WriteRune('[')
	for key, values := range db.bufferedValues {
		if buf.Len() > 1 {
			buf.WriteRune(',')
		}
		fmt.Fprintf(&buf, `{"name":"%v",`, key)
		buf.WriteString(`"columns":["time","value"],`)
		buf.WriteString(`"points":[`)
		for i, value := range values {
			if i > 0 {
				buf.WriteRune(',')
			}
			fmt.Fprintf(&buf, `[%v,%v]`, value.Time.Unix()*1000+int64(value.Time.Nanosecond()/1e6), value.Value)
		}
		buf.WriteString("]}")
	}
	buf.WriteRune(']')

	client := http.Client{Timeout: 1 * time.Second}
	dbUrl := fmt.Sprintf("%v/db/%v/series?time_precision=ms&u=%v&p=%v", db.influxAddr, db.influxDb, db.influxUser, db.influxPass)
	resp, err := client.Post(dbUrl, "text/plain; charset=utf-8", &buf)
	if err != nil {
		panic(err.Error())
	}
	if resp.StatusCode != 200 {
		panic(resp.Status)
	}

	db.bufferedValues = make(map[string][]Value)
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
			slice := d.bufferedValues[bval.sensor]
			if slice == nil {
				slice = make([]Value, 0)
			}
			d.bufferedValues[bval.sensor] = append(slice, bval.value)
			d.bufferedValueCount++

			if d.bufferedValueCount >= bufferSize {
				d.flushBuffer()
			}

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
		return nil
	})

	result := &db{
		store:          store,
		influxUser:     influxUser,
		influxAddr:     influxAddr,
		influxPass:     influxPass,
		influxDb:       influxDb,
		bufferedValues: make(map[string][]Value),
		bufferInput:    make(chan bufferValue),
	}
	go result.manageBuffer()
	return result, nil
}

func (d *db) Close() {
	close(d.bufferInput)
	d.store.Close()
}

func dbSensorName(user, device, sensor string) string {
	return user + "-" + device + "-" + sensor
}

func (d *db) View(fn func(DbTx) error) error {
	return d.store.View(func(tx *bolt.Tx) error {
		return fn(dbTx{tx})
	})
}

func (d *db) Update(fn func(DbTx) error) error {
	return d.store.Update(func(tx *bolt.Tx) error {
		return fn(dbTx{tx})
	})
}

func (d *db) AddReading(user, device, sensor string, time time.Time, value float64) error {
	err := d.View(func(tx DbTx) error {
		user := tx.User(user)
		if user == nil {
			return InvalidName
		}
		device := user.Device(device)
		if device == nil {
			return InvalidName
		}
		sensor := device.Sensor(sensor)
		if sensor == nil {
			return InvalidName
		}
		return nil
	})
	if err != nil {
		return err
	}
	d.bufferInput <- bufferValue{dbSensorName(user, device, sensor), Value{time, value}}
	return nil
}

type dbTx struct {
	*bolt.Tx
}

func (tx dbTx) AddUser(name string) (User, error) {
	nameBytes := []byte(name)
	if len(nameBytes) == 0 || len(nameBytes) >= bolt.MaxKeySize {
		return nil, InvalidName
	}

	b := tx.Bucket(dbUsersKey)
	ub, err := b.CreateBucket(nameBytes)
	if err != nil {
		return nil, NameExists
	}

	result := dbUser{ub}
	result.init()
	return result, nil
}

func (tx dbTx) User(name string) User {
	b := tx.Bucket(dbUsersKey)
	ub := b.Bucket([]byte(name))
	if ub != nil {
		return dbUser{ub}
	}
	return nil
}

func (tx dbTx) Users() map[string]User {
	result := make(map[string]User)
	b := tx.Bucket(dbUsersKey)
	b.ForEach(func(k, v []byte) error {
		result[string(k)] = dbUser{b.Bucket(k)}
		return nil
	})
	return result
}

type dbUser struct {
	b *bolt.Bucket
}

func (u dbUser) init() {
	u.b.CreateBucketIfNotExists(dbUserDevicesKey)
}

func (u dbUser) AddDevice(name string, key []byte) (Device, error) {
	nameBytes := []byte(name)
	if len(nameBytes) == 0 || len(nameBytes) >= bolt.MaxKeySize {
		return nil, InvalidName
	}

	b := u.b.Bucket(dbUserDevicesKey)
	db, err := b.CreateBucket(nameBytes)
	if err != nil {
		return nil, NameExists
	}

	result := dbDevice{db}
	result.init(key)
	return result, nil
}

func (d dbUser) Device(name string) Device {
	b := d.b.Bucket(dbUserDevicesKey).Bucket([]byte(name))
	if b != nil {
		return dbDevice{b}
	}
	return nil
}

func (d dbUser) Devices() map[string]Device {
	result := make(map[string]Device)
	b := d.b.Bucket(dbUserDevicesKey)
	b.ForEach(func(k, v []byte) error {
		result[string(k)] = dbDevice{b.Bucket(k)}
		return nil
	})
	return result
}

type dbDevice struct {
	b *bolt.Bucket
}

func (d dbDevice) init(key []byte) {
	d.b.CreateBucketIfNotExists(dbUserDeviceSensorsKey)
	d.b.Put(dbUserDeviceKeyKey, key)
}

func (d dbDevice) AddSensor(name string) (Sensor, error) {
	nameBytes := []byte(name)
	if len(nameBytes) == 0 || len(nameBytes) >= bolt.MaxKeySize {
		return nil, InvalidName
	}

	b := d.b.Bucket(dbUserDeviceSensorsKey)
	sb, err := b.CreateBucket(nameBytes)
	if err != nil {
		return nil, NameExists
	}

	result := dbSensor{sb}
	result.init()
	return result, nil
}

func (d dbDevice) Sensor(name string) Sensor {
	b := d.b.Bucket(dbUserDeviceSensorsKey).Bucket([]byte(name))
	if b != nil {
		return dbSensor{b}
	}
	return nil
}

func (d dbDevice) Sensors() map[string]Sensor {
	result := make(map[string]Sensor)
	b := d.b.Bucket(dbUserDeviceSensorsKey)
	b.ForEach(func(k, v []byte) error {
		result[string(k)] = dbSensor{b.Bucket(k)}
		return nil
	})
	return result
}

func (d dbDevice) Key() []byte {
	return d.b.Get(dbUserDeviceKeyKey)
}

type dbSensor struct {
	b *bolt.Bucket
}

func (s dbSensor) init() {
}

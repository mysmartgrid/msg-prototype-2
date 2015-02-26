package msgp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
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

	AddReading(user User, device Device, sensor Sensor, time time.Time, value float64) error
}

type User interface {
	AddDevice(name string, key []byte) (Device, error)
	Device(name string) Device
	Devices() map[string]Device

	Id() string

	LoadReadings(since time.Time, sensors map[Device][]Sensor) (map[Device]map[Sensor][]Value, error)
}

type Device interface {
	AddSensor(name string) (Sensor, error)
	Sensor(name string) Sensor
	Sensors() map[string]Sensor

	Id() string
	Key() []byte
}

type Sensor interface {
	Id() string
}

type Value struct {
	Time  time.Time
	Value float64
}

var (
	InvalidName = errors.New("name invalid")
	NameExists  = errors.New("name exists")

	idKey                  = []byte("id")
	dbUsersKey             = []byte("users")
	dbUserDevicesKey       = []byte("devices")
	dbUserDeviceKeyKey     = []byte("key")
	dbUserDeviceSensorsKey = []byte("sensors")
)

type influxHandler struct {
	influxAddr, influxDb, influxUser, influxPass string
}

func influxTime(t time.Time) int64 {
	return t.Unix()*1000 + int64(t.Nanosecond()/1e6)
}

func goTime(t float64) time.Time {
	return time.Unix(int64(t/1000), int64(t)%1000*1e6)
}

func (h *influxHandler) dbUrl(args map[string]string) string {
	result := fmt.Sprintf(
		"%v/db/%v/series?time_precision=ms&u=%v&p=%v",
		h.influxAddr,
		url.QueryEscape(h.influxDb),
		url.QueryEscape(h.influxUser),
		url.QueryEscape(h.influxPass))

	for key, value := range args {
		result += fmt.Sprintf("&%v=%v", url.QueryEscape(key), url.QueryEscape(value))
	}

	return result
}

func (h *influxHandler) saveValues(values map[bufferKey][]Value) error {
	var buf bytes.Buffer

	buf.WriteRune('[')
	for key, values := range values {
		if buf.Len() > 1 {
			buf.WriteRune(',')
		}
		fmt.Fprintf(&buf, `{"name":"%v-%v-%v",`, key.user, key.device, key.sensor)
		buf.WriteString(`"columns":["time","value"],`)
		buf.WriteString(`"points":[`)
		for i, value := range values {
			if i > 0 {
				buf.WriteRune(',')
			}
			fmt.Fprintf(&buf, `[%v,%v]`, influxTime(value.Time), value.Value)
		}
		buf.WriteString("]}")
	}
	buf.WriteRune(']')

	client := http.Client{Timeout: 1 * time.Second}
	resp, err := client.Post(h.dbUrl(nil), "application/json; charset=utf-8", &buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		data, _ := ioutil.ReadAll(resp.Body)
		return errors.New(resp.Status + " " + string(data))
	}
	return nil
}

func (h *influxHandler) loadValues(since time.Time, keys []bufferKey) (map[bufferKey][]Value, error) {
	type inputSeries struct {
		Name   string       `json:"name"`
		Points [][3]float64 `json:"points"`
	}

	var queryResult []inputSeries

	series := make([]string, 0, len(keys))
	for _, key := range keys {
		series = append(series, fmt.Sprintf("%v-%v-%v", key.user, key.device, key.sensor))
	}
	query := fmt.Sprintf("select * from /%v/ where time > %v", strings.Join(series, "|"), influxTime(since))

	client := http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(h.dbUrl(map[string]string{"q": query}))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		data, _ := ioutil.ReadAll(resp.Body)
		return nil, errors.New(resp.Status + " " + string(data))
	}

	err = json.NewDecoder(resp.Body).Decode(&queryResult)
	if err != nil {
		return nil, err
	}

	result := make(map[bufferKey][]Value, len(keys))
	for _, series := range queryResult {
		parts := strings.Split(series.Name, "-")
		key := bufferKey{parts[0], parts[1], parts[2]}
		values := make([]Value, 0, len(series.Points))
		for _, point := range series.Points {
			values = append(values, Value{goTime(point[0]), point[2]})
		}
		result[key] = values
	}

	return result, nil
}

type db struct {
	store *bolt.DB

	influx influxHandler

	bufferedValues     map[bufferKey][]Value
	bufferedValueCount uint32

	bufferInput chan bufferValue
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

	err := db.influx.saveValues(db.bufferedValues)
	if err != nil {
		panic(err.Error())
	}

	db.bufferedValues = make(map[bufferKey][]Value)
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
			slice := d.bufferedValues[bval.key]
			if slice == nil {
				slice = make([]Value, 0)
			}
			d.bufferedValues[bval.key] = append(slice, bval.value)
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
		influx:         influxHandler{influxAddr, influxDb, influxUser, influxPass},
		bufferedValues: make(map[bufferKey][]Value),
		bufferInput:    make(chan bufferValue),
	}
	go result.manageBuffer()
	return result, nil
}

func (d *db) Close() {
	close(d.bufferInput)
	d.store.Close()
}

func (d *db) View(fn func(DbTx) error) error {
	return d.store.View(func(tx *bolt.Tx) error {
		return fn(&dbTx{d, tx})
	})
}

func (d *db) Update(fn func(DbTx) error) error {
	return d.store.Update(func(tx *bolt.Tx) error {
		return fn(&dbTx{d, tx})
	})
}

func (d *db) AddReading(user User, device Device, sensor Sensor, time time.Time, value float64) error {
	d.bufferInput <- bufferValue{bufferKey{user.Id(), device.Id(), sensor.Id()}, Value{time, value}}
	return nil
}

type dbTx struct {
	db *db
	*bolt.Tx
}

func (tx *dbTx) AddUser(name string) (User, error) {
	nameBytes := []byte(name)
	if len(nameBytes) == 0 || len(nameBytes) >= bolt.MaxKeySize {
		return nil, InvalidName
	}

	b := tx.Bucket(dbUsersKey)
	ub, err := b.CreateBucket(nameBytes)
	if err != nil {
		return nil, NameExists
	}
	seq, _ := b.NextSequence()

	result := dbUser{tx, ub}
	result.init(seq)
	return result, nil
}

func (tx *dbTx) User(name string) User {
	b := tx.Bucket(dbUsersKey)
	ub := b.Bucket([]byte(name))
	if ub != nil {
		return dbUser{tx, ub}
	}
	return nil
}

func (tx *dbTx) Users() map[string]User {
	result := make(map[string]User)
	b := tx.Bucket(dbUsersKey)
	b.ForEach(func(k, v []byte) error {
		result[string(k)] = dbUser{tx, b.Bucket(k)}
		return nil
	})
	return result
}

func (tx *dbTx) loadReadings(since time.Time, user User, sensors map[Device][]Sensor) (map[Device]map[Sensor][]Value, error) {
	keys := make([]bufferKey, 0)
	dmap := make(map[string]Device)
	smap := make(map[Device]map[string]Sensor)
	for device, sensors := range sensors {
		dmap[device.Id()] = device
		smap[device] = make(map[string]Sensor)
		for _, sensor := range sensors {
			smap[device][sensor.Id()] = sensor
			keys = append(keys, bufferKey{user.Id(), device.Id(), sensor.Id()})
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

type dbUser struct {
	tx *dbTx
	b  *bolt.Bucket
}

func (u dbUser) init(seq uint64) {
	u.b.CreateBucketIfNotExists(dbUserDevicesKey)
	u.b.Put(idKey, []byte(fmt.Sprintf("uid%v", seq)))
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
	seq, _ := b.NextSequence()

	result := dbDevice{db}
	result.init(key, seq)
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

func (d dbUser) Id() string {
	return string(d.b.Get(idKey))
}

func (d dbUser) LoadReadings(since time.Time, sensors map[Device][]Sensor) (map[Device]map[Sensor][]Value, error) {
	return d.tx.loadReadings(since, d, sensors)
}

type dbDevice struct {
	b *bolt.Bucket
}

func (d dbDevice) init(key []byte, seq uint64) {
	d.b.CreateBucketIfNotExists(dbUserDeviceSensorsKey)
	d.b.Put(dbUserDeviceKeyKey, key)
	d.b.Put(idKey, []byte(fmt.Sprintf("did%v", seq)))
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
	seq, _ := b.NextSequence()

	result := dbSensor{sb}
	result.init(seq)
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

func (d dbDevice) Id() string {
	return string(d.b.Get(idKey))
}

type dbSensor struct {
	b *bolt.Bucket
}

func (s dbSensor) init(seq uint64) {
	s.b.Put(idKey, []byte(fmt.Sprintf("sid%v", seq)))
}

func (s dbSensor) Id() string {
	return string(s.b.Get(idKey))
}

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
	AddUser(id string) (User, error)
	User(id string) User
	Users() map[string]User
}

type Db interface {
	Close()

	Update(func(DbTx) error) error
	View(func(DbTx) error) error

	AddReading(user User, device Device, sensor Sensor, time time.Time, value float64) error
}

type User interface {
	AddDevice(id string, key []byte) (Device, error)
	Device(id string) Device
	Devices() map[string]Device

	Id() string

	LoadReadings(since time.Time, sensors map[Device][]Sensor) (map[Device]map[Sensor][]Value, error)
}

type Device interface {
	AddSensor(id string) (Sensor, error)
	Sensor(id string) Sensor
	Sensors() map[string]Sensor
	RemoveSensor(id string) error

	Id() string
	Key() []byte

	Name() string
	SetName(string)
}

type Sensor interface {
	Id() string

	Name() string
	SetName(string)
}

type Value struct {
	Time  time.Time
	Value float64
}

var (
	InvalidId = errors.New("id invalid")
	IdExists  = errors.New("id exists")

	nameKey                = []byte("name")
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
	query := fmt.Sprintf("select * from /%v/ where time > %vms", strings.Join(series, "|"), influxTime(since))

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

func (h *influxHandler) removeSeriesFor(user, device, sensor string) error {
	query := fmt.Sprintf("drop series \"%v-%v-%v\"", user, device, sensor)

	client := http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(h.dbUrl(map[string]string{"q": query}))
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

type db struct {
	store *bolt.DB

	influx influxHandler

	bufferedValues     map[bufferKey][]Value
	bufferedValueCount uint32

	bufferInput chan bufferValue
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

		case key := <-d.bufferKill:
			delete(d.bufferedValues, key)

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
		bufferKill:     make(chan bufferKey),
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

func (tx *dbTx) AddUser(id string) (User, error) {
	idBytes := []byte(id)
	if len(idBytes) == 0 || len(idBytes) >= bolt.MaxKeySize {
		return nil, InvalidId
	}

	b := tx.Bucket(dbUsersKey)
	ub, err := b.CreateBucket(idBytes)
	if err != nil {
		return nil, IdExists
	}

	result := &dbUser{tx, ub, id}
	result.init()
	return result, nil
}

func (tx *dbTx) User(id string) User {
	b := tx.Bucket(dbUsersKey)
	ub := b.Bucket([]byte(id))
	if ub != nil {
		return &dbUser{tx, ub, id}
	}
	return nil
}

func (tx *dbTx) Users() map[string]User {
	result := make(map[string]User)
	b := tx.Bucket(dbUsersKey)
	b.ForEach(func(k, v []byte) error {
		result[string(k)] = &dbUser{tx, b.Bucket(k), string(k)}
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

func (tx *dbTx) removeSeriesFor(user, device, sensor string) error {
	tx.db.bufferKill <- bufferKey{user, device, sensor}
	return tx.db.influx.removeSeriesFor(user, device, sensor)
}

type dbUser struct {
	tx *dbTx
	b  *bolt.Bucket
	id string
}

func (u *dbUser) init() {
	u.b.CreateBucketIfNotExists(dbUserDevicesKey)
}

func (u *dbUser) AddDevice(id string, key []byte) (Device, error) {
	idBytes := []byte(id)
	if len(idBytes) == 0 || len(idBytes) >= bolt.MaxKeySize {
		return nil, InvalidId
	}

	b := u.b.Bucket(dbUserDevicesKey)
	db, err := b.CreateBucket(idBytes)
	if err != nil {
		return nil, IdExists
	}

	result := &dbDevice{db, u, id}
	result.init(key, id)
	return result, nil
}

func (d *dbUser) Device(id string) Device {
	b := d.b.Bucket(dbUserDevicesKey).Bucket([]byte(id))
	if b != nil {
		return &dbDevice{b, d, id}
	}
	return nil
}

func (d *dbUser) Devices() map[string]Device {
	result := make(map[string]Device)
	b := d.b.Bucket(dbUserDevicesKey)
	b.ForEach(func(k, v []byte) error {
		result[string(k)] = &dbDevice{b.Bucket(k), d, string(k)}
		return nil
	})
	return result
}

func (d *dbUser) Id() string {
	return d.id
}

func (d *dbUser) LoadReadings(since time.Time, sensors map[Device][]Sensor) (map[Device]map[Sensor][]Value, error) {
	return d.tx.loadReadings(since, d, sensors)
}

type dbDevice struct {
	b    *bolt.Bucket
	user *dbUser
	id   string
}

func (d *dbDevice) init(key []byte, name string) {
	d.b.CreateBucketIfNotExists(dbUserDeviceSensorsKey)
	d.b.Put(dbUserDeviceKeyKey, key)
	d.b.Put(nameKey, []byte(name))
}

func (d *dbDevice) AddSensor(id string) (Sensor, error) {
	idBytes := []byte(id)
	if len(idBytes) == 0 || len(idBytes) >= bolt.MaxKeySize {
		return nil, InvalidId
	}

	b := d.b.Bucket(dbUserDeviceSensorsKey)
	sb, err := b.CreateBucket(idBytes)
	if err != nil {
		return nil, IdExists
	}

	result := &dbSensor{sb, id}
	result.init(id)
	return result, nil
}

func (d *dbDevice) Sensor(id string) Sensor {
	b := d.b.Bucket(dbUserDeviceSensorsKey).Bucket([]byte(id))
	if b != nil {
		return &dbSensor{b, id}
	}
	return nil
}

func (d *dbDevice) Sensors() map[string]Sensor {
	result := make(map[string]Sensor)
	b := d.b.Bucket(dbUserDeviceSensorsKey)
	b.ForEach(func(k, v []byte) error {
		result[string(k)] = &dbSensor{b.Bucket(k), string(k)}
		return nil
	})
	return result
}

func (d *dbDevice) RemoveSensor(id string) error {
	if err := d.b.Bucket(dbUserDeviceSensorsKey).DeleteBucket([]byte(id)); err != nil {
		return InvalidId
	}
	return d.user.tx.removeSeriesFor(d.user.Id(), d.Id(), id)
}

func (d *dbDevice) Key() []byte {
	return d.b.Get(dbUserDeviceKeyKey)
}

func (d *dbDevice) Id() string {
	return d.id
}

func (d *dbDevice) Name() string {
	return string(d.b.Get(nameKey))
}

func (d *dbDevice) SetName(name string) {
	d.b.Put(nameKey, []byte(name))
}

type dbSensor struct {
	b  *bolt.Bucket
	id string
}

func (s *dbSensor) init(name string) {
	s.b.Put(nameKey, []byte(name))
}

func (s *dbSensor) Id() string {
	return s.id
}

func (s *dbSensor) Name() string {
	return string(s.b.Get(nameKey))
}

func (s *dbSensor) SetName(name string) {
	s.b.Put(nameKey, []byte(name))
}

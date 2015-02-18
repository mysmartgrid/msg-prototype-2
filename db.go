package msgp

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"html/template"
	"net/http"
	"net/url"
	"time"
)

const (
	bufferSize = 10000
)

type Db struct {
	influxAddr, influxUser, influxPass string

	store *bolt.DB

	bufferedValues     map[bufferKey][]Value
	bufferedValueCount uint32

	bufferInput chan bufferValue
}

type DbTx struct {
	tx *bolt.Tx
}

type User struct {
	Name      string
	AuthToken string

	Sensors []Sensor

	db *Db
}

type Sensor struct {
	Name      string
	AuthToken string
}

type Value struct {
	Time  int64
	Value float64
}

type bufferKey struct {
	User, Sensor string
}

type bufferValue struct {
	key   bufferKey
	value Value
}

var (
	InvalidName = errors.New("name invalid")
	NameExists  = errors.New("name exists")

	userBucket   = []byte("user")
	nameKey      = []byte("name")
	authtokenKey = []byte("authtoken")
	sensorsKey   = []byte("sensors")
)

func dbSensorName(user, sensor string) string {
	return fmt.Sprintf("u%v%v-s%v%v", len(user), user, len(sensor), sensor)
}

type bufWriter struct {
	buf *bytes.Buffer
}

func (buf bufWriter) Write(b []byte) (int, error) {
	return buf.buf.Write(b)
}

func (db *Db) flushBuffer() {
	if db.bufferedValueCount == 0 {
		return
	}

	var buf bytes.Buffer
	writer := bufWriter{&buf}

	buf.WriteRune('[')
	for key, values := range db.bufferedValues {
		if buf.Len() > 1 {
			buf.WriteRune(',')
		}
		fmt.Fprintf(writer, `{"name":"%v",`, template.JSEscapeString(dbSensorName(key.User, key.Sensor)))
		buf.WriteString(`"columns":["time","value"],`)
		buf.WriteString(`"points":[`)
		for i, value := range values {
			if i > 0 {
				buf.WriteRune(',')
			}
			fmt.Fprintf(writer, `[%v,%v]`, value.Time, value.Value)
		}
		buf.WriteString("]}")
	}
	buf.WriteRune(']')

	bufReader := bytes.NewReader(buf.Bytes())

	client := http.Client{Timeout: 1 * time.Second}
	resp, err := client.Post(db.influxAddr+"/db/msgp/series?time_precision=s&u="+url.QueryEscape(db.influxUser)+
		"&p="+url.QueryEscape(db.influxPass), "text/plain; charset=utf-8", bufReader)
	if err != nil {
		panic(err.Error())
	}
	if resp.StatusCode != 200 {
		panic(resp.Status)
	}

	db.bufferedValues = make(map[bufferKey][]Value)
	db.bufferedValueCount = 0
}

func (db *Db) manageDbBuffer() {
	ticker := time.NewTicker(1 * time.Second)
	defer func() {
		ticker.Stop()
		db.flushBuffer()
	}()

	for {
		select {
		case bval, ok := <-db.bufferInput:
			if !ok {
				return
			}
			slice := db.bufferedValues[bval.key]
			if slice == nil {
				slice = make([]Value, 0)
			}
			db.bufferedValues[bval.key] = append(slice, bval.value)
			db.bufferedValueCount++

			if db.bufferedValueCount >= bufferSize {
				db.flushBuffer()
			}

		case <-ticker.C:
			db.flushBuffer()
		}
	}
}

func OpenDb(path, influxAddr, influxUser, influxPass string) (*Db, error) {
	store, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}

	err = store.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(userBucket); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	result := &Db{
		influxUser:     influxUser,
		influxAddr:     influxAddr,
		influxPass:     influxPass,
		store:          store,
		bufferedValues: make(map[bufferKey][]Value),
		bufferInput:    make(chan bufferValue),
	}

	go result.manageDbBuffer()
	return result, nil
}

func (db *Db) Close() {
	close(db.bufferInput)
	db.store.Close()
}

func newToken() string {
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		panic(errors.New("RNG bad"))
	}
	return fmt.Sprintf("%x", token)
}

func (db *Db) Add(name string) (*User, error) {
	var result *User
	err := db.store.Update(func(tx *bolt.Tx) error {
		nameBytes := []byte(name)

		if len(nameBytes) == 0 || len(nameBytes) >= bolt.MaxKeySize {
			return InvalidName
		}

		result = &User{
			Name:      name,
			AuthToken: newToken(),
			db:        db,
		}

		ub := tx.Bucket(userBucket)
		b, err := ub.CreateBucket(nameBytes)

		if err != nil {
			return NameExists
		}

		b.Put(nameKey, nameBytes)
		b.Put(authtokenKey, []byte(result.AuthToken))
		b.CreateBucket(sensorsKey)
		return nil
	})
	if err != nil {
		result = nil
	}
	return result, err
}

func loadSensors(b *bolt.Bucket) []Sensor {
	result := make([]Sensor, 0)
	b.ForEach(func(key, value []byte) error {
		result = append(result, Sensor{string(key), string(value)})
		return nil
	})
	return result
}

func loadUser(db *Db, b *bolt.Bucket) *User {
	return &User{
		Name:      string(b.Get(nameKey)),
		AuthToken: string(b.Get(authtokenKey)),
		Sensors:   loadSensors(b.Bucket(sensorsKey)),
		db:        db,
	}
}

func (db *Db) ForEach(fn func(*User) error) error {
	return db.store.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(userBucket)
		return b.ForEach(func(key, value []byte) error {
			return fn(loadUser(db, b.Bucket(key)))
		})
	})
}

func (db *Db) Find(name string) *User {
	var result *User

	db.store.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(userBucket).Bucket([]byte(name))
		if b == nil {
			return InvalidName
		}
		result = loadUser(db, b)
		return nil
	})
	return result
}

func (db *Db) Update(fn func(DbTx) error) error {
	return db.store.Update(func(tx *bolt.Tx) error {
		return fn(DbTx{tx})
	})
}

func (db *Db) CheckAuthTokenFor(userName, sensorName, token string) bool {
	err := db.store.View(func(tx *bolt.Tx) error {
		ub := tx.Bucket(userBucket)
		user := ub.Bucket([]byte(userName))
		if user == nil {
			return InvalidName
		}
		storedToken := string(user.Bucket(sensorsKey).Get([]byte(sensorName)))
		if token != storedToken {
			return InvalidName
		}
		return nil
	})
	return err == nil
}

func (db *Db) AddSensor(user, name string) (Sensor, error) {
	var result Sensor
	err := db.store.Update(func(tx *bolt.Tx) error {
		ub := tx.Bucket(userBucket)
		b := ub.Bucket([]byte(user))
		if b == nil {
			return InvalidName
		}
		sensors := b.Bucket(sensorsKey)
		if sensors.Get([]byte(name)) != nil {
			return NameExists
		}
		result = Sensor{name, newToken()}
		sensors.Put([]byte(name), []byte(result.AuthToken))
		return nil
	})
	return result, err
}

func (db *Db) AddReading(user, sensor string, time int64, value float64) {
	db.bufferInput <- bufferValue{
		key:   bufferKey{user, sensor},
		value: Value{time, value},
	}
}

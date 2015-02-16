package msgp

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"time"
)

type Db struct {
	store *bolt.DB
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

var (
	InvalidName  = errors.New("name invalid")
	NameExists   = errors.New("name exists")

	userBucket   = []byte("user")
	nameKey      = []byte("name")
	authtokenKey = []byte("authtoken")
	sensorsKey   = []byte("sensors")
)

func OpenDb(path string) (*Db, error) {
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

	return &Db{
		store: store,
	}, nil
}

func (db *Db) Close() {
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

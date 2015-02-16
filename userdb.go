package msgp

import (
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"time"
)

type UserDb struct {
	store *bolt.DB
}

type User struct {
	Name      string
	AuthToken string

	Sensors map[string]bool

	db *UserDb
}

var (
	InvalidName = errors.New("name invalid")
	NameExists  = errors.New("name exists")

	userBucket   = []byte("user")
	nameKey      = []byte("name")
	authtokenKey = []byte("authtoken")
	sensorsKey   = []byte("sensors")
)

func OpenUserDb(path string) (*UserDb, error) {
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

	return &UserDb{
		store: store,
	}, nil
}

func (db *UserDb) Close() {
	db.store.Close()
}

func (db *UserDb) Add(name string) (*User, error) {
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return nil, err
	}

	var result *User
	err := db.store.Update(func(tx *bolt.Tx) error {
		result = &User{
			Name:      name,
			AuthToken: fmt.Sprintf("%x", token),
			db:        db,
		}
		return saveUser(tx, result, false)
	})
	if err != nil {
		result = nil
	}
	return result, err
}

func saveUser(tx *bolt.Tx, u *User, update bool) error {
	var nameBytes = []byte(u.Name)

	if len(nameBytes) == 0 || len(nameBytes) >= bolt.MaxKeySize {
		return InvalidName
	}

	ub := tx.Bucket(userBucket)
	if update {
		err := ub.DeleteBucket(nameBytes)
		switch err {
		case nil:
		case bolt.ErrBucketNotFound:

		default:
			return err
		}
	}

	b, err := ub.CreateBucket(nameBytes)
	if err != nil {
		return NameExists
	}

	b.Put(nameKey, []byte(u.Name))
	b.Put(authtokenKey, []byte(u.AuthToken))

	sb, _ := b.CreateBucket(sensorsKey)
	for sensor := range u.Sensors {
		sb.Put([]byte(sensor), []byte{})
	}

	return nil
}

func loadSensors(b *bolt.Bucket) map[string]bool {
	result := make(map[string]bool)
	b.ForEach(func(key, value []byte) error {
		result[string(key)] = true
		return nil
	})
	return result
}

func loadUser(db *UserDb, b *bolt.Bucket) *User {
	return &User{
		Name:      string(b.Get(nameKey)),
		AuthToken: string(b.Get(authtokenKey)),
		Sensors:   loadSensors(b.Bucket(sensorsKey)),
		db:        db,
	}
}

func (db *UserDb) ForEach(fn func(*User) error) error {
	return db.store.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(userBucket)
		return b.ForEach(func(key, value []byte) error {
			return fn(loadUser(db, b.Bucket(key)))
		})
	})
}

func (db *UserDb) Find(name string) *User {
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

func (db *UserDb) Update(name string, fn func(*User) error) error {
	return db.store.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(userBucket).Bucket([]byte(name))
		if b == nil {
			return InvalidName
		}
		u := loadUser(db, b)
		if err := fn(u); err != nil {
			return err
		}
		return saveUser(tx, u, true)
	})
}

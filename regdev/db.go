package regdev

import (
	"errors"
	"github.com/boltdb/bolt"
)

var (
	InvalidId     = errors.New("id invalid")
	IdExists      = errors.New("id exists")
	AlreadyLinked = errors.New("already linked")

	db_registeredDevices = []byte("registeredDevices")
)

type db struct {
	store *bolt.DB
}

func Open(path string) (Db, error) {
	store, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	store.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists(db_registeredDevices)
		return nil
	})

	result := &db{
		store: store,
	}

	return result, nil
}

func (d *db) Close() {
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

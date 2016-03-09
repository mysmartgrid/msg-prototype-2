package regdev

import (
	"errors"
	"github.com/boltdb/bolt"
)

var (
	// ErrIDExists is returned after an attempt to insert a new object into the DB using an id which already exists in the DB.
	ErrIDExists = errors.New("id exists")
	// ErrAlreadyLinked is returned when trying to link a user to a device already associated with a user.
	ErrAlreadyLinked = errors.New("already linked")

	dbRegisteredDevices = []byte("registeredDevices")
)

type db struct {
	store *bolt.DB
}

// Open opens the BoltDB database containing the device database in
// the file located at path.
func Open(path string) (Db, error) {
	store, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}

	store.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists(dbRegisteredDevices)
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

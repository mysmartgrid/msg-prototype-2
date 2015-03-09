package db

import (
	"github.com/boltdb/bolt"
)

type registeredDevice struct {
	b  *bolt.Bucket
	id string
}

func (r *registeredDevice) init(key []byte) {
	r.b.Put(keyKey, key)
}

func (r *registeredDevice) Id() string {
	return r.id
}

func (r *registeredDevice) Key() []byte {
	return r.b.Get(keyKey)
}

func (r *registeredDevice) UserLink() (string, bool) {
	if uid := r.b.Get(userKey); uid != nil {
		return string(uid), true
	}
	return "", false
}

func (r *registeredDevice) LinkTo(uid string) error {
	if r.b.Get(userKey) != nil {
		return AlreadyLinked
	}
	return r.b.Put(userKey, []byte(uid))
}

func (r *registeredDevice) Unlink() error {
	return r.b.Delete(userKey)
}

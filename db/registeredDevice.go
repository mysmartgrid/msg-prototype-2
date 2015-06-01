package db

import (
	"github.com/boltdb/bolt"
)

type registeredDevice struct {
	b  *bolt.Bucket
	id string
}

var (
	registeredDevice_key  = []byte("key")
	registeredDevice_user = []byte("user")
)

func (r *registeredDevice) init(key []byte) {
	r.b.Put(registeredDevice_key, key)
}

func (r *registeredDevice) Id() string {
	return r.id
}

func (r *registeredDevice) Key() []byte {
	return r.b.Get(registeredDevice_key)
}

func (r *registeredDevice) UserLink() (string, bool) {
	if uid := r.b.Get(registeredDevice_user); uid != nil {
		return string(uid), true
	}
	return "", false
}

func (r *registeredDevice) LinkTo(uid string) error {
	if r.b.Get(registeredDevice_user) != nil {
		return AlreadyLinked
	}
	return r.b.Put(registeredDevice_user, []byte(uid))
}

func (r *registeredDevice) Unlink() error {
	return r.b.Delete(registeredDevice_user)
}

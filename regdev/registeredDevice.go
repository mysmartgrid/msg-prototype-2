package regdev

import (
	"encoding/json"
	"github.com/boltdb/bolt"
)

type registeredDevice struct {
	b  *bolt.Bucket
	id string
}

var (
	registeredDevice_key     = []byte("key")
	registeredDevice_user    = []byte("user")
	registeredDevice_network = []byte("network")
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

func (r *registeredDevice) GetNetworkConfig() DeviceConfigNetwork {
	var result DeviceConfigNetwork

	data := r.b.Get(registeredDevice_network)
	if data == nil || json.Unmarshal(data, &result) != nil {
		return DeviceConfigNetwork{}
	}

	return result
}

func (r *registeredDevice) SetNetworkConfig(conf *DeviceConfigNetwork) error {
	data, err := json.Marshal(conf)
	if err != nil {
		return err
	}
	return r.b.Put(registeredDevice_network, data)
}

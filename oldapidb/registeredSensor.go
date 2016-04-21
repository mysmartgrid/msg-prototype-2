package oldapidb

import (
//	"errors"
	"github.com/boltdb/bolt"
)

type registeredSensor struct {
	b  *bolt.Bucket
	id string
}

var (
	registeredSensorLastValue = []byte("lastvalue")
	registeredSensorTimestamp = []byte("lasttimestamp")
)

func (r* registeredSensor) init(key []byte) {
	//r.b.Put(registeredSensorKey, key)
	r.b.CreateBucket(registeredSensorTimestamp)
}

func (r *registeredSensor) ID() string {
	return r.id
}

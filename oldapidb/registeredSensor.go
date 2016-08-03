package oldapidb

import (
//	"errors"
//	"time"
)

type registeredSensor struct {
	id string
}

var (
	registeredSensorLastValue = []byte("lastvalue")
	registeredSensorTimestamp = []byte("lasttimestamp")
)

func (r* registeredSensor) init(lasttimestamp []byte, lastvalue []byte) {
}

func (r *registeredSensor) ID() string {
	return r.id
}

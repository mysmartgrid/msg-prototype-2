package oldapidb


import (
//	"encoding/json"
//	"time"
)

// Tx provides a set of operations to interact with the database through the Db's 'Open' and 'View' methods.
type Tx interface {
	AddSensor(id string, key []byte) error
	Sensor(sensorID string) RegisteredSensor
//	Sensors() map[string]RegisteredSensor
}

type RegisteredSensor interface {
	// Id returns the unique device id identifying the device.
	ID() string
	
}

// Db provides methods the interact with the device database.
type Db interface {
	// Close closes the connection to the underlying BoltDB database.
	Close()

	// Update executes a database transaction defined by a series of operations
	// in the function fn on its Tx struct.
	Update(func(Tx) error) error
	// View executes a read only database transaction defined by a series of operations
	// in the function fn on its Tx struct.
	View(func(Tx) error) error

	//
	AddLastValue(sensor Sensor, time time.Time, value float64) error
}

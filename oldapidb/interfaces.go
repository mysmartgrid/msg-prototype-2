package oldapidb


import (
	"time"
)

// Tx provides a set of operations to interact with the database through the Db's 'Open' and 'View' methods.
type Tx interface {
	AddLastValue(sensorID string, Time time.Time, value float64) error
	GetLastValue(sensorID string) (time.Time, float64, error)
	Sensor(sensorID string) RegisteredSensor
}

type RegisteredSensor interface {
	// Id returns the unique device id identifying the device.
	ID() string
	
}

// Db provides methods the interact with the device database.
type Db interface {
	// Update executes a database transaction defined by a series of operations
	// in the function fn on its Tx struct.
	Update(func(Tx) error) error
	// View executes a read only database transaction defined by a series of operations
	// in the function fn on its Tx struct.
	View(func(Tx) error) error
}

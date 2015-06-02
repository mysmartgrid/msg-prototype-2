package db

import "time"

type Tx interface {
	AddUser(id string) (User, error)
	User(id string) User
	Users() map[string]User
}

type Db interface {
	Close()

	Update(func(Tx) error) error
	View(func(Tx) error) error

	AddReading(user User, device Device, sensor Sensor, time time.Time, value float64) error
}

type User interface {
	AddDevice(id string, key []byte) (Device, error)
	RemoveDevice(id string) error
	Device(id string) Device
	Devices() map[string]Device

	Id() string
	dbId() uint64

	LoadReadings(since time.Time, sensors map[Device][]Sensor) (map[Device]map[Sensor][]Value, error)
}

type Device interface {
	AddSensor(id string) (Sensor, error)
	Sensor(id string) Sensor
	Sensors() map[string]Sensor
	RemoveSensor(id string) error

	Id() string
	dbId() uint64
	Key() []byte

	Name() string
	SetName(string)
}

type Sensor interface {
	Id() string
	dbId() uint64

	Name() string
	SetName(string)
}

type Value struct {
	Time  time.Time
	Value float64
}

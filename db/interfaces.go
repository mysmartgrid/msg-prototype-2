package db

import "time"

type Tx interface {
	AddUser(id string) (User, error)
	User(id string) User
	Users() map[string]User

	AddDevice(id string, key []byte) error
	Device(devId string) RegisteredDevice
	Devices() map[string]RegisteredDevice
}

type RegisteredDevice interface {
	Id() string
	Key() []byte
	UserLink() (string, bool)

	LinkTo(uid string) error
	Unlink() error
}

type Db interface {
	Close()

	Update(func(Tx) error) error
	View(func(Tx) error) error

	AddReading(user User, device Device, sensor Sensor, time time.Time, value float64) error
}

type User interface {
	AddDevice(id string, key []byte) (Device, error)
	Device(id string) Device
	Devices() map[string]Device

	Id() string

	LoadReadings(since time.Time, sensors map[Device][]Sensor) (map[Device]map[Sensor][]Value, error)
}

type Device interface {
	AddSensor(id string) (Sensor, error)
	Sensor(id string) Sensor
	Sensors() map[string]Sensor
	RemoveSensor(id string) error

	Id() string
	Key() []byte

	Name() string
	SetName(string)
}

type Sensor interface {
	Id() string

	Name() string
	SetName(string)
}

type Value struct {
	Time  time.Time
	Value float64
}

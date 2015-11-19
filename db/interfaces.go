package db

import "time"

type Tx interface {
	AddUser(id, password string) (User, error)
	RemoveUser(id string) error
	User(id string) User
	Users() map[string]User
	AddGroup(id string) (Group, error)
	RemoveGroup(id string) error
	Group(id string) Group
	Groups() map[string]Group
}

type Db interface {
	Close()

	Update(func(Tx) error) error
	View(func(Tx) error) error

	AddReading(sensor Sensor, time time.Time, value float64) error
	SetRealtimeHandler(handler func(values map[string]map[string]map[string]map[string][]Value))
	RequestRealtimeUpdates(user, device, resolution string, sensors []string) error

	Run()

	RunBenchmark(usrCount, devCnt, snsCnt int, duration time.Duration)
}

type User interface {
	AddDevice(id string, key []byte) (Device, error)
	RemoveDevice(id string) error
	Device(id string) Device
	Devices() map[string]Device
	HasPassword(pw string) bool

	IsAdmin() bool
	SetAdmin(b bool) error

	Groups() map[string]Group
	IsGroupAdmin(groupId string) bool

	Id() string

	LoadReadings(since, until time.Time, resolution string, sensors map[Device][]Sensor) (map[Device]map[Sensor][]Value, error)
}

type Group interface {
	AddUser(id string) error
	RemoveUser(id string) error
	GetUsers() map[string]User

	SetAdmin(id string) error
	UnsetAdmin(id string) error
	GetAdmins() map[string]User

	AddSensor(dbid uint64) error
	RemoveSensor(dbid uint64) error
	GetSensors() []uint64

	Id() string
}

type Device interface {
	AddSensor(id, unit string, port int32) (Sensor, error)
	Sensor(id string) Sensor
	Sensors() map[string]Sensor
	RemoveSensor(id string) error

	Id() string
	User() User
	Key() []byte

	Name() string
	SetName(string) error
}

type Sensor interface {
	Id() string
	DbId() uint64
	Device() Device

	Name() string
	SetName(string) error

	Groups() map[string]Group

	Port() int32
	Unit() string
}

type Value struct {
	Time  time.Time
	Value float64
}

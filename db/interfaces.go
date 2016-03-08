package db

import (
	"github.com/mysmartgrid/msg2api"
	"time"
)

// Db defines the interface to a database containing users, device, sensors and their measurment data.
type Db interface {
	// Close closes the database connection and its buffer, writing back all
	// buffered values to the database.
	Close()

	// Update executes a database transaction defined by a series of operations
	// in the function fn on its Tx struct.
	Update(func(Tx) error) error
	// View executes a read only database transaction defined by a series of operations
	// in the function fn on its Tx struct.
	View(func(Tx) error) error

	// AddReading adds a single measurment of a specific sensor to the database buffer.
	AddReading(sensor Sensor, time time.Time, value float64) error

	RunBenchmark(usrCount, devCnt, snsCnt int, duration time.Duration)
}

// Tx provides a set of operations on the top level of the database hirarchy
// It is meant to be used within the Update and View functions of a Db struct.
type Tx interface {
	// Add user adds a new user with id and password to the database and return the representing struct.
	// The password is provided in plain text and is hashed before it is written to the database.
	// Returns an error if the user id already exists in the database.
	AddUser(id, password string) (User, error)

	// RemoveUser removes a user from the database.
	// The database itself should also remvove all devices, sensors and measurements associated with the user.
	// Returns an error if the user id does not exist in the database.
	RemoveUser(id string) error

	// User gets the user with id from the database and creates the representing user struct.
	// Returns nil if the user does not exist in the database.
	User(id string) User

	// Users gets all users from the database and retrurns a map associating user ids with their representing structs.
	Users() map[string]User

	// AddGroups adds new group to the database and returns the representing struct.
	// Returns an error if the group id already exists in the database.
	AddGroup(id string) (Group, error)

	// RemoveGroup removes a group from the database.
	// Returns an error if the group id does not exist in the database.
	RemoveGroup(id string) error

	// Group gets the group with id from the database and creates the representing group struct.
	// Returns nil if the group does not exist in the database.
	Group(id string) Group

	// Groups gets all groups from the database and retrurns a map associating group ids with their representing structs.
	Groups() map[string]Group
}

// User provides a set of operations on users as represented in the database.
// It is meant to be used within the Update and View functions of a Db struct.
type User interface {
	// AddDevice adds a new device with id to the database associated with the current user. key is the secret key used for authentication between device client and server.
	// The isVirtual flag hints if the is only existent in the user database and is not available on the device api.
	// Returns the representing struct on success or an error if the device is already present in the database.
	AddDevice(id string, key []byte, isVirtual bool) (Device, error)

	// RemoveDevice removes a device from the database if ith associated with the current user.
	// The database itself should also remvove all sensors and measurements associated with the device.
	// Returns an error if the device id does not exist in the database.
	RemoveDevice(id string) error

	// Device gets the device with id from the database if it is associated with the current user and creates the representing device struct.
	// Returns nil if the device does not exist in the database.
	Device(id string) Device

	// Devices gets all devices (virtual and non-virtual) associated with the current user from the database and retrurns a map associating device ids with their representing structs.
	Devices() map[string]Device

	// VirtualDevices gets all virtual devices associated with the current user from the database and retrurns a map associating device ids with their representing structs.
	VirtualDevices() map[string]Device

	// HasPassword returns true if the hash stored in the database for the current user matches the hash of the provided pw string, return false otherwise.
	HasPassword(pw string) bool

	// IsAdmin returns the state of the Admin flag stored in the database for the currrent user.
	IsAdmin() bool

	// SetAdmin sets the Admin flag in the database for the current user.
	SetAdmin(b bool) error

	// Groups returns a map of group ids to Group objects for all groups the current users belongs to.
	Groups() map[string]Group

	// IsGroupAdmin returns true if the current user is listed as an admin for the given group in the database, return false otherwise.
	IsGroupAdmin(groupId string) bool

	// Returns the id that identifies the current user in the database.
	Id() string

	// LoadReadings loads measurements for the given timespan, resolution and sensors identified by device and id from the database, if they belong to the user.
	// Returns a mapping device id to sensorid to Value arrays.
	LoadReadings(since, until time.Time, resolution string, sensors map[string][]string) (map[string]map[string][]msg2api.Measurement, error)
}

// Group provides a set of operations on groups as represented in the database.
// It is meant to be used within the Update and View functions of a Db struct.
type Group interface {
	// AddUser associates the given user id with the current group.
	// Returns an error if the id is already associated.
	AddUser(id string) error

	// RemoveUser disassociates the given user id from the group.
	// Returns an error if the id is not associated.
	RemoveUser(id string) error

	// GetUsers returns a map from user ids to User structs, containing all users associated with the current group.
	GetUsers() map[string]User

	// SetAdmin adds the given user id to the list of admins for the current group.
	// Returns an error if the id is already in the list.
	SetAdmin(id string) error

	// UnsetAdmin removes the given user id from the list of group admins of the current group.
	// Returns an error if the id is not in the list.
	UnsetAdmin(id string) error

	// GetAdmins returns a map from user ids to User structs, containing all user listed as admins of the current group.
	GetAdmins() map[string]User

	// AddSensor associates the sensor with the given dbid to the current group.
	// Returns an error if the sensors is already associated.
	AddSensor(dbid uint64) error

	// RemoveSensor disassociates the sensor with the given dbid from the current group.
	// Returns an error if the sensor is not associated with the current group.
	RemoveSensor(dbid uint64) error

	// GetSensors returns an array of dbids of all sensors associated with the current group.
	GetSensors() []uint64

	// Returns the id that identifies the current group in the database.
	Id() string
}

// Device provides a set of operations on devices as represented in the database.
// It is meant to be used within the Update and View functions of a Db struct.
type Device interface {
	// AddSensor adds a new sensor with id, unit and port associated with the current device to the database.
	// Returns the representing Sensors struct or an error if the the device already exists.
	AddSensor(id, unit string, port int32) (Sensor, error)

	// Sensor gets the sensor with id from the database if it is associated with the current device and creates the representing sensor struct.
	// Returns nil if the seonsor does not exist in the database.
	Sensor(id string) Sensor

	// Sensors gets all sensors (virtual and non-virtual) associated with the current device from the database and retrurns a map associating sensors ids with their representing structs.
	Sensors() map[string]Sensor

	// Sensors gets all virtual sensors associated with the current device from the database and retrurns a map associating sensors ids with their representing structs.
	VirtualSensors() map[string]Sensor

	// RemoveDevice removes a device from the database if ith associated with the current user.
	// The database itself should also remvove all sensors and measurements associated with the device.
	// Returns an error if the device id does not exist in the database.
	RemoveSensor(id string) error

	// Id returns the device id that, in combination with the associated user, identifies the current device in the database.
	Id() string

	// User retruns the struct to the user the device is associated with.
	User() User

	// Key returns the secret key that is used for device authentication in the device api.
	Key() []byte

	// Name returns the name given to the current device.
	Name() string

	// Set name renames the current device.
	SetName(string) error

	// IsVirtual returns the state of the virtual flag of the current device in the database.
	IsVirtual() bool
}

// Sensor provides a set of operations on sensors as represented in the database.
// It is meant to be used within the Update and View functions of a Db struct.
type Sensor interface {
	// Id return the sensor id the, in combination with the associated device, identifies the sensor.
	Id() string

	// DbId returns the unique id the sensor has in the database.
	DbId() uint64

	// Device returns the struct to the device the sensor is associated with.
	Device() Device

	// Name returns the name given to the current sensor.
	Name() string

	// SetName renames the current sensor.
	SetName(string) error

	// Groups returns a map of group ids to Group objects for all groups the current sensor belongs to.
	Groups() map[string]Group

	// Port returns the physical port the sensor was assinged with on creation.
	Port() int32

	// Unit return the unit of the measured values of the sensors assinged on creation.
	Unit() string

	// IsVirtual returns the state of the virtual flag of the current sensor in the database.
	IsVirtual() bool
}

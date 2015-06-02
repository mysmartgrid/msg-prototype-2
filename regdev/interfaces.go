package regdev

type Tx interface {
	AddDevice(id string, key []byte) error
	Device(devId string) RegisteredDevice
	Devices() map[string]RegisteredDevice
}

type DeviceConfiguration struct {
	LinkedTo string `json:"linkedTo,omitempty"`
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
}

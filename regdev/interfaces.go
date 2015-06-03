package regdev

type Tx interface {
	AddDevice(id string, key []byte) error
	Device(devId string) RegisteredDevice
	Devices() map[string]RegisteredDevice
}

type DeviceConfigNetLan struct {
	Enabled    bool   `json:"enabled"`
	Protocol   string `json:"protocol,omitempty"`
	IP         string `json:"ip,omitempty"`
	Netmask    string `json:"netmask,omitempty"`
	Gateway    string `json:"gateway,omitempty"`
	Nameserver string `json:"nameserver,omitempty"`
}

type DeviceConfigNetWifi struct {
	Enabled    bool   `json:"enabled"`
	SSID       string `json:"essid,omitempty"`
	Encryption string `json:"enc,omitempty"`
	PSK        string `json:"psk,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
	IP         string `json:"ip,omitempty"`
	Netmask    string `json:"netmask,omitempty"`
	Gateway    string `json:"gateway,omitempty"`
	Nameserver string `json:"nameserver,omitempty"`
}

type DeviceConfigNetwork struct {
	LAN  *DeviceConfigNetLan  `json:"lan,omitempty"`
	Wifi *DeviceConfigNetWifi `json:"wifi,omitempty"`
}

type DeviceConfiguration struct {
	LinkedTo string               `json:"linkedTo,omitempty"`
	Network  *DeviceConfigNetwork `json:"network,omitempty"`
}

type RegisteredDevice interface {
	Id() string
	Key() []byte

	UserLink() (string, bool)
	LinkTo(uid string) error
	Unlink() error

	GetNetworkConfig() DeviceConfigNetwork
	SetNetworkConfig(conf *DeviceConfigNetwork) error
}

type Db interface {
	Close()

	Update(func(Tx) error) error
	View(func(Tx) error) error
}

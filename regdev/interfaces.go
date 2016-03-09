package regdev

import (
	"encoding/json"
	"time"
)

// Tx provides a set of operations to interact with the database through the Db's 'Open' and 'View' methods.
type Tx interface {
	AddDevice(id string, key []byte) error
	Device(devID string) RegisteredDevice
	Devices() map[string]RegisteredDevice
}

// DeviceIfaceIPConfig contains the configuration of a device's network interface.
type DeviceIfaceIPConfig struct {
	Protocol   string `json:"protocol,omitempty"`
	IP         string `json:"ip,omitempty"`
	Netmask    string `json:"netmask,omitempty"`
	Gateway    string `json:"gateway,omitempty"`
	Nameserver string `json:"nameserver,omitempty"`
}

// DeviceConfigNetLan contains the configuration of a device's wired network interface.
type DeviceConfigNetLan struct {
	DeviceIfaceIPConfig

	Enabled bool `json:"enabled"`
}

// DeviceConfigNetWifi contains the configuration of a device's wireless network interface.
type DeviceConfigNetWifi struct {
	DeviceIfaceIPConfig

	Enabled    bool   `json:"enabled"`
	SSID       string `json:"essid,omitempty"`
	Encryption string `json:"enc,omitempty"`
	PSK        string `json:"psk,omitempty"`
}

// DeviceConfigNetwork contains configurations of all network interfaces of a device.
type DeviceConfigNetwork struct {
	LAN  *DeviceConfigNetLan  `json:"lan,omitempty"`
	Wifi *DeviceConfigNetWifi `json:"wifi,omitempty"`
}

// DeviceConfiguration contains all configuration information of a device.
type DeviceConfiguration struct {
	LinkedTo string               `json:"linkedTo,omitempty"`
	Network  *DeviceConfigNetwork `json:"network,omitempty"`
}

// Heartbeat contains all information sent in a heartbeat from a device.
type Heartbeat struct {
	Time   time.Time
	Memory struct {
		Total    uint64
		Cached   uint64
		Buffered uint64
		Free     uint64
	}
	Uptime   time.Duration
	Resets   uint64
	Type     string
	Syslog   string
	Firmware struct {
		Version     string
		ReleaseTime string
		Build       string
		Tag         string
	}
	Config *DeviceConfigNetwork
}

type hbData struct {
	Time   int64
	Memory struct {
		Total    uint64
		Cached   uint64
		Buffered uint64
		Free     uint64
	}
	Uptime   uint64
	Resets   uint64
	Type     string
	Syslog   string
	Firmware struct {
		Version     string
		ReleaseTime string
		Build       string
		Tag         string
	}
	Config *DeviceConfigNetwork
}

// MarshalJSON converts the Heartbeat struct to raw json data.
func (hb Heartbeat) MarshalJSON() ([]byte, error) {
	data := hbData{
		Time:     hb.Time.Unix(),
		Memory:   hb.Memory,
		Uptime:   uint64(hb.Uptime.Seconds()),
		Resets:   hb.Resets,
		Type:     hb.Type,
		Syslog:   hb.Syslog,
		Firmware: hb.Firmware,
		Config:   hb.Config,
	}
	return json.Marshal(data)
}

// UnmarshalJSON converts raw json hartbeat data into the Heartbeat struct.
func (hb *Heartbeat) UnmarshalJSON(raw []byte) error {
	var data hbData
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}
	*hb = Heartbeat{
		Time:     time.Unix(data.Time, 0),
		Memory:   data.Memory,
		Uptime:   time.Duration(data.Uptime) * time.Second,
		Resets:   data.Resets,
		Type:     data.Type,
		Syslog:   data.Syslog,
		Firmware: data.Firmware,
		Config:   data.Config,
	}
	return nil
}

// RegisteredDevice provides methods to manage a device registered in the device database.
type RegisteredDevice interface {
	// Id returns the unique device id identifyin the device.
	ID() string
	// Key returns the secret key that is used for device authentication.
	Key() []byte

	// UserLink returns the user id linked to the device and true, if a user is linked to the device,
	// an emtpy string and false otherwise.
	UserLink() (string, bool)
	// LinkTo links the device to the given user id or returns an error if the device is already linked to a user.
	LinkTo(uid string) error
	// Unlink unlinks the currently linked user from the device or returns an error if there is no user linked.
	Unlink() error

	// RegisterHeartbeat writes a new heartbeat from the device to the database
	// and updates the devices network config in the database if neccessary.
	RegisterHeartbeat(hb Heartbeat) error
	// GetHeartbeats return the last 'maxCount' heartbeats received from the device or
	// all of them if maxCount is zero.
	GetHeartbeats(maxCount uint64) []Heartbeat

	// GetNetworkConfig returns the current network configuration stroed for the device.
	GetNetworkConfig() DeviceConfigNetwork
	// Updates the network configuration for the device in the database.
	SetNetworkConfig(conf *DeviceConfigNetwork) error
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
}

package regdev

import (
	"encoding/json"
	"time"
)

type Tx interface {
	AddDevice(id string, key []byte) error
	Device(devId string) RegisteredDevice
	Devices() map[string]RegisteredDevice
}

type DeviceIfaceIPConfig struct {
	Protocol   string `json:"protocol,omitempty"`
	IP         string `json:"ip,omitempty"`
	Netmask    string `json:"netmask,omitempty"`
	Gateway    string `json:"gateway,omitempty"`
	Nameserver string `json:"nameserver,omitempty"`
}

type DeviceConfigNetLan struct {
	DeviceIfaceIPConfig

	Enabled bool `json:"enabled"`
}

type DeviceConfigNetWifi struct {
	DeviceIfaceIPConfig

	Enabled    bool   `json:"enabled"`
	SSID       string `json:"essid,omitempty"`
	Encryption string `json:"enc,omitempty"`
	PSK        string `json:"psk,omitempty"`
}

type DeviceConfigNetwork struct {
	LAN  *DeviceConfigNetLan  `json:"lan,omitempty"`
	Wifi *DeviceConfigNetWifi `json:"wifi,omitempty"`
}

type DeviceConfiguration struct {
	LinkedTo string               `json:"linkedTo,omitempty"`
	Network  *DeviceConfigNetwork `json:"network,omitempty"`
}

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
}

func (hb Heartbeat) MarshalJSON() ([]byte, error) {
	data := hbData{
		Time: hb.Time.Unix(),
		Memory: hb.Memory,
		Uptime: uint64(hb.Uptime.Seconds()),
		Resets: hb.Resets,
		Type: hb.Type,
		Syslog: hb.Syslog,
		Firmware: hb.Firmware,
	}
	return json.Marshal(data)
}

func (hb *Heartbeat) UnmarshalJSON(raw []byte) error {
	var data hbData
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}
	*hb = Heartbeat{
		Time: time.Unix(data.Time, 0),
		Memory: data.Memory,
		Uptime: time.Duration(data.Uptime) * time.Second,
		Resets: data.Resets,
		Type: data.Type,
		Syslog: data.Syslog,
		Firmware: data.Firmware,
	}
	return nil
}

type RegisteredDevice interface {
	Id() string
	Key() []byte

	UserLink() (string, bool)
	LinkTo(uid string) error
	Unlink() error

	RegisterHeartbeat(hb Heartbeat) error
	GetHeartbeats(maxCount uint64) []Heartbeat

	GetNetworkConfig() DeviceConfigNetwork
	SetNetworkConfig(conf *DeviceConfigNetwork) error
}

type Db interface {
	Close()

	Update(func(Tx) error) error
	View(func(Tx) error) error
}

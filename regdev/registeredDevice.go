package regdev

import (
	"encoding/json"
	"errors"
	"github.com/boltdb/bolt"
	"net"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type registeredDevice struct {
	b  *bolt.Bucket
	id string
}

var (
	registeredDeviceKey       = []byte("key")
	registeredDeviceUser      = []byte("user")
	registeredDeviceNetwork   = []byte("network")
	registeredDeviceHeartbeat = []byte("heartbeat")

	errBadNetworkConfig = errors.New("bad network config")
)

func (r *registeredDevice) init(key []byte) {
	r.b.Put(registeredDeviceKey, key)
	r.b.CreateBucket(registeredDeviceHeartbeat)
}

func (r *registeredDevice) ID() string {
	return r.id
}

func (r *registeredDevice) Key() []byte {
	return r.b.Get(registeredDeviceKey)
}

func (r *registeredDevice) UserLink() (string, bool) {
	if uid := r.b.Get(registeredDeviceUser); uid != nil {
		return string(uid), true
	}
	return "", false
}

func (r *registeredDevice) LinkTo(uid string) error {
	if r.b.Get(registeredDeviceUser) != nil {
		return ErrAlreadyLinked
	}
	return r.b.Put(registeredDeviceUser, []byte(uid))
}

func (r *registeredDevice) Unlink() error {
	if err := r.b.Delete(registeredDeviceUser); err != nil {
		return err
	}
	return r.b.Delete(registeredDeviceNetwork)
}

func (r *registeredDevice) RegisterHeartbeat(hb Heartbeat) error {
	hbKey := []byte(strconv.FormatInt(hb.Time.Unix(), 10))

	if hb.Config != nil {
		if err := r.SetNetworkConfig(hb.Config); err != nil {
			return err
		}
		hb.Config = nil
	}

	bucket := r.b.Bucket(registeredDeviceHeartbeat)
	value, err := json.Marshal(hb)
	if err != nil {
		return err
	}
	return bucket.Put(hbKey, value)
}

func (r *registeredDevice) GetHeartbeats(maxCount uint64) (result []Heartbeat) {
	if maxCount == 0 {
		maxCount = 0xFFFFFFFFFFFFFFFF
	}

	cursor := r.b.Bucket(registeredDeviceHeartbeat).Cursor()
	key, value := cursor.Last()
	maxCount--
	for ; maxCount > 0 && key != nil; maxCount-- {
		ts, err := strconv.ParseInt(string(key), 10, 64)
		if err != nil {
			panic(err)
		}
		var hb Heartbeat
		if err := json.Unmarshal(value, &hb); err != nil {
			panic(err)
		}
		hb.Time = time.Unix(ts, 0)
		result = append(result, hb)
		key, value = cursor.Prev()
	}
	return
}

func (r *registeredDevice) GetNetworkConfig() DeviceConfigNetwork {
	var result DeviceConfigNetwork

	data := r.b.Get(registeredDeviceNetwork)
	if data == nil || json.Unmarshal(data, &result) != nil {
		return DeviceConfigNetwork{}
	}

	return result
}

func checkAndCleanProtocol(conf *DeviceIfaceIPConfig) bool {
	switch conf.Protocol {
	case "dhcp":
		conf.IP = ""
		conf.Netmask = ""
		conf.Gateway = ""
		conf.Nameserver = ""
		return true

	case "static":
		if net.ParseIP(conf.IP) == nil || net.ParseIP(conf.Netmask) == nil || net.ParseIP(conf.Gateway) == nil ||
			net.ParseIP(conf.Nameserver) == nil {
			return false
		}
		return true

	default:
		return false
	}
}

func checkConfigLan(conf *DeviceConfigNetLan) bool {
	if conf == nil || !conf.Enabled {
		return true
	}

	return checkAndCleanProtocol(&conf.DeviceIfaceIPConfig)
}

func checkConfigWifi(conf *DeviceConfigNetWifi) bool {
	if conf == nil || !conf.Enabled {
		return true
	}

	if !checkAndCleanProtocol(&conf.DeviceIfaceIPConfig) {
		return false
	}

	isPrintable := func(s string) bool {
		return -1 == strings.IndexFunc(s, func(r rune) bool {
			return !unicode.IsPrint(r)
		})
	}

	if !isPrintable(conf.SSID) {
		return false
	}

	switch conf.Encryption {
	case "wpa", "wpa2":
		if !isPrintable(conf.PSK) {
			return false
		}
		return true

	case "open":
		conf.PSK = ""
		return true
	}

	return false
}

func (r *registeredDevice) SetNetworkConfig(conf *DeviceConfigNetwork) error {
	if !checkConfigLan(conf.LAN) || !checkConfigWifi(conf.Wifi) {
		return errBadNetworkConfig
	}

	data, err := json.Marshal(conf)
	if err != nil {
		return err
	}
	return r.b.Put(registeredDeviceNetwork, data)
}

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
	registeredDevice_key       = []byte("key")
	registeredDevice_user      = []byte("user")
	registeredDevice_network   = []byte("network")
	registeredDevice_heartbeat = []byte("heartbeat")

	badNetworkConfig = errors.New("bad network config")
)

func (r *registeredDevice) init(key []byte) {
	r.b.Put(registeredDevice_key, key)
	r.b.CreateBucket(registeredDevice_heartbeat)
}

func (r *registeredDevice) Id() string {
	return r.id
}

func (r *registeredDevice) Key() []byte {
	return r.b.Get(registeredDevice_key)
}

func (r *registeredDevice) UserLink() (string, bool) {
	if uid := r.b.Get(registeredDevice_user); uid != nil {
		return string(uid), true
	}
	return "", false
}

func (r *registeredDevice) LinkTo(uid string) error {
	if r.b.Get(registeredDevice_user) != nil {
		return AlreadyLinked
	}
	return r.b.Put(registeredDevice_user, []byte(uid))
}

func (r *registeredDevice) Unlink() error {
	if err := r.b.Delete(registeredDevice_user); err != nil {
		return err
	}
	return r.b.Delete(registeredDevice_network)
}

func (r *registeredDevice) RegisterHeartbeat(hb Heartbeat) error {
	hbKey := []byte(strconv.FormatInt(hb.Time.Unix(), 10))

	bucket := r.b.Bucket(registeredDevice_heartbeat)
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

	cursor := r.b.Bucket(registeredDevice_heartbeat).Cursor()
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

	data := r.b.Get(registeredDevice_network)
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
	if !conf.Enabled {
		return true
	}

	return checkAndCleanProtocol(&conf.DeviceIfaceIPConfig)
}

func checkConfigWifi(conf *DeviceConfigNetWifi) bool {
	if !conf.Enabled {
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
		return badNetworkConfig
	}

	data, err := json.Marshal(conf)
	if err != nil {
		return err
	}
	return r.b.Put(registeredDevice_network, data)
}

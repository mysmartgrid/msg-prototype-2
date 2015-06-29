package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	crand "crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	msgp "github.com/mysmartgrid/msg2api"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var useSSL bool
var tlsConfig tls.Config
var dev *Device

type Sensor struct {
	Name string
	Unit string
	Port int32
}

type Device struct {
	Id  string
	Key []byte

	User string

	Sensors map[string]Sensor

	api     string
	client_ *msgp.DeviceClient

	regdevApi string
}

var deviceNotRegistered = errors.New("device not registered")

func (dev *Device) client() *msgp.DeviceClient {
	if dev.client_ == nil {
		client, err := msgp.NewDeviceClient(dev.api+"/"+dev.User+"/"+dev.Id, dev.Key, &tlsConfig)
		if err != nil {
			log.Fatalf("Device::client: %v", err.Error())
		}

		client.RequestRealtimeUpdates = func(sensors []string) {
			log.Printf("server requested realtime updates for %v", sensors)
		}

		dev.client_ = client
	}

	return dev.client_
}

func newRandomDevice() *Device {
	var buf [32]byte

	_, err := crand.Read(buf[:])
	if err != nil {
		log.Fatalf("rand read: %v", err.Error())
	}

	return &Device{
		Id:  hex.EncodeToString(buf[0:16]),
		Key: buf[16:32],
	}
}

func (dev *Device) GenerateRandomSensors(count int64) {
	if dev.Sensors == nil {
		dev.Sensors = make(map[string]Sensor)
	}

	for count > 0 {
		var raw [16]byte

		if _, err := crand.Read(raw[:]); err != nil {
			log.Fatal("rand read: %v", err.Error())
		}

		id := hex.EncodeToString(raw[:])
		if _, ok := dev.Sensors[id]; ok {
			continue
		}

		dev.Sensors[id] = Sensor{
			Name: fmt.Sprintf("Sensor %v", count),
			Unit: []string{"U1", "U2"}[rand.Int31n(2)],
			Port: int32(len(dev.Sensors)),
		}
		count--
	}
}

func (dev *Device) Register() error {
	req, err := http.NewRequest("POST", dev.regdevApi+"/"+dev.Id, nil)
	if err != nil {
		return err
	}
	req.Header["X-Key"] = []string{hex.EncodeToString(dev.Key)}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return errors.New(string(body))
	}
	return nil
}

func getMemInfo() map[string]uint64 {
	data, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(data), "\n")
	result := make(map[string]uint64)
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Split(line, ":")
		key := fields[0]
		value, err := strconv.ParseUint(strings.Fields(fields[1])[0], 10, 64)
		if err != nil {
			panic(err)
		}

		switch key {
		case "MemTotal":
			result["Total"] = value
		case "MemFree":
			result["Free"] = value
		case "Cached":
			result["Cached"] = value
		case "Buffers":
			result["Buffered"] = value
		}
	}

	return result
}

func getUptime() uint64 {
	data, err := ioutil.ReadFile("/proc/uptime")
	if err != nil {
		panic(err.Error())
	}
	uptime, err := strconv.ParseFloat(strings.Fields(string(data))[0], 64)
	if err != nil {
		panic(err)
	}
	return uint64(uptime)
}

func (dev *Device) Heartbeat() (map[string]interface{}, error) {
	mac := hmac.New(sha256.New, dev.Key)
	hbInfo := map[string]interface{}{
		"Time":   time.Now().Unix(),
		"Memory": getMemInfo(),
		"Uptime": getUptime(),
		"Resets": 0,
		"Type":   "msgpc",
		"Syslog": "",
		"Firmware": map[string]string{
			"Version":     "0.1",
			"ReleaseTime": "not yet",
			"Build":       "from git",
			"Tag":         "<unknown>",
		},
		"config": map[string]interface{}{
			"lan": map[string]interface{}{
				"enabled":  true,
				"protocol": "dhcp",
			},
		},
	}
	hbData, err := json.Marshal(hbInfo)
	if err != nil {
		return nil, err
	}

	hbUrl, _ := url.Parse(dev.regdevApi + "/" + dev.Id + "/status")
	params := url.Values{
		"ts": []string{strconv.FormatInt(time.Now().Unix(), 10)},
	}
	mac.Write([]byte(params["ts"][0]))
	mac.Write(hbData)
	params["sig"] = []string{hex.EncodeToString(mac.Sum(nil))}
	hbUrl.RawQuery = params.Encode()
	mac.Reset()

	req, err := http.NewRequest("POST", hbUrl.String(), bytes.NewReader(hbData))
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		var err error
		var body, nonce, iv, macValue []byte

		if body, err = ioutil.ReadAll(resp.Body); err != nil {
			return nil, err
		}
		if body, err = hex.DecodeString(string(body)); err != nil {
			return nil, err
		}

		if nonce, err = hex.DecodeString(resp.Header.Get("X-Nonce")); err != nil {
			return nil, err
		}
		if iv, err = hex.DecodeString(resp.Header.Get("X-IV")); err != nil {
			return nil, err
		}
		if macValue, err = hex.DecodeString(resp.Header.Get("X-HMAC")); err != nil {
			return nil, err
		}

		mac.Write(body)

		if !hmac.Equal(mac.Sum(nil), macValue) {
			return nil, errors.New("bad hmac")
		}

		mac.Reset()
		mac.Write(nonce)
		key := mac.Sum(nil)[:16]

		cinst, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		transform := cipher.NewCFBDecrypter(cinst, iv[:])
		transform.XORKeyStream(body, body)

		var content map[string]interface{}
		if err := json.Unmarshal(body, &content); err != nil {
			log.Panic(err)
		}
		if user, ok := content["linkedTo"].(string); ok {
			dev.User = user
		} else {
			dev.User = ""
		}
		return content, nil

	case 404:
		return nil, deviceNotRegistered

	default:
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, errors.New(string(body))
	}
}

func (dev *Device) RegisterSensors() error {
	for id, sens := range dev.Sensors {
		if err := dev.client().AddSensor(id, sens.Unit, sens.Port); err != nil {
			return err
		}
		md := msgp.SensorMetadata{
			Name: &sens.Name,
		}
		if err := dev.client().UpdateSensor(id, md); err != nil {
			return err
		}
	}

	return nil
}

func (dev *Device) UpdateSensors() error {
	for id, sens := range dev.Sensors {
		md := msgp.SensorMetadata{
			Name: &sens.Name,
		}
		if err := dev.client().UpdateSensor(id, md); err != nil {
			return err
		}
	}

	return nil
}

func (dev *Device) SendUpdates(interval time.Duration, count int64) error {
	for ; count != 0; count-- {
		for id, _ := range dev.Sensors {
			values := make(map[string][]msgp.Measurement, len(dev.Sensors))
			values[id] = []msgp.Measurement{{time.Now(), rand.Float64()}}
			if err := dev.client().Update(values); err != nil {
				return err
			}
		}

		time.Sleep(interval)
	}

	return nil
}

func (dev *Device) RenameSensors() error {
	for id, sens := range dev.Sensors {
		name := fmt.Sprintf("%v (%v)", sens.Name, rand.Int31n(1000))
		if err := dev.client().UpdateSensor(id, msgp.SensorMetadata{Name: &name}); err != nil {
			return err
		}
	}

	return nil
}

func (dev *Device) ReplaceSensors() error {
	for id, _ := range dev.Sensors {
		err := dev.client().RemoveSensor(id)
		switch e := err.(type) {
		case *msgp.Error:
			if e.Code != "operation failed" || e.Extra != "id invalid" {
				return err
			}

		case nil:
		default:
			return err
		}
	}
	count := len(dev.Sensors)
	dev.Sensors = nil
	dev.GenerateRandomSensors(int64(count))

	if err := dev.RegisterSensors(); err != nil {
		return err
	}

	return nil
}

func (dev *Device) Rename() error {
	return dev.client().Rename(fmt.Sprintf("%v (%v)", dev.Id, rand.Int31n(100)))
}

func (dev *Device) Wait(count uint64) error {
	for ; count > 0; count-- {
		if err := dev.client().RunOnce(); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if len(os.Args) < 2 {
		log.Println("bad args")
		return
	}

	rand.Seed(int64(time.Now().UnixNano()))

	for i := 1; i < len(os.Args); i++ {
		cmdName := os.Args[i]
		bailIf := func(err error) {
			if err != nil {
				log.Fatalf("%v: %v", cmdName, err.Error())
			}
		}
		next := func(name ...string) {
			i++
			if i >= len(os.Args) {
				if len(name) > 0 {
					log.Fatalf("argument %v to %v missing", name[0], cmdName)
				} else {
					log.Fatalf("argument to %v missing", cmdName)
				}
			}
		}

		switch os.Args[i] {
		case "-bad-tls":
			tlsConfig.InsecureSkipVerify = true
			http.DefaultTransport.(*http.Transport).TLSClientConfig = &tlsConfig

		case "newRandom":
			dev = newRandomDevice()
			dev.regdevApi = "http://[::1]:8080/regdev/v1"
			bailIf(dev.Register())

		case "print":
			data, err := json.MarshalIndent(dev, "", "  ")
			bailIf(err)
			log.Println(string(data))

		case "save":
			data, err := json.MarshalIndent(dev, "", "\t")
			bailIf(err)
			next()
			err = ioutil.WriteFile(os.Args[i], data, 0666)
			bailIf(err)

		case "load":
			next()
			data, err := ioutil.ReadFile(os.Args[i])
			bailIf(err)
			dev = new(Device)
			bailIf(json.Unmarshal(data, dev))
			dev.api = "ws://[::1]:8080/ws/device"
			dev.regdevApi = "http://[::1]:8080/regdev/v1"

		case "heartbeat":
			info, err := dev.Heartbeat()
			bailIf(err)
			log.Println(info)

		case "genSensors":
			next()
			count, err := strconv.ParseInt(os.Args[i], 10, 32)
			bailIf(err)
			dev.GenerateRandomSensors(count)

		case "registerSensors":
			bailIf(dev.RegisterSensors())

		case "sendUpdates":
			next("interval")
			interval, err := time.ParseDuration(os.Args[i])
			bailIf(err)
			next("count")
			count, err := strconv.ParseInt(os.Args[i], 10, 32)
			bailIf(err)
			bailIf(dev.SendUpdates(interval, count))

		case "renameSensors":
			bailIf(dev.RenameSensors())

		case "replaceSensors":
			bailIf(dev.ReplaceSensors())

		case "rename":
			bailIf(dev.Rename())

		case "wait":
			next("count")
			count, err := strconv.ParseUint(os.Args[i], 10, 32)
			bailIf(err)
			bailIf(dev.Wait(count))

		default:
			log.Fatalf("bad command %v", cmdName)
		}
	}
}

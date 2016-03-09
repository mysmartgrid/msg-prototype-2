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
	sdm630 "github.com/mysmartgrid/gosdm630"
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
var dev *device

type sensor struct {
	Name                string
	Unit                string
	Port                int32
	LastRealtimeRequest time.Time
}

type device struct {
	ID  string
	Key []byte

	User string

	Sensors map[string]sensor

	api    string
	client *msgp.DeviceClient

	regdevAPI string
}

var errDeviceNotRegistered = errors.New("device not registered")

var sensorDefinitons = [...]sensor{
	{Name: "Voltage L1", Unit: "V", Port: 1, LastRealtimeRequest: time.Unix(0, 0)},
	{Name: "Voltage L2", Unit: "V", Port: 1, LastRealtimeRequest: time.Unix(0, 0)},
	{Name: "Voltage L3", Unit: "V", Port: 1, LastRealtimeRequest: time.Unix(0, 0)},

	{Name: "Current L1", Unit: "A", Port: 1, LastRealtimeRequest: time.Unix(0, 0)},
	{Name: "Current L2", Unit: "A", Port: 1, LastRealtimeRequest: time.Unix(0, 0)},
	{Name: "Current L3", Unit: "A", Port: 1, LastRealtimeRequest: time.Unix(0, 0)},

	{Name: "Power L1", Unit: "W", Port: 1, LastRealtimeRequest: time.Unix(0, 0)},
	{Name: "Power L2", Unit: "W", Port: 1, LastRealtimeRequest: time.Unix(0, 0)},
	{Name: "Power L3", Unit: "W", Port: 1, LastRealtimeRequest: time.Unix(0, 0)},

	{Name: "Import L1", Unit: "kWh", Port: 1, LastRealtimeRequest: time.Unix(0, 0)},
	{Name: "Import L2", Unit: "kWh", Port: 1, LastRealtimeRequest: time.Unix(0, 0)},
	{Name: "Import L3", Unit: "kWh", Port: 1, LastRealtimeRequest: time.Unix(0, 0)},

	{Name: "Export L1", Unit: "kWh", Port: 1, LastRealtimeRequest: time.Unix(0, 0)},
	{Name: "Export L2", Unit: "kWh", Port: 1, LastRealtimeRequest: time.Unix(0, 0)},
	{Name: "Export L3", Unit: "kWh", Port: 1, LastRealtimeRequest: time.Unix(0, 0)},

	{Name: "Power Factor L1", Unit: "", Port: 1, LastRealtimeRequest: time.Unix(0, 0)},
	{Name: "Power Factor L2", Unit: "", Port: 1, LastRealtimeRequest: time.Unix(0, 0)},
	{Name: "Power Factor L3", Unit: "", Port: 1, LastRealtimeRequest: time.Unix(0, 0)},
}

func (dev *device) getClient() *msgp.DeviceClient {
	if dev.client == nil {
		client, err := msgp.NewDeviceClient(dev.api+"/"+dev.User+"/"+dev.ID, dev.Key, &tlsConfig)
		if err != nil {
			log.Fatalf("Device::client: %v", err.Error())
		}

		client.RequestRealtimeUpdates = func(sensors []string) {
			log.Printf("server requested realtime updates for %v", sensors)
			for _, sensor := range sensors {
				if s, ok := dev.Sensors[sensor]; ok {
					s.LastRealtimeRequest = time.Now()
				}
			}
		}

		dev.client = client
	}

	return dev.client
}

func newRandomDevice() *device {
	var buf [32]byte

	_, err := crand.Read(buf[:])
	if err != nil {
		log.Fatalf("rand read: %v", err.Error())
	}

	return &device{
		ID:  hex.EncodeToString(buf[0:16]),
		Key: buf[16:32],
	}
}

func newSDM630Device() *device {
	var buf [32]byte

	_, err := crand.Read(buf[:])
	if err != nil {
		log.Fatalf("rand read: %v", err.Error())
	}

	device := device{
		ID:      hex.EncodeToString(buf[0:16]),
		Key:     buf[16:32],
		Sensors: make(map[string]sensor),
	}

	for _, sensor := range sensorDefinitons {
		var raw [16]byte

		if _, err := crand.Read(raw[:]); err != nil {
			log.Fatalf("rand read: %v", err.Error())
		}

		for {
			id := hex.EncodeToString(raw[:])
			if _, ok := device.Sensors[id]; !ok {
				device.Sensors[id] = sensor
				break
			}
		}
	}

	return &device
}

func (dev *device) generateRandomSensors(count int64) {
	if dev.Sensors == nil {
		dev.Sensors = make(map[string]sensor)
	}

	for count > 0 {
		var raw [16]byte

		if _, err := crand.Read(raw[:]); err != nil {
			log.Fatalf("rand read: %v", err.Error())
		}

		id := hex.EncodeToString(raw[:])
		if _, ok := dev.Sensors[id]; ok {
			continue
		}

		dev.Sensors[id] = sensor{
			Name: fmt.Sprintf("Sensor %v", count),
			Unit: []string{"U1", "U2"}[rand.Int31n(2)],
			Port: int32(len(dev.Sensors)),
		}
		count--
	}
}

func (dev *device) register() error {
	req, err := http.NewRequest("POST", dev.regdevAPI+"/"+dev.ID, nil)
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

func (dev *device) heartbeat() (map[string]interface{}, error) {
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

	hbURL, _ := url.Parse(dev.regdevAPI + "/" + dev.ID + "/status")
	params := url.Values{
		"ts": []string{strconv.FormatInt(time.Now().Unix(), 10)},
	}
	mac.Write([]byte(params["ts"][0]))
	mac.Write(hbData)
	params["sig"] = []string{hex.EncodeToString(mac.Sum(nil))}
	hbURL.RawQuery = params.Encode()
	mac.Reset()

	req, err := http.NewRequest("POST", hbURL.String(), bytes.NewReader(hbData))
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
		return nil, errDeviceNotRegistered

	default:
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, errors.New(string(body))
	}
}

func (dev *device) registerSensors() error {
	for id, sens := range dev.Sensors {
		if err := dev.getClient().AddSensor(id, sens.Unit, sens.Port); err != nil {
			return err
		}
		md := msgp.SensorMetadata{
			Name: &sens.Name,
		}
		if err := dev.getClient().UpdateSensor(id, md); err != nil {
			return err
		}
	}

	return nil
}

func (dev *device) updateSensors() error {
	for id, sens := range dev.Sensors {
		md := msgp.SensorMetadata{
			Name: &sens.Name,
		}
		if err := dev.getClient().UpdateSensor(id, md); err != nil {
			return err
		}
	}

	return nil
}

func (dev *device) sendRandomUpdates(interval time.Duration, count int64) error {
	for ; count != 0; count-- {
		for id := range dev.Sensors {
			values := make(map[string][]msgp.Measurement, len(dev.Sensors))
			values[id] = []msgp.Measurement{{time.Now(), rand.Float64()}}
			if err := dev.getClient().Update(values); err != nil {
				return err
			}
		}

		time.Sleep(interval)
	}

	return nil
}

func initSDM639(serialDevice string, interval int) *sdm630.MeasurementCache {
	var rc = make(sdm630.ReadingChannel)
	qe := sdm630.NewQueryEngine(
		serialDevice,
		interval,
		true,
		rc,
	)
	go qe.Produce()
	mc := sdm630.NewMeasurementCache(
		rc,
		interval,
		true,
	)
	go mc.ConsumeData()

	return mc
}

func (dev *device) sendSDM630Updates(interval time.Duration, count int64, serialDevice string) error {
	mc := initSDM639(serialDevice, int(interval.Seconds()))

	for ; count != 0; count-- {

		r := mc.GetLast()
		for id, sensor := range dev.Sensors {
			values := make(map[string][]msgp.Measurement, len(dev.Sensors))
			var val float32
			switch sensor.Name {
			case "Voltage L1":
				val = r.Voltage.L1
			case "Voltage L2":
				val = r.Voltage.L2
			case "Voltage L3":
				val = r.Voltage.L3
			case "Current L1":
				val = r.Current.L1
			case "Current L2":
				val = r.Current.L2
			case "Current L3":
				val = r.Current.L3
			case "Power L1":
				val = r.Power.L1
			case "Power L2":
				val = r.Power.L2
			case "Power L3":
				val = r.Power.L3
			case "Import L1":
				val = r.Import.L1
			case "Import L2":
				val = r.Import.L2
			case "Import L3":
				val = r.Import.L3
			case "Export L1":
				val = r.Export.L1
			case "Export L2":
				val = r.Export.L2
			case "Export L3":
				val = r.Export.L3
			case "Power Factor L1":
				val = r.Cosphi.L1
			case "Power Factor L2":
				val = r.Cosphi.L2
			case "Power Factor L3":
				val = r.Cosphi.L3
			default:
				continue
			}
			values[id] = []msgp.Measurement{{time.Now(), float64(val)}}
			if err := dev.getClient().Update(values); err != nil {
				return err
			}
		}

		time.Sleep(interval)
	}

	return nil
}

func (dev *device) renameSensors() error {
	for id, sens := range dev.Sensors {
		name := fmt.Sprintf("%v (%v)", sens.Name, rand.Int31n(1000))
		if err := dev.getClient().UpdateSensor(id, msgp.SensorMetadata{Name: &name}); err != nil {
			return err
		}
	}

	return nil
}

func (dev *device) replaceSensors() error {
	for id := range dev.Sensors {
		err := dev.getClient().RemoveSensor(id)
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
	dev.generateRandomSensors(int64(count))

	if err := dev.registerSensors(); err != nil {
		return err
	}

	return nil
}

func (dev *device) rename() error {
	return dev.getClient().Rename(fmt.Sprintf("%v (%v)", dev.ID, rand.Int31n(100)))
}

func (dev *device) wait(count uint64) error {
	for ; count > 0; count-- {
		if err := dev.getClient().RunOnce(); err != nil {
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
			dev.regdevAPI = "http://[::1]:8080/api/regdev/v1"
			bailIf(dev.register())

		case "newSDM630":
			dev = newSDM630Device()
			dev.regdevAPI = "http://[::1]:8080/api/regdev/v1"
			bailIf(dev.register())

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
			dev = new(device)
			bailIf(json.Unmarshal(data, dev))
			dev.api = "ws://[::1]:8080/ws/device"
			dev.regdevAPI = "http://[::1]:8080/api/regdev/v1"

		case "heartbeat":
			info, err := dev.heartbeat()
			bailIf(err)
			log.Println(info)

		case "genSensors":
			next()
			count, err := strconv.ParseInt(os.Args[i], 10, 32)
			bailIf(err)
			dev.generateRandomSensors(count)

		case "registerSensors":
			bailIf(dev.registerSensors())

		case "sendRandomUpdates":
			next("interval")
			interval, err := time.ParseDuration(os.Args[i])
			bailIf(err)
			next("count")
			count, err := strconv.ParseInt(os.Args[i], 10, 32)
			bailIf(err)
			bailIf(dev.sendRandomUpdates(interval, count))

		case "sendSDM630Updates":
			next("interval")
			interval, err := time.ParseDuration(os.Args[i])
			bailIf(err)
			next("count")
			count, err := strconv.ParseInt(os.Args[i], 10, 32)
			bailIf(err)
			next()
			bailIf(dev.sendSDM630Updates(interval, count, os.Args[i]))

		case "renameSensors":
			bailIf(dev.renameSensors())

		case "replaceSensors":
			bailIf(dev.replaceSensors())

		case "rename":
			bailIf(dev.rename())

		case "wait":
			next("count")
			count, err := strconv.ParseUint(os.Args[i], 10, 32)
			bailIf(err)
			bailIf(dev.wait(count))

		default:
			log.Fatalf("bad command %v", cmdName)
		}
	}
}

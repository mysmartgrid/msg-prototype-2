package main

import (
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
	"os"
	"strconv"
	"time"
)

var useSSL bool
var tlsConfig tls.Config
var dev *Device

type Device struct {
	Id  string
	Key []byte

	User string

	Sensors map[string]string

	api     string
	client_ msgp.DeviceClient

	regdevApi string
}

var deviceNotRegistered = errors.New("device not registered")

func (dev *Device) client() msgp.DeviceClient {
	if dev.client_ == nil {
		client, err := msgp.NewDeviceClient(dev.api+"/"+dev.User+"/"+dev.Id, dev.Key, &tlsConfig)
		if err != nil {
			log.Fatalf("Device::client: %v", err.Error())
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
		dev.Sensors = make(map[string]string)
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

		dev.Sensors[id] = fmt.Sprintf("Sensor %v", count)
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

func (dev *Device) GetInfo() (map[string]interface{}, error) {
	resp, err := http.Get(dev.regdevApi + "/" + dev.Id)
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

		mac := hmac.New(sha256.New, dev.Key)
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
	for id, _ := range dev.Sensors {
		if err := dev.client().AddSensor(id); err != nil {
			return err
		}
	}

	return nil
}

func (dev *Device) UpdateSensors() error {
	for id, name := range dev.Sensors {
		if err := dev.client().RenameSensor(id, name); err != nil {
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
	for id, name := range dev.Sensors {
		if err := dev.client().RenameSensor(id, fmt.Sprintf("%v (%v)", name, rand.Int31n(1000))); err != nil {
			return err
		}
	}

	return nil
}

func (dev *Device) ReplaceSensors() error {
	for id, _ := range dev.Sensors {
		if err := dev.client().RemoveSensor(id); err != nil {
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

		case "new-random":
			dev = newRandomDevice()

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

		case "register":
			bailIf(dev.Register())

		case "getInfo":
			info, err := dev.GetInfo()
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

		default:
			log.Fatalf("bad command %v", cmdName)
		}
	}
}

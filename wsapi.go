package msgp

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/mysmartgrid/msg2api"
	"io/ioutil"
	"log"
	"msgp/db"
	"msgp/hub"
	"net/http"
	"time"
)

var (
	notAuthorized    = errors.New("not authorized")
	apiNotAuthorized = &msg2api.Error{Code: notAuthorized.Error()}
	noSensor         = errors.New("sensor not found")
)

type measurementWithMetadata struct {
	Device, Sensor string
	Time           time.Time
	Value          float64
}

type wsDevApi struct {
	User, Device string
	Db           db.Db
	Hub          *hub.Hub
	Writer       http.ResponseWriter
	Request      *http.Request

	server *msg2api.DeviceServer
}

func (api *wsDevApi) Run() error {
	var key []byte

	err := api.Db.View(func(tx db.Tx) error {
		user := tx.User(api.User)
		if user == nil {
			http.Error(api.Writer, notAuthorized.Error(), http.StatusUnauthorized)
			return notAuthorized
		}
		device := user.Device(api.Device)
		if device == nil {
			http.Error(api.Writer, notAuthorized.Error(), http.StatusUnauthorized)
			return notAuthorized
		}
		key = device.Key()
		key = append(make([]byte, 0, len(key)), key...)
		return nil
	})
	if err != nil {
		return err
	}

	server, err := msg2api.NewDeviceServer(api.Writer, api.Request)
	if err != nil {
		return err
	}

	api.server = server
	api.server.Update = api.doUpdate
	api.server.AddSensor = api.doAddSensor
	api.server.RemoveSensor = api.doRemoveSensor
	api.server.UpdateMetadata = api.doUpdateMetadata
	return api.server.Run(key)
}

func (api *wsDevApi) viewDevice(fn func(tx db.Tx, user db.User, device db.Device) *msg2api.Error) (err *msg2api.Error) {
	api.Db.View(func(tx db.Tx) error {
		u := tx.User(api.User)
		if u == nil {
			err = apiNotAuthorized
			return err
		}
		d := u.Device(api.Device)
		if d == nil {
			err = apiNotAuthorized
			return err
		}
		err = fn(tx, u, d)
		return err
	})
	return
}

func (api *wsDevApi) updateDevice(fn func(tx db.Tx, user db.User, device db.Device) *msg2api.Error) (err *msg2api.Error) {
	api.Db.Update(func(tx db.Tx) error {
		u := tx.User(api.User)
		if u == nil {
			err = apiNotAuthorized
			return err
		}
		d := u.Device(api.Device)
		if d == nil {
			err = apiNotAuthorized
			return err
		}
		err = fn(tx, u, d)
		return err
	})
	return
}

func (api *wsDevApi) doUpdate(values map[string][]msg2api.Measurement) *msg2api.Error {
	return api.viewDevice(func(tx db.Tx, user db.User, device db.Device) *msg2api.Error {
		for name, _ := range values {
			if device.Sensor(name) == nil {
				return &msg2api.Error{Code: "no sensor", Extra: name}
			}
		}

		for sensor, values := range values {
			s := device.Sensor(sensor)
			for _, value := range values {
				err := api.Db.AddReading(user, device, s, value.Time, value.Value)
				if err != nil {
					return &msg2api.Error{Code: "could not add readings"}
				}
				api.Hub.Publish(api.User, measurementWithMetadata{device.Id(), s.Id(), value.Time, value.Value})
			}
		}

		return nil
	})

	return nil
}

func (api *wsDevApi) doAddSensor(name string) *msg2api.Error {
	return api.updateDevice(func(tx db.Tx, user db.User, device db.Device) *msg2api.Error {
		_, err := device.AddSensor(name)
		if err != nil {
			return &msg2api.Error{Code: "operation failed", Extra: err.Error()}
		}
		return nil
	})
}

func (api *wsDevApi) doRemoveSensor(name string) *msg2api.Error {
	return api.updateDevice(func(tx db.Tx, user db.User, device db.Device) *msg2api.Error {
		if err := device.RemoveSensor(name); err != nil {
			return &msg2api.Error{Code: "operation failed", Extra: err.Error()}
		}
		api.Hub.Publish(api.User, msg2api.UserEventMetadataArgs{
			Devices: map[string]msg2api.DeviceMetadata{
				api.Device: msg2api.DeviceMetadata{
					DeletedSensors: map[string]*string{
						name: nil,
					},
				},
			},
		})
		return nil
	})
}

func (api *wsDevApi) doUpdateMetadata(metadata *msg2api.DeviceMetadata) *msg2api.Error {
	return api.updateDevice(func(tx db.Tx, user db.User, device db.Device) *msg2api.Error {
		if metadata.Name != "" {
			device.SetName(metadata.Name)
		}

		for sid, sname := range metadata.Sensors {
			dbs := device.Sensor(sid)
			if dbs == nil {
				return &msg2api.Error{Code: "no sensor", Extra: "sid"}
			}
			dbs.SetName(sname)
		}

		api.Hub.Publish(api.User, msg2api.UserEventMetadataArgs{
			Devices: map[string]msg2api.DeviceMetadata{
				api.Device: msg2api.DeviceMetadata(*metadata),
			},
		})

		return nil
	})
}

type wsProxyDevApi struct {
	Device, PostUrl, CertFile string
	Key                       []byte
	Writer                    http.ResponseWriter
	Request                   *http.Request

	server *msg2api.DeviceServer
	client *http.Client
}

func (api *wsProxyDevApi) Run() error {
	client := http.DefaultClient

	if api.CertFile != "" {
		cert, err := ioutil.ReadFile(api.CertFile)
		if err != nil {
			return err
		}

		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(cert) {
			return errors.New("could not parse cert")
		}

		tlsConfig := &tls.Config{
			RootCAs: certPool,
		}
		client = &http.Client{
			Transport: &http.Transport{TLSClientConfig: tlsConfig},
			Timeout:   2 * time.Second,
		}
	}

	api.client = client

	server, err := msg2api.NewDeviceServer(api.Writer, api.Request)
	if err != nil {
		return err
	}

	api.server = server
	api.server.Update = api.doUpdate
	return api.server.Run(api.Key)
}

func (api *wsProxyDevApi) doUpdate(values map[string][]msg2api.Measurement) *msg2api.Error {
	if len(values) != 1 {
		return &msg2api.Error{Code: "invalid input", Extra: "exactly one sensor required"}
	}

	var buf bytes.Buffer
	var sensor string
	buf.WriteString(`{"measurements":[`)
	for s, values := range values {
		sensor = s
		writtenAny := false
		for _, value := range values {
			if writtenAny {
				buf.WriteString(",")
			}
			buf.WriteString(fmt.Sprintf("[%v,%v]", value.Time.Unix(), value.Value))
			writtenAny = true
		}
	}
	buf.WriteString(`]}`)

	body := buf.Bytes()

	mac := hmac.New(sha1.New, []byte(api.Key))
	mac.Write(body)

	req, err := http.NewRequest("POST", api.PostUrl+sensor, &buf)
	if err != nil {
		return &msg2api.Error{Code: "operation failed", Extra: err.Error()}
	}
	req.Header["Content-Type"] = []string{"application/json"}
	req.Header["X-Version"] = []string{"1.0"}
	req.Header["X-Digest"] = []string{hex.EncodeToString(mac.Sum(nil))}

	resp, err := api.client.Do(req)
	if err != nil {
		return &msg2api.Error{Code: "operation failed", Extra: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return &msg2api.Error{Code: "operation failed", Extra: resp.Status}
	}

	respBody, _ := ioutil.ReadAll(resp.Body)
	respBodyStr := string(respBody)
	if respBodyStr != `{"response":"ok"}` {
		return &msg2api.Error{Code: "operation failed", Extra: respBodyStr}
	}

	return nil
}

type wsUserApi struct {
	User    string
	Db      db.Db
	Hub     *hub.Hub
	Writer  http.ResponseWriter
	Request *http.Request

	server msg2api.UserServer
}

func (api *wsUserApi) Run() error {
	conn := api.Hub.Connect()
	defer conn.Close()
	conn.Subscribe(api.User)

	go func() {
		for {
			val, open := <-conn.Value
			if !open {
				return
			}
			switch v := val.Data.(type) {
			case measurementWithMetadata:
				api.server.SendUpdate(map[string]map[string][]msg2api.Measurement{
					v.Device: {
						v.Sensor: {
							{v.Time, v.Value},
						},
					},
				})

			case msg2api.UserEventMetadataArgs:
				api.server.SendMetadata(v)

			default:
				log.Printf("bad hub value type %T\n", val.Data)
			}
		}
	}()

	api.server.GetValues = api.doGetValues
	return api.server.Run()
}

func (api *wsUserApi) doGetValues(since time.Time, withMetadata bool) error {
	return api.Db.View(func(tx db.Tx) error {
		user := tx.User(api.User)
		if user == nil {
			return notAuthorized
		}
		sensors := make(map[db.Device][]db.Sensor)
		for _, dev := range user.Devices() {
			smap := dev.Sensors()
			slist := make([]db.Sensor, 0, len(smap))
			for _, sensor := range smap {
				slist = append(slist, sensor)
			}
			sensors[dev] = slist
		}
		if withMetadata {
			meta := make(map[string]msg2api.DeviceMetadata)
			for did, dev := range user.Devices() {
				meta[did] = msg2api.DeviceMetadata{
					Name:    dev.Name(),
					Sensors: make(map[string]string),
				}
				for sid, sensor := range dev.Sensors() {
					meta[dev.Id()].Sensors[sid] = sensor.Name()
				}
			}
			if err := api.server.SendMetadata(msg2api.UserEventMetadataArgs{meta}); err != nil {
				return err
			}
		}
		readings, err := user.LoadReadings(since, sensors)
		if err != nil {
			return err
		}
		update := make(map[string]map[string][]msg2api.Measurement)
		for dev, svalues := range readings {
			dupdate := make(map[string][]msg2api.Measurement, len(svalues))
			update[dev.Id()] = dupdate
			for sensor, values := range svalues {
				supdate := make([]msg2api.Measurement, 0, len(values))
				for _, val := range values {
					supdate = append(supdate, msg2api.Measurement{val.Time, val.Value})
				}
				dupdate[sensor.Id()] = supdate
			}
		}
		return api.server.SendUpdate(update)
	})
}

package msgp

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/mysmartgrid/msg-prototype-2/db"
	"github.com/mysmartgrid/msg-prototype-2/hub"
	"github.com/mysmartgrid/msg2api"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

var (
	errNotAuthorized    = errors.New("not authorized")
	errAPINotAuthorized = &msg2api.Error{Code: errNotAuthorized.Error()}

	errDeviceNotRegistered     = errors.New("device not registered")
	errDeviceAlreadyRegistered = errors.New("device already registered")
)

type measurementWithMetadata struct {
	Device, Sensor string
	Time           time.Time
	Value          float64
	Resolution     string
}

// WsAPIContext is the basic container for the Websocket API.
// It holds the Hub and Database and allows device management.
type WsAPIContext struct {
	// Bd is the user database associated with the context
	Db db.Db
	// Hub is the communication hub all associated APIs are using.
	Hub *hub.Hub

	devices map[string]*WsDevAPI
	devMtx  sync.RWMutex
}

// RegisterDevice registers a new device, accessible via WsDevAPI at the API context.
// Returns the currently associated API and an error if the device is already registered
// or just the given API otherwise.
func (ctx *WsAPIContext) RegisterDevice(dev *WsDevAPI) (*WsDevAPI, error) {
	ctx.devMtx.Lock()
	defer ctx.devMtx.Unlock()

	if ctx.devices == nil {
		ctx.devices = make(map[string]*WsDevAPI)
	}

	if ctx.devices[dev.Device] != nil {
		return ctx.devices[dev.Device], errDeviceAlreadyRegistered
	}

	ctx.devices[dev.Device] = dev
	dev.ctx = ctx

	return dev, nil
}

// WithDevice executes function fn using the WsDevAPI for the device with the given id registered at the API context.
// Returns an error if the device is not registered.
func (ctx *WsAPIContext) WithDevice(device string, fn func(dev *WsDevAPI) error) error {
	ctx.devMtx.RLock()
	defer ctx.devMtx.RUnlock()

	dev := ctx.devices[device]
	if dev == nil {
		return errDeviceNotRegistered
	}

	return fn(dev)
}

// RemoveDevice removes the given WsDevAPI from the API context.
// Returns an error if the device API is not regsitered.
func (ctx *WsAPIContext) RemoveDevice(dev *WsDevAPI) error {
	ctx.devMtx.Lock()
	defer ctx.devMtx.Unlock()

	if ctx.devices[dev.Device] == nil {
		return errDeviceNotRegistered
	}
	delete(ctx.devices, dev.Device)
	return nil
}

// WsDevAPI represents a websocket connection for a device.
// It manages the msg2api device server, and all messages coming from the device.
type WsDevAPI struct {
	ctx    *WsAPIContext
	server *msg2api.DeviceServer

	lastRealtimeUpdateRequest time.Time

	// User and Device id associated with the connection
	User, Device string

	// HTTP connection to communicate with the client.
	Writer  http.ResponseWriter
	Request *http.Request

	// Secret device key for authentification
	Key []byte

	// URL and HTTP client to mirror values to the old mysmartgrid.
	PostURL    string
	PostClient *http.Client
}

// Run checks if the user associated to the API is authorized to access the associated device, posts an error to the HTTP connection and returns an error if not.
// On success a new device server is created an run.
func (api *WsDevAPI) Run() error {
	var key []byte

	err := api.ctx.Db.View(func(tx db.Tx) error {
		user := tx.User(api.User)
		if user == nil {
			http.Error(api.Writer, errNotAuthorized.Error(), http.StatusUnauthorized)
			return errNotAuthorized
		}
		device := user.Device(api.Device)
		if device == nil {
			http.Error(api.Writer, errNotAuthorized.Error(), http.StatusUnauthorized)
			return errNotAuthorized
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

// RequestRealtimeUpdates forwards a realtime updates request to the device if enough time has passed since the last request.
func (api *WsDevAPI) RequestRealtimeUpdates(req msg2api.DeviceCmdRequestRealtimeUpdatesArgs) {
	if time.Now().Sub(api.lastRealtimeUpdateRequest) >= 25*time.Second && len(req) > 0 {
		api.server.RequestRealtimeUpdates(req)
		api.lastRealtimeUpdateRequest = time.Now()
	}
}

// Close closes the device servers connection to the device.
func (api *WsDevAPI) Close() {
	if api.server != nil {
		api.server.Close()
	}
}

func (api *WsDevAPI) viewDevice(fn func(tx db.Tx, user db.User, device db.Device) *msg2api.Error) (err *msg2api.Error) {
	api.ctx.Db.View(func(tx db.Tx) error {
		u := tx.User(api.User)
		if u == nil {
			err = errAPINotAuthorized
			return err
		}
		d := u.Device(api.Device)
		if d == nil {
			err = errAPINotAuthorized
			return err
		}
		err = fn(tx, u, d)
		if err == nil {
			return nil
		}
		return err
	})
	return
}

func (api *WsDevAPI) updateDevice(fn func(tx db.Tx, user db.User, device db.Device) *msg2api.Error) (err *msg2api.Error) {
	api.ctx.Db.Update(func(tx db.Tx) error {
		u := tx.User(api.User)
		if u == nil {
			err = errAPINotAuthorized
			return err
		}
		d := u.Device(api.Device)
		if d == nil {
			err = errAPINotAuthorized
			return err
		}
		err = fn(tx, u, d)
		if err == nil {
			return nil
		}
		return err
	})
	return
}

func (api *WsDevAPI) postValuesToOldMSG(sensor string, values []msg2api.Measurement) *msg2api.Error {
	var buf bytes.Buffer
	buf.WriteString(`{"measurements":[`)
	writtenAny := false
	for _, value := range values {
		if writtenAny {
			buf.WriteString(",")
		}
		buf.WriteString(fmt.Sprintf("[%v,%v]", value.Time.Unix(), value.Value))
		writtenAny = true
	}
	buf.WriteString(`]}`)

	mac := hmac.New(sha1.New, api.Key)
	mac.Write(buf.Bytes())

	req, err := http.NewRequest("POST", api.PostURL+sensor, &buf)
	if err != nil {
		return &msg2api.Error{Code: "operation failed", Extra: err.Error()}
	}
	req.Header["Content-Type"] = []string{"application/json"}
	req.Header["X-Version"] = []string{"1.0"}
	req.Header["X-Digest"] = []string{hex.EncodeToString(mac.Sum(nil))}

	resp, err := api.PostClient.Do(req)
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

func (api *WsDevAPI) doUpdate(values map[string][]msg2api.Measurement) *msg2api.Error {
	if len(values) != 1 {
		return &msg2api.Error{Code: "invalid input", Extra: "exactly one sensor required"}
	}

	return api.viewDevice(func(tx db.Tx, user db.User, device db.Device) *msg2api.Error {
		for name := range values {
			if device.Sensor(name) == nil {
				return &msg2api.Error{Code: "no sensor", Extra: name}
			}
		}

		for sensor, values := range values {
			if sensor[len(sensor)-3:] == "/wh" {
				if err := api.postValuesToOldMSG(sensor[0:len(sensor)-3], values); err != nil {
					return err
				}
			}

			s := device.Sensor(sensor)
			for _, value := range values {
				err := api.ctx.Db.AddReading(s, value.Time, value.Value)
				if err != nil {
					return &msg2api.Error{Code: "could not add readings"}
				}

				if time.Now().Sub(api.lastRealtimeUpdateRequest) < 40*time.Second {
					api.ctx.Hub.Publish(api.User, measurementWithMetadata{device.ID(), s.ID(), value.Time, value.Value, "raw"})
				}
			}
		}

		return nil
	})
}

func (api *WsDevAPI) doAddSensor(name, unit string, port int32) *msg2api.Error {
	return api.updateDevice(func(tx db.Tx, user db.User, device db.Device) *msg2api.Error {
		_, err := device.AddSensor(name, unit, port)
		if err != nil {
			return &msg2api.Error{Code: "operation failed", Extra: err.Error()}
		}
		api.ctx.Hub.Publish(api.User, msg2api.UserEventMetadataArgs{
			Devices: map[string]msg2api.DeviceMetadata{
				api.Device: {
					Sensors: map[string]msg2api.SensorMetadata{
						name: {
							Name: &name,
							Unit: &unit,
							Port: &port,
						},
					},
				},
			},
		})
		return nil
	})
}

func (api *WsDevAPI) doRemoveSensor(name string) *msg2api.Error {
	return api.updateDevice(func(tx db.Tx, user db.User, device db.Device) *msg2api.Error {
		if err := device.RemoveSensor(name); err != nil {
			return &msg2api.Error{Code: "operation failed", Extra: err.Error()}
		}
		api.ctx.Hub.Publish(api.User, msg2api.UserEventMetadataArgs{
			Devices: map[string]msg2api.DeviceMetadata{
				api.Device: {
					DeletedSensors: map[string]*string{
						name: nil,
					},
				},
			},
		})
		return nil
	})
}

func (api *WsDevAPI) doUpdateMetadata(metadata *msg2api.DeviceMetadata) *msg2api.Error {
	return api.updateDevice(func(tx db.Tx, user db.User, device db.Device) *msg2api.Error {
		if metadata.Name != "" {
			device.SetName(metadata.Name)
		}

		for sid, sd := range metadata.Sensors {
			dbs := device.Sensor(sid)
			if dbs == nil {
				return &msg2api.Error{Code: "no sensor", Extra: "sid"}
			}
			if sd.Name != nil {
				if err := dbs.SetName(*sd.Name); err != nil {
					return &msg2api.Error{Code: "failed", Extra: err.Error()}
				}
			}
			if sd.Unit != nil {
				return &msg2api.Error{Code: "failed", Extra: "unit may not be changed"}
			}
			if sd.Port != nil {
				return &msg2api.Error{Code: "failed", Extra: "port may not be changed"}
			}
		}

		api.ctx.Hub.Publish(api.User, msg2api.UserEventMetadataArgs{
			Devices: map[string]msg2api.DeviceMetadata{
				api.Device: msg2api.DeviceMetadata(*metadata),
			},
		})

		return nil
	})
}

// WsUserAPI represents a websocket connection to a user.
// It manages the msg2api user server and user messages.
type WsUserAPI struct {
	Ctx    *WsAPIContext
	server *msg2api.UserServer

	// User id associated with the API.
	User string

	// HTTP connection to communicate with the client.
	Writer  http.ResponseWriter
	Request *http.Request
}

// Run listens for messages for the user at the Hub and starts the user server.
func (api *WsUserAPI) Run() error {
	conn := api.Ctx.Hub.Connect()
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
				api.server.SendUpdate(msg2api.UserEventUpdateArgs{
					Resolution: v.Resolution,
					Values: map[string]map[string][]msg2api.Measurement{
						v.Device: {
							v.Sensor: {
								{v.Time, v.Value},
							},
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

	server, err := msg2api.NewUserServer(api.Writer, api.Request)
	if err != nil {
		return err
	}

	api.server = server
	api.server.GetMetadata = api.doGetMetadata
	api.server.GetValues = api.doGetValues
	api.server.RequestRealtimeUpdates = api.doRequestRealtimeUpdates
	return api.server.Run()
}

// Close closes the user server connections.
func (api *WsUserAPI) Close() {
	if api.server != nil {
		api.server.Close()
	}
}

func (api *WsUserAPI) doGetMetadata() error {
	return api.Ctx.Db.View(func(tx db.Tx) error {
		user := tx.User(api.User)
		if user == nil {
			return errNotAuthorized
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

		meta := make(map[string]msg2api.DeviceMetadata)
		for did, dev := range user.Devices() {
			meta[did] = msg2api.DeviceMetadata{
				Name:    dev.Name(),
				Sensors: make(map[string]msg2api.SensorMetadata),
			}
			for sid, sensor := range dev.Sensors() {
				name := sensor.Name()
				unit := sensor.Unit()
				port := sensor.Port()
				meta[dev.ID()].Sensors[sid] = msg2api.SensorMetadata{
					Name: &name,
					Unit: &unit,
					Port: &port,
				}
			}
		}
		return api.server.SendMetadata(msg2api.UserEventMetadataArgs{Devices: meta})
	})
}

func (api *WsUserAPI) doGetValues(since, until time.Time, resolution string, sensors map[string][]string) error {
	return api.Ctx.Db.View(func(tx db.Tx) error {
		user := tx.User(api.User)
		if user == nil {
			return errNotAuthorized
		}

		readings, err := user.LoadReadings(since, until, resolution, sensors)
		if err != nil {
			return err
		}

		update := msg2api.UserEventUpdateArgs{
			Resolution: resolution,
			Values:     readings,
		}

		// Also send already aggregated second values as 'raw'
		if resolution == "raw" {
			secondReadings, err := user.LoadReadings(since, until, "second", sensors)
			if err != nil {
				return err
			}

			secondUpdate := msg2api.UserEventUpdateArgs{
				Resolution: resolution,
				Values:     secondReadings,
			}

			err = api.server.SendUpdate(secondUpdate)
			if err != nil {
				return err
			}
		}

		return api.server.SendUpdate(update)
	})
}

func (api *WsUserAPI) doRequestRealtimeUpdates(sensors map[string][]string) error {
	for dev, sensors := range sensors {
		err := api.Ctx.WithDevice(dev, func(dev *WsDevAPI) error {
			dev.RequestRealtimeUpdates(sensors)
			return nil
		})
		if err != nil && err != errDeviceNotRegistered {
			return err
		}
	}
	return nil
}

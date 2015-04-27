package msgp

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"io/ioutil"
	"log"
	"msgp/db"
	"msgp/hub"
	"msgp/ws"
	"net/http"
	"time"
)

type WSAPI struct {
	Db      db.Db
	Hub     *hub.Hub
	Writer  http.ResponseWriter
	Request *http.Request
	User    string

	dispatch *ws.Dispatcher
	protocol string
}

type wsDeviceAPI struct {
	*WSAPI

	Device string
}

type wsDeviceProxyAPI struct {
	*WSAPI

	Device  string
	Key     string
	PostUrl string

	client *http.Client
}

type wsUserAPI struct {
	*WSAPI
}

type v1MessageIn struct {
	Command string          `json:"cmd"`
	Error   *v1Error        `json:"error"`
	Args    json.RawMessage `json:"args"`
}

type v1MessageOut struct {
	Command string      `json:"cmd,omitempty"`
	Now     *int64      `json:"now,omitempty"`
	Error   *v1Error    `json:"error,omitempty"`
	Args    interface{} `json:"args,omitempty"`
}

type v1DeviceCmdUpdateArgs struct {
	Values map[string][]Measurement `json:"values"`
}

type v1DeviceCmdAddSensorArgs struct {
	Name string `json:"name"`
}

type v1DeviceCmdUpdateMetadataArgs v1DeviceMetadata

type v1UserCmdGetValuesArgs struct {
	SinceUnixMs  float64 `json:"since"`
	WithMetadata bool    `json:"withMetadata"`
}

type v1UserEventUpdateArgs struct {
	Values map[string]map[string][]Measurement `json:"values"`
}

type v1UserEventMetadataArgs struct {
	Devices map[string]v1DeviceMetadata `json:"devices"`
}

type v1DeviceMetadata struct {
	Name           string             `json:"name,omitempty"`
	Sensors        map[string]string  `json:"sensors,omitempty"`
	DeletedSensors map[string]*string `json:"deletedSensors,omitempty"`
}

type v1Error struct {
	Code        string      `json:"error"`
	Description string      `json:"description,omitempty"`
	Extra       interface{} `json:"extra,omitempty"`
}

type Measurement struct {
	Time  time.Time
	Value float64
}

type measurementWithMetadata struct {
	Device, Sensor string
	Time           time.Time
	Value          float64
}

const (
	upgradeTimeout = 10 * time.Second

	deviceApiProtocolV1 = "v1.device.msg"
	userApiProtocolV1   = "v1.user.msg"
)

var (
	deviceApiProtocols = []string{deviceApiProtocolV1}
	userApiProtocols   = []string{userApiProtocolV1}

	methodNotAllowed    = errors.New("method not allowed")
	notAuthorized       = errors.New("not authorized")
	unsupportedProtocol = errors.New("unsupported protocol")
	protocolViolation   = errors.New("protocol violation")
	noSensor            = errors.New("sensor not found")

	apiNotAuthorized = &v1Error{Code: "not authorized"}
)

func badCommand(cmd string) *v1Error {
	return &v1Error{Code: "bad command", Extra: cmd}
}

func invalidInput(desc, extra string) *v1Error {
	return &v1Error{Code: "invalid input", Description: desc, Extra: extra}
}

func operationFailed(extra string) *v1Error {
	return &v1Error{Code: "operation failed", Extra: extra}
}

func (e *v1Error) Error() string {
	result := e.Code
	if e.Description != "" {
		result = fmt.Sprintf("%v (%v)", result, e.Description)
	}
	if e.Extra != nil {
		result = fmt.Sprintf("%v [%v]", result, e.Extra)
	}
	return result
}

func (api *WSAPI) prepare(protocols []string, checkUserDb bool) error {
	if api.dispatch != nil {
		return nil
	}

	if api.Request.Method != "GET" {
		http.Error(api.Writer, methodNotAllowed.Error(), http.StatusMethodNotAllowed)
		return methodNotAllowed
	}

	if checkUserDb {
		err := api.Db.View(func(tx db.Tx) error {
			user := tx.User(api.User)
			if user == nil {
				http.Error(api.Writer, notAuthorized.Error(), http.StatusUnauthorized)
				return notAuthorized
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	upgrader := websocket.Upgrader{
		HandshakeTimeout: upgradeTimeout,
		Subprotocols:     protocols,
	}
	conn, err := upgrader.Upgrade(api.Writer, api.Request, nil)
	if err != nil {
		return err
	}

	conn.SetReadLimit(4096)

	api.dispatch = &ws.Dispatcher{
		Socket: conn,
	}

	if conn.Subprotocol() == "" {
		return unsupportedProtocol
	}

	api.protocol = conn.Subprotocol()
	return nil
}

func (api *WSAPI) RunDevice(device string) error {
	if err := api.prepare(deviceApiProtocols, true); err != nil {
		return err
	}
	devapi := wsDeviceAPI{api, device}
	return devapi.Run()
}

func (api *WSAPI) RunDeviceProxy(device, key, postUrl, certFile string) error {
	if err := api.prepare(deviceApiProtocols, false); err != nil {
		return err
	}

	client := http.DefaultClient

	if certFile != "" {
		cert, err := ioutil.ReadFile(certFile)
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
			Timeout: 2 * time.Second,
		}
	}

	devapi := wsDeviceProxyAPI{api, device, key, postUrl, client}
	return devapi.Run()
}

func (api *WSAPI) RunUser() error {
	if err := api.prepare(userApiProtocols, true); err != nil {
		return err
	}
	uapi := wsUserAPI{api}
	return uapi.Run()
}

func (api *WSAPI) Close() {
	if api.dispatch != nil {
		api.dispatch.Close()
		api.dispatch = nil
	}
}

func (api *wsDeviceAPI) viewDevice(fn func(tx db.Tx, user db.User, device db.Device) *v1Error) (result *v1Error) {
	api.Db.View(func(tx db.Tx) error {
		u := tx.User(api.User)
		if u == nil {
			result = apiNotAuthorized
			return nil
		}
		d := u.Device(api.Device)
		if d == nil {
			result = apiNotAuthorized
			return nil
		}
		result = fn(tx, u, d)
		return nil
	})
	return
}

func (api *wsDeviceAPI) updateDevice(fn func(tx db.Tx, user db.User, device db.Device) *v1Error) (result *v1Error) {
	api.Db.Update(func(tx db.Tx) error {
		u := tx.User(api.User)
		if u == nil {
			result = apiNotAuthorized
			return notAuthorized
		}
		d := u.Device(api.Device)
		if d == nil {
			result = apiNotAuthorized
			return notAuthorized
		}
		result = fn(tx, u, d)
		return nil
	})
	return
}

func (api *wsDeviceAPI) authenticateDevice() (result error) {
	var buf [sha256.Size]byte

	if _, err := rand.Read(buf[:]); err != nil {
		return err
	}

	challenge := hex.EncodeToString(buf[:])
	api.dispatch.Write(challenge)

	msgType, msg, err := api.dispatch.Receive()
	switch {
	case err != nil:
		return err
	case msgType != websocket.TextMessage:
		return protocolViolation
	}

	msg, err = hex.DecodeString(string(msg))
	if err != nil {
		return err
	}

	aerr := api.viewDevice(func(tx db.Tx, user db.User, device db.Device) *v1Error {
		mac := hmac.New(sha256.New, device.Key())
		mac.Write(buf[:])
		expected := mac.Sum(nil)
		if !hmac.Equal(msg, expected) {
			result = notAuthorized
			return apiNotAuthorized
		}
		result = api.dispatch.Write("proceed")
		return nil
	})
	if aerr != nil {
		return errors.New(aerr.Code)
	}
	return
}

func (api *wsDeviceAPI) Run() error {
	var err error

	if err = api.authenticateDevice(); err != nil {
		goto fail
	}

	for {
		var msg v1MessageIn

		if err = api.dispatch.ReceiveJSON(&msg); err != nil {
			goto fail
		}

		var apiErr *v1Error

		switch msg.Command {
		case "update":
			apiErr = api.doUpdate(&msg)
		case "addSensor":
			apiErr = api.doAddSensor(&msg)
		case "removeSensor":
			apiErr = api.doRemoveSensor(&msg)
		case "updateMetadata":
			apiErr = api.doUpdateMetadata(&msg)
		default:
			apiErr = badCommand(msg.Command)
		}

		if apiErr != nil {
			api.dispatch.WriteJSON(v1MessageOut{Error: apiErr})
		} else {
			now := time.Now().Unix()
			api.dispatch.WriteJSON(v1MessageOut{Now: &now})
		}
	}

	return nil

fail:
	api.dispatch.CloseWith(websocket.CloseProtocolError, err.Error())
	api.dispatch = nil
	return err
}

func (api *wsDeviceAPI) doUpdate(msg *v1MessageIn) *v1Error {
	var args v1DeviceCmdUpdateArgs

	if err := json.Unmarshal(msg.Args, &args); err != nil {
		return invalidInput(err.Error(), "")
	}

	return api.viewDevice(func(tx db.Tx, user db.User, device db.Device) *v1Error {
		for name, _ := range args.Values {
			if device.Sensor(name) == nil {
				return invalidInput(noSensor.Error(), name)
			}
		}

		for sensor, values := range args.Values {
			s := device.Sensor(sensor)
			for _, value := range values {
				err := api.Db.AddReading(user, device, s, value.Time, value.Value)
				if err != nil {
					return operationFailed(err.Error())
				}
				api.Hub.Publish(api.User, measurementWithMetadata{device.Id(), s.Id(), value.Time, value.Value})
			}
		}

		return nil
	})

	return nil
}

func (api *wsDeviceAPI) doAddSensor(msg *v1MessageIn) *v1Error {
	var args v1DeviceCmdAddSensorArgs

	if err := json.Unmarshal(msg.Args, &args); err != nil {
		return invalidInput(err.Error(), "")
	}

	return api.updateDevice(func(tx db.Tx, user db.User, device db.Device) *v1Error {
		_, err := device.AddSensor(args.Name)
		if err != nil {
			return operationFailed(err.Error())
		}
		return nil
	})
}

func (api *wsDeviceAPI) doRemoveSensor(msg *v1MessageIn) *v1Error {
	var args string

	if err := json.Unmarshal(msg.Args, &args); err != nil {
		return invalidInput(err.Error(), "")
	}

	return api.updateDevice(func(tx db.Tx, user db.User, device db.Device) *v1Error {
		if err := device.RemoveSensor(args); err != nil {
			return operationFailed(err.Error())
		}
		api.Hub.Publish(api.User, v1UserEventMetadataArgs{
			Devices: map[string]v1DeviceMetadata{
				api.Device: v1DeviceMetadata{
					DeletedSensors: map[string]*string{
						args: nil,
					},
				},
			},
		})
		return nil
	})
}

func (api *wsDeviceAPI) doUpdateMetadata(msg *v1MessageIn) *v1Error {
	var args v1DeviceCmdUpdateMetadataArgs

	if err := json.Unmarshal(msg.Args, &args); err != nil {
		return invalidInput(err.Error(), "")
	}

	return api.updateDevice(func(tx db.Tx, user db.User, device db.Device) *v1Error {
		if args.Name != "" {
			device.SetName(args.Name)
		}

		for sid, sname := range args.Sensors {
			dbs := device.Sensor(sid)
			if dbs == nil {
				return operationFailed("unknown sensor " + sid)
			}
			dbs.SetName(sname)
		}

		api.Hub.Publish(api.User, v1UserEventMetadataArgs{
			Devices: map[string]v1DeviceMetadata{
				api.Device: v1DeviceMetadata(args),
			},
		})

		return nil
	})
}

func (api *wsDeviceProxyAPI) authenticateDevice() (result error) {
	var buf [sha256.Size]byte

	if _, err := rand.Read(buf[:]); err != nil {
		return err
	}

	challenge := hex.EncodeToString(buf[:])
	api.dispatch.Write(challenge)

	msgType, msg, err := api.dispatch.Receive()
	switch {
	case err != nil:
		return err
	case msgType != websocket.TextMessage:
		return protocolViolation
	}

	msg, err = hex.DecodeString(string(msg))
	if err != nil {
		return err
	}

	devkey, err := hex.DecodeString(api.Key)
	if err != nil {
		return err
	}

	mac := hmac.New(sha256.New, devkey)
	mac.Write(buf[:])
	expected := mac.Sum(nil)

	if !hmac.Equal(msg, expected) {
		result = notAuthorized
		return
	}
	result = api.dispatch.Write("proceed")
	return
}

func (api *wsDeviceProxyAPI) Run() error {
	var err error

	if err = api.authenticateDevice(); err != nil {
		goto fail
	}

	for {
		var msg v1MessageIn

		if err = api.dispatch.ReceiveJSON(&msg); err != nil {
			goto fail
		}

		var apiErr *v1Error

		switch msg.Command {
		case "update":
			apiErr = api.doUpdate(&msg)
		default:
			apiErr = badCommand(msg.Command)
		}

		if apiErr != nil {
			api.dispatch.WriteJSON(v1MessageOut{Error: apiErr})
		} else {
			now := time.Now().Unix()
			api.dispatch.WriteJSON(v1MessageOut{Now: &now})
		}
	}

	return nil

fail:
	api.dispatch.CloseWith(websocket.CloseProtocolError, err.Error())
	api.dispatch = nil
	return err
}

func (api *wsDeviceProxyAPI) doUpdate(msg *v1MessageIn) *v1Error {
	var args v1DeviceCmdUpdateArgs

	if err := json.Unmarshal(msg.Args, &args); err != nil {
		return invalidInput(err.Error(), "")
	}

	if len(args.Values) != 1 {
		return invalidInput("exactly one sensor required", "")
	}

	var buf bytes.Buffer
	var sensor string
	buf.WriteString(`{"measurements":[`)
	for s, values := range args.Values {
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

	req, err := http.NewRequest("POST", api.PostUrl + sensor, &buf)
	if err != nil {
		return operationFailed(err.Error())
	}
	req.Header["Content-Type"] = []string{"application/json"}
	req.Header["X-Version"] = []string{"1.0"}
	req.Header["X-Digest"] = []string{hex.EncodeToString(mac.Sum(nil))}

	resp, err := api.client.Do(req)
	if err != nil {
		return operationFailed(err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return operationFailed(resp.Status)
	}

	respBody, _ := ioutil.ReadAll(resp.Body)
	respBodyStr := string(respBody)
	if respBodyStr != `{"response":"ok"}` {
		return operationFailed(respBodyStr)
	}

	return nil
}

func (api *wsUserAPI) Run() error {
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
				api.sendUpdate(map[string]map[string][]Measurement{
					v.Device: {
						v.Sensor: {
							{v.Time, v.Value},
						},
					},
				})

			case v1UserEventMetadataArgs:
				api.dispatch.WriteJSON(v1MessageOut{Command: "metadata", Args: v})

			default:
				log.Printf("bad hub value type %T\n", val.Data)
			}
		}
	}()

	for {
		var msg v1MessageIn

		if err := api.dispatch.ReceiveJSON(&msg); err != nil {
			api.dispatch.CloseWith(websocket.CloseProtocolError, err.Error())
			api.dispatch = nil
			return err
		}

		var apiErr *v1Error

		switch msg.Command {
		case "getValues":
			apiErr = api.doGetValues(&msg)

		default:
			api.dispatch.WriteJSON(v1MessageOut{Error: badCommand(msg.Command)})
		}

		if apiErr != nil {
			api.dispatch.WriteJSON(v1MessageOut{Error: apiErr})
		}
	}

	return nil
}

func (api *wsUserAPI) sendUpdate(values map[string]map[string][]Measurement) error {
	return api.dispatch.WriteJSON(v1MessageOut{Command: "update", Args: values})
}

func (api *wsUserAPI) doGetValues(cmd *v1MessageIn) (result *v1Error) {
	var args v1UserCmdGetValuesArgs
	var err error

	if err = json.Unmarshal(cmd.Args, &args); err != nil {
		goto fail
	}

	err = api.Db.View(func(tx db.Tx) error {
		user := tx.User(api.User)
		if user == nil {
			result = apiNotAuthorized
			return nil
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
		if args.WithMetadata {
			meta := make(map[string]v1DeviceMetadata)
			for did, dev := range user.Devices() {
				meta[did] = v1DeviceMetadata{
					Name:    dev.Name(),
					Sensors: make(map[string]string),
				}
				for sid, sensor := range dev.Sensors() {
					meta[dev.Id()].Sensors[sid] = sensor.Name()
				}
			}
			if err := api.dispatch.WriteJSON(v1MessageOut{Command: "metadata", Args: v1UserEventMetadataArgs{meta}}); err != nil {
				return err
			}
		}
		readings, err := user.LoadReadings(
			time.Unix(
				int64(args.SinceUnixMs/1000),
				int64(args.SinceUnixMs)%1000*1e6),
			sensors)
		if err != nil {
			return err
		}
		update := make(map[string]map[string][]Measurement)
		for dev, svalues := range readings {
			dupdate := make(map[string][]Measurement, len(svalues))
			update[dev.Id()] = dupdate
			for sensor, values := range svalues {
				supdate := make([]Measurement, 0, len(values))
				for _, val := range values {
					supdate = append(supdate, Measurement{val.Time, val.Value})
				}
				dupdate[sensor.Id()] = supdate
			}
		}
		return api.sendUpdate(update)
	})

fail:
	if err != nil {
		result = operationFailed(err.Error())
	}
	return result
}

func (p *Measurement) UnmarshalJSON(data []byte) error {
	var arr [2]float64

	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}

	ms := int64(arr[0])
	p.Time = time.Unix(ms/1000, (ms%1000)*1e6)
	p.Value = arr[1]
	return nil
}

func (p *Measurement) MarshalJSON() ([]byte, error) {
	return json.Marshal([2]float64{float64(jsTime(p.Time)), p.Value})
}

func jsTime(time time.Time) int64 {
	return 1000*time.Unix() + int64(time.Nanosecond()/1e6)
}

type wsClient struct {
	dispatch *ws.Dispatcher
	protocol string
}

type wsClientDevice struct {
	wsClient
}

type WSClientDevice interface {
	Close()

	AddSensor(name string) error
	Update(values map[string][]Measurement) error

	Rename(name string) error
	RenameSensor(id, name string) error
	RemoveSensor(id string) error
}

func (c *wsClient) prepare(url string, protocols []string, tlsConfig *tls.Config) error {
	if c.dispatch != nil {
		return nil
	}

	dialer := websocket.Dialer{
		TLSClientConfig: tlsConfig,
		Subprotocols:    protocols,
	}
	sock, _, err := dialer.Dial(url, nil)
	if err != nil {
		return err
	}

	c.dispatch = &ws.Dispatcher{
		Socket: sock,
	}

	if sock.Subprotocol() == "" {
		return unsupportedProtocol
	}

	c.protocol = sock.Subprotocol()
	return nil
}

func (c *wsClient) Close() {
	c.dispatch.Close()
}

func NewWSClientDevice(url, user, device string, key []byte, tlsConfig *tls.Config) (WSClientDevice, error) {
	result := new(wsClientDevice)

	if err := result.prepare(url+"/ws/device/"+user+"/"+device, deviceApiProtocols, tlsConfig); err != nil {
		return nil, err
	}

	if err := result.authenticate(key); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *wsClientDevice) authenticate(key []byte) error {
	msgType, msg, err := c.dispatch.Receive()
	switch {
	case err != nil:
		return err
	case msgType != websocket.TextMessage:
		return protocolViolation
	}

	challenge, err := hex.DecodeString(string(msg))
	if err != nil {
		return err
	}

	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(challenge)
	response := hex.EncodeToString(mac.Sum(nil))
	if err := c.dispatch.Write(response); err != nil {
		return err
	}

	msgType, msg, err = c.dispatch.Receive()
	switch {
	case err != nil:
		return err
	case msgType != websocket.TextMessage:
		return protocolViolation
	case string(msg) != "proceed":
		return errors.New(string(msg))
	}

	return nil
}

func (c *wsClientDevice) executeCommand(cmd *v1MessageOut) error {
	if err := c.dispatch.WriteJSON(cmd); err != nil {
		return err
	}

	var result v1MessageIn
	if err := c.dispatch.ReceiveJSON(&result); err != nil {
		return err
	}
	if result.Error == nil {
		return nil
	}
	return result.Error
}

func (c *wsClientDevice) Update(values map[string][]Measurement) error {
	cmd := v1MessageOut{
		Command: "update",
		Args:    v1DeviceCmdUpdateArgs{values},
	}

	return c.executeCommand(&cmd)
}

func (c *wsClientDevice) AddSensor(name string) error {
	cmd := v1MessageOut{
		Command: "addSensor",
		Args: v1DeviceCmdAddSensorArgs{
			Name: name,
		},
	}

	return c.executeCommand(&cmd)
}

func (c *wsClientDevice) Rename(name string) error {
	cmd := v1MessageOut{
		Command: "updateMetadata",
		Args: v1DeviceCmdUpdateMetadataArgs{
			Name:    name,
			Sensors: nil,
		},
	}

	return c.executeCommand(&cmd)
}

func (c *wsClientDevice) RenameSensor(id, name string) error {
	cmd := v1MessageOut{
		Command: "updateMetadata",
		Args: v1DeviceCmdUpdateMetadataArgs{
			Sensors: map[string]string{
				id: name,
			},
		},
	}

	return c.executeCommand(&cmd)
}

func (c *wsClientDevice) RemoveSensor(id string) error {
	cmd := v1MessageOut{
		Command: "removeSensor",
		Args:    id,
	}

	return c.executeCommand(&cmd)
}

package msg2api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
)

type DeviceServer struct {
	*apiBase

	Update         func(values map[string][]Measurement) *Error
	AddSensor      func(name, unit string, port int32) *Error
	RemoveSensor   func(name string) *Error
	UpdateMetadata func(metadata *DeviceMetadata) *Error
}

var authenticationFailed = errors.New("authentication failed")

func (d *DeviceServer) authenticate(key []byte) error {
	var buf [sha256.Size]byte

	if _, err := rand.Read(buf[:]); err != nil {
		return err
	}

	challenge := hex.EncodeToString(buf[:])
	d.socket.Write(challenge)

	msgRaw, err := d.socket.Receive()
	switch {
	case err != nil:
		return err
	}

	msg, err := hex.DecodeString(string(msgRaw))
	if err != nil {
		return err
	}

	mac := hmac.New(sha256.New, key)
	mac.Write(buf[:])
	expected := mac.Sum(nil)
	if !hmac.Equal(msg, expected) {
		return authenticationFailed
	}
	return d.socket.Write("proceed")
}

func (d *DeviceServer) Run(key []byte) error {
	var err error

	if err = d.authenticate(key); err != nil {
		goto fail
	}

	for {
		var msg MessageIn

		if err = d.socket.ReceiveJSON(&msg); err != nil {
			goto fail
		}

		var opError *Error

		switch msg.Command {
		case "update":
			opError = d.doUpdate(&msg)
		case "addSensor":
			opError = d.doAddSensor(&msg)
		case "removeSensor":
			opError = d.doRemoveSensor(&msg)
		case "updateMetadata":
			opError = d.doUpdateMetadata(&msg)
		default:
			opError = badCommand(msg.Command)
		}

		if opError != nil {
			d.socket.WriteJSON(MessageOut{Error: opError})
		} else {
			now := time.Now().UnixNano() / 1e6
			d.socket.WriteJSON(MessageOut{Now: &now})
		}
	}

fail:
	d.socket.Close(websocket.CloseProtocolError, err.Error())
	return err
}

func (d *DeviceServer) RequestRealtimeUpdates(sensors []string) {
	d.socket.WriteJSON(MessageOut{Command: "requestRealtimeUpdates", Args: sensors})
}

func (d *DeviceServer) doUpdate(msg *MessageIn) *Error {
	var args DeviceCmdUpdateArgs

	if err := json.Unmarshal(msg.Args, &args); err != nil {
		return invalidInput(err.Error(), "")
	}

	if d.Update == nil {
		return operationFailed("not supported")
	}

	return d.Update(args.Values)
}

func (d *DeviceServer) doAddSensor(msg *MessageIn) *Error {
	var args DeviceCmdAddSensorArgs

	if err := json.Unmarshal(msg.Args, &args); err != nil {
		return invalidInput(err.Error(), "")
	}

	if d.AddSensor == nil {
		return operationFailed("not supported")
	}

	return d.AddSensor(args.Name, args.Unit, args.Port)
}

func (d *DeviceServer) doRemoveSensor(msg *MessageIn) *Error {
	var args DeviceCmdRemoveSensorArgs

	if err := json.Unmarshal(msg.Args, &args); err != nil {
		return invalidInput(err.Error(), "")
	}

	if d.RemoveSensor == nil {
		return operationFailed("not supported")
	}

	return d.RemoveSensor(args.Name)
}

func (d *DeviceServer) doUpdateMetadata(msg *MessageIn) *Error {
	var args DeviceMetadata

	if err := json.Unmarshal(msg.Args, &args); err != nil {
		return invalidInput(err.Error(), "")
	}

	if d.UpdateMetadata == nil {
		return operationFailed("not supported")
	}

	md := DeviceMetadata(args)
	return d.UpdateMetadata(&md)
}

func NewDeviceServer(w http.ResponseWriter, r *http.Request) (*DeviceServer, error) {
	base, err := initApiBaseFromHttp(w, r, []string{deviceApiProtocolV1})
	if err != nil {
		return nil, err
	}

	result := &DeviceServer{
		apiBase: base,
	}
	return result, nil
}

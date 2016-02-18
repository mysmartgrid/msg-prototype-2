package msg2api

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
)

type unknownCommand struct {
	cmd string
}

func (b unknownCommand) Error() string {
	return "received unknown command " + b.cmd
}

// DeviceClient contains the websocket connection to the DeviceServer and
// stores handler functions to handle server messages.
type DeviceClient struct {
	*apiBase

	// RequestRealtimeUpdates handles requests for realtime updates for the given array of sensor IDs.
	RequestRealtimeUpdates func(sensors []string)
}

func (c *DeviceClient) waitForServer() (result *MessageIn, err error) {
	result = new(MessageIn)
	if err = c.socket.ReceiveJSON(&result); err != nil {
		return
	}

	if result.Error != nil {
		return nil, result.Error
	}

	switch result.Command {
	case "":
		break

	case "requestRealtimeUpdates":
		if c.RequestRealtimeUpdates != nil {
			var sensors DeviceCmdRequestRealtimeUpdatesArgs
			if err = json.Unmarshal(result.Args, &sensors); err != nil {
				return
			}
			c.RequestRealtimeUpdates(sensors)
		}
		return nil, nil

	default:
		return nil, unknownCommand{result.Command}
	}

	return result, nil
}

func (c *DeviceClient) executeCommand(cmd *MessageOut) error {
	if err := c.socket.WriteJSON(cmd); err != nil {
		return err
	}

	for {
		result, err := c.waitForServer()
		if err != nil {
			return err
		}
		if result != nil {
			return nil
		}
	}
}

// RunOnce waits for one command from the server, handles it and then returns.
func (c *DeviceClient) RunOnce() error {
	_, err := c.waitForServer()
	if err != nil {
		return err
	}
	return nil
}

// Close closes the websocket connection to the DeviceServer
func (c *DeviceClient) Close() {
	c.socket.Close(websocket.CloseGoingAway, "")
}

// AddSensor registers a new sensor at the server.
// This only has to be doen once for every sensor and is then stored.
func (c *DeviceClient) AddSensor(name, unit string, port int32) error {
	cmd := MessageOut{
		Command: "addSensor",
		Args: DeviceCmdAddSensorArgs{
			Name: name,
			Unit: unit,
			Port: port,
		},
	}

	return c.executeCommand(&cmd)
}

// Update sends a set of measurements to the server.
// 'values' maps sensor IDs to measurements.
func (c *DeviceClient) Update(values map[string][]Measurement) error {
	cmd := MessageOut{
		Command: "update",
		Args:    DeviceCmdUpdateArgs{values},
	}

	return c.executeCommand(&cmd)
}

// Rename sends a metadata update containing the device name to the server.
func (c *DeviceClient) Rename(name string) error {
	cmd := MessageOut{
		Command: "updateMetadata",
		Args: DeviceCmdUpdateMetadataArgs{
			Name:    name,
			Sensors: nil,
		},
	}

	return c.executeCommand(&cmd)
}

// UpdateSensor sends a metadata update for the given sensor ID to the server.
func (c *DeviceClient) UpdateSensor(id string, md SensorMetadata) error {
	cmd := MessageOut{
		Command: "updateMetadata",
		Args: DeviceCmdUpdateMetadataArgs{
			Sensors: map[string]SensorMetadata{
				id: md,
			},
		},
	}

	return c.executeCommand(&cmd)
}

// RemoveSensor deregisters the sensor ID on the server.
func (c *DeviceClient) RemoveSensor(id string) error {
	cmd := MessageOut{
		Command: "removeSensor",
		Args: DeviceCmdRemoveSensorArgs{
			Name: id,
		},
	}

	return c.executeCommand(&cmd)
}

func (c *DeviceClient) authenticate(key []byte) error {
	msg, err := c.socket.Receive()
	if err != nil {
		return err
	}

	challenge, err := hex.DecodeString(msg)
	if err != nil {
		return err
	}

	mac := hmac.New(sha256.New, key)
	mac.Write(challenge)
	response := hex.EncodeToString(mac.Sum(nil))
	if err := c.socket.Write(response); err != nil {
		return err
	}

	msg, err = c.socket.Receive()
	switch {
	case err != nil:
		return err
	case msg != "proceed":
		return errors.New(msg)
	}

	return nil
}

// NewDeviceClient opens a new websocket connection on the given url, tries an authentication
// with the given key and returns a new DeviceClient on success.
func NewDeviceClient(url string, key []byte, tlsConfig *tls.Config) (*DeviceClient, error) {
	dialer := websocket.Dialer{
		TLSClientConfig: tlsConfig,
		Subprotocols:    []string{deviceApiProtocolV1},
	}
	sock, _, err := dialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}

	wrap, err := initApiBaseFromSocket(sock)
	if err != nil {
		return nil, err
	}
	result := &DeviceClient{
		apiBase: wrap,
	}

	if err := result.authenticate(key); err != nil {
		return nil, err
	}

	return result, nil
}

package msg2api

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"github.com/gorilla/websocket"
)

type DeviceClient interface {
	Close()

	AddSensor(name string) error
	Update(values map[string][]Measurement) error

	Rename(name string) error
	RenameSensor(id, name string) error
	RemoveSensor(id string) error
}

type deviceClient struct {
	*apiBase
}

func (c *deviceClient) executeCommand(cmd *MessageOut) error {
	if err := c.socket.WriteJSON(cmd); err != nil {
		return err
	}

	var result MessageIn
	if err := c.socket.ReceiveJSON(&result); err != nil {
		return err
	}
	if result.Error == nil {
		return nil
	}
	return result.Error
}

func (c *deviceClient) Close() {
	c.socket.Close(websocket.CloseGoingAway, "")
}

func (c *deviceClient) AddSensor(name string) error {
	cmd := MessageOut{
		Command: "addSensor",
		Args: DeviceCmdAddSensorArgs{
			Name: name,
		},
	}

	return c.executeCommand(&cmd)
}

func (c *deviceClient) Update(values map[string][]Measurement) error {
	cmd := MessageOut{
		Command: "update",
		Args:    DeviceCmdUpdateArgs{values},
	}

	return c.executeCommand(&cmd)
}

func (c *deviceClient) Rename(name string) error {
	cmd := MessageOut{
		Command: "updateMetadata",
		Args: DeviceCmdUpdateMetadataArgs{
			Name:    name,
			Sensors: nil,
		},
	}

	return c.executeCommand(&cmd)
}

func (c *deviceClient) RenameSensor(id, name string) error {
	cmd := MessageOut{
		Command: "updateMetadata",
		Args: DeviceCmdUpdateMetadataArgs{
			Sensors: map[string]string{
				id: name,
			},
		},
	}

	return c.executeCommand(&cmd)
}

func (c *deviceClient) RemoveSensor(id string) error {
	cmd := MessageOut{
		Command: "removeSensor",
		Args:    id,
	}

	return c.executeCommand(&cmd)
}

func (c *deviceClient) authenticate(key []byte) error {
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

func NewDeviceClient(url string, key []byte, tlsConfig *tls.Config) (DeviceClient, error) {
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
	result := &deviceClient{wrap}

	if err := result.authenticate(key); err != nil {
		return nil, err
	}

	return result, nil
}

package msg2api

import (
	"encoding/json"
	"fmt"
)

// MessageIn describes a genric incoming message.
// It can contain a command and/or an Error, as well as arguments in json encoded form.
type MessageIn struct {
	// Command
	Command string          `json:"cmd"`
	Error   *Error          `json:"error"`
	Args    json.RawMessage `json:"args"`
}

// MessageOut describes a genric outgoing message.
// All fields are optinal.
type MessageOut struct {
	Command string `json:"cmd,omitempty"`

	// Now should contain the corrent server time.
	Now   *int64      `json:"now,omitempty"`
	Error *Error      `json:"error,omitempty"`
	Args  interface{} `json:"args,omitempty"`
}

// DeviceCmdUpdateArgs describes the Args field of a "udpate" message from the device.
type DeviceCmdUpdateArgs struct {
	// Values maps sensor IDs to measurement arrays.
	Values map[string][]Measurement `json:"values"`
}

// DeviceCmdAddSensorArgs describes the Args field of a "addSensor" message from the device.
type DeviceCmdAddSensorArgs struct {
	Name string `json:"name"`
	Unit string `json:"unit"`
	Port int32  `json:"port"`
}

// DeviceCmdRemoveSensorArgs describes the Args field of a "removeSensor" message from the device.
// It has the same structure as DeviceCmdAddSensorArgs.
type DeviceCmdRemoveSensorArgs DeviceCmdAddSensorArgs

// DeviceCmdUpdateMetadataArgs describes the Args field of a "updateMetadata" message from the device.
// It contains only one DeviceMetadata struct.
type DeviceCmdUpdateMetadataArgs DeviceMetadata

// DeviceCmdRequestRealtimeUpdatesArgs describes the args field of a "requestRealtimeUpdates" message to the device.
// It contains only an array of sensor IDs.
type DeviceCmdRequestRealtimeUpdatesArgs []string

// UserCmdGetValuesArgs describes the Args field of a "getValues" message from the user.
type UserCmdGetValuesArgs struct {
	SinceUnixMs    float64 `json:"since"`
	UntilUnixMs    float64 `json:"until"`
	TimeResolution string  `json:"resolution"`

	// Sensors maps resolutions to sensor arrays
	Sensors map[string][]string `json:"sensors"`
}

// UserCmdRequestRealtimeUpdatesArgs describes the Args field of a realtime update request from the user.
// It maps device IDs to sensor arrays.
type UserCmdRequestRealtimeUpdatesArgs map[string][]string

// UserEventUpdateArgs describes the Args field of an "update" message to the user.
type UserEventUpdateArgs struct {
	Resolution string `json:"resolution"`

	// Values maps Device IDs to Sensor ID to Measurement array.
	Values map[string]map[string][]Measurement `json:"values"`
}

// UserEventMetadataArgs describes the Args field of a "metadata" message to the user.
type UserEventMetadataArgs struct {

	// Devices maps device IDs to metadata
	Devices map[string]DeviceMetadata `json:"devices"`
}

// SensorMetadata contains different kinds of metadata on a sensor.
// All fields are optional.
type SensorMetadata struct {
	Name *string `json:"name,omitempty"`
	Unit *string `json:"unit,omitempty"`
	Port *int32  `json:"port,omitempty"`
}

// DeviceMetadata contains different kinds of metadata on a device.
// All fields are optional.
type DeviceMetadata struct {
	Name string `json:"name,omitempty"`

	// Sensors maps sensor IDs to Metadata
	Sensors map[string]SensorMetadata `json:"sensors,omitempty"`

	// DeletedSensors maps sensor IDs to an arbitrary string.
	DeletedSensors map[string]*string `json:"deletedSensors,omitempty"`
}

// Error describes the structure of an Error field in MessageIn and MessageOut.
// Description and Extra fields are optional.
type Error struct {
	Code        string      `json:"error"`
	Description string      `json:"description,omitempty"`
	Extra       interface{} `json:"extra,omitempty"`
}

func (e *Error) Error() string {
	result := e.Code
	if e.Description != "" {
		result = fmt.Sprintf("%v (%v)", result, e.Description)
	}
	if e.Extra != nil {
		result = fmt.Sprintf("%v [%v]", result, e.Extra)
	}
	return result
}

func badCommand(cmd string) *Error {
	return &Error{Code: "bad command", Extra: cmd}
}

func invalidInput(desc, extra string) *Error {
	return &Error{Code: "invalid input", Description: desc, Extra: extra}
}

func operationFailed(extra string) *Error {
	return &Error{Code: "operation failed", Extra: extra}
}

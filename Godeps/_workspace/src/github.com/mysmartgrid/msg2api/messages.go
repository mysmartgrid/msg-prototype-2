package msg2api

import (
	"encoding/json"
	"fmt"
)

type MessageIn struct {
	Command string          `json:"cmd"`
	Error   *Error          `json:"error"`
	Args    json.RawMessage `json:"args"`
}

type MessageOut struct {
	Command string      `json:"cmd,omitempty"`
	Now     *int64      `json:"now,omitempty"`
	Error   *Error      `json:"error,omitempty"`
	Args    interface{} `json:"args,omitempty"`
}

type DeviceCmdUpdateArgs struct {
	Values map[string][]Measurement `json:"values"`
}

type DeviceCmdAddSensorArgs struct {
	Name string `json:"name"`
	Unit string `json:"unit"`
	Port int32  `json:"port"`
}

type DeviceCmdRemoveSensorArgs DeviceCmdAddSensorArgs

type DeviceCmdUpdateMetadataArgs DeviceMetadata

type DeviceCmdRequestRealtimeUpdatesArgs []string

type UserCmdGetValuesArgs struct {
	SinceUnixMs    float64             `json:"since"`
	UntilUnixMs    float64             `json:"until"`
	TimeResolution string              `json:"resolution"`
	Sensors        map[string][]string `json:"sensors"`
}

type UserCmdRequestRealtimeUpdatesArgs map[string]map[string][]string //[Device][Resolution][Sensor]

type UserEventUpdateArgs struct {
	Resolution string                              `json:"resolution"`
	Values     map[string]map[string][]Measurement `json:"values"`
}

type UserEventMetadataArgs struct {
	Devices map[string]DeviceMetadata `json:"devices"`
}

type SensorMetadata struct {
	Name *string `json:"name,omitempty"`
	Unit *string `json:"unit,omitempty"`
	Port *int32  `json:"port,omitempty"`
}

type DeviceMetadata struct {
	Name           string                    `json:"name,omitempty"`
	Sensors        map[string]SensorMetadata `json:"sensors,omitempty"`
	DeletedSensors map[string]*string        `json:"deletedSensors,omitempty"`
}

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

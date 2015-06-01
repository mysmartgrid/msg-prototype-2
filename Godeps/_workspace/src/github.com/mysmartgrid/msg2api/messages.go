package msg2api

import (
	"encoding/json"
	"fmt"
)

type MessageIn struct {
	Command string          `json:"cmd"`
	Error   *Error        `json:"error"`
	Args    json.RawMessage `json:"args"`
}

type MessageOut struct {
	Command string      `json:"cmd,omitempty"`
	Now     *int64      `json:"now,omitempty"`
	Error   *Error    `json:"error,omitempty"`
	Args    interface{} `json:"args,omitempty"`
}

type DeviceCmdUpdateArgs struct {
	Values map[string][]Measurement `json:"values"`
}

type DeviceCmdAddSensorArgs struct {
	Name string `json:"name"`
}

type DeviceCmdRemoveSensorArgs DeviceCmdAddSensorArgs

type DeviceCmdUpdateMetadataArgs DeviceMetadata

type DeviceCmdRequestRealtimeUpdatesArgs []string

type UserCmdGetValuesArgs struct {
	SinceUnixMs  float64 `json:"since"`
	WithMetadata bool    `json:"withMetadata"`
}

type UserCmdRequestRealtimeUpdatesArgs map[string][]string

type UserEventUpdateArgs struct {
	Values map[string]map[string][]Measurement `json:"values"`
}

type UserEventMetadataArgs struct {
	Devices map[string]DeviceMetadata `json:"devices"`
}

type DeviceMetadata struct {
	Name           string             `json:"name,omitempty"`
	Sensors        map[string]string  `json:"sensors,omitempty"`
	DeletedSensors map[string]*string `json:"deletedSensors,omitempty"`
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

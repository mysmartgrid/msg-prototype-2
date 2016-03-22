package oldapi

import (
	"encoding/json"
	"log"
	"errors"
	"time"
)

var httpErrorMoved = errors.New("Moved Permanently")
var httpErrorMovedNo = 301

var httpErrorBadRequest = errors.New("Bad request")
var httpErrorBadRequestNo = 400

var httpErrorUnauthorized = errors.New("Unauthorized")
var httpErrorUnauthorizedNo = 401

var httpErrorForbidden = errors.New("Forbidden")
var httpErrorForbiddenNo = 402

var httpErrorNotFound = errors.New("Not Found")
var httpErrorNotFoundNo = 404

var httpErrorMethodNotAllowed = errors.New("Method Not Allowed")
var httpErrorMethodNotAllowedNo = 405

var httpErrorNotAcceptable = errors.New("Not acceptable")
var httpErrorNotAcceptableNo = 406

var httpErrorInvalidTimestamp = errors.New("Invalid Timestamp")
var httpErrorInvalidTimestampNo = 470

var httpErrorInvalidUnit = errors.New("Invalid Unit")
var httpErrorInvalidUnitNo = 471

var httpErrorInvalidMeasurement = errors.New("Invalid Measurement")
var httpErrorInvalidMeasurementNo = 472

var httpErrorInvalidObjectType = errors.New("Invalid Object Type")
var httpErrorInvalidObjectTypeNo = 473

var httpErrorInvalidObjectId = errors.New("Invalid Object Id")
var httpErrorInvalidObjectIdNo = 474

var httpErrorInvalidKey = errors.New("Invalid Key")
var httpErrorInvalidKeyNo = 475

var httpErrorInvalidTimePeriod = errors.New("Invalid Time Period")
var httpErrorInvalidTimePeriodNo = 476

var httpErrorInvalidEvent = errors.New("Invalid Event")
var httpErrorInvalidEventNo = 477

var httpErrorUnupgradableFirmware = errors.New("Unupgradable Firmware")
var httpErrorUnupgradableFirmwareNo = 478

var httpErrorInvalidSensorExternalId = errors.New("Invalid Sensor External Id")
var httpErrorInvalidSensorExternalIdNo = 479

var httpErrorInvalidCharacters = errors.New("Invalid Characters")
var httpErrorInvalidCharactersNo = 480

var httpErrorInvalidNetworkConfiguration = errors.New("Invalid Network Configuration")
var httpErrorInvalidNetworkConfigurationNo = 481

var httpErrorClientClosedRequest = errors.New("Client Closed Request")
var httpErrorClientClosedRequestNo = 499

var httpErrorInternalServerError = errors.New("Internal Server Error")
var httpErrorInternalServerErrorNo = 500

var httpErrorNotImplemented = errors.New("Not Implemented")
var httpErrorNotImplementedNo = 501

/*
var httpError = errors.New("")
var httpErrorNo = 47

*/

var badType = errors.New("bad type")

// Device Registration
type DeviceRegistration struct {
	Key	 string     `json:"key"`
	Type     string     `json:"type"`
	Firmware struct {
		Version     string
		ReleaseTime string
		Build       string
		Tag         string
	}
}


type DeviceRegistrationResponse struct {
	Upgrade       uint64
	Time          time.Time
}

// Heartbeat
type HeartbeatResponse struct {
	Upgrade    uint64
	Time       time.Time
}

// Device Event notification
type Event struct {
	Device    string
}

type EventResponse struct {
	Time      time.Time
}

// Sensor configuration
type SensorConfig struct {
	Device     string
	Externalid string    `json:"externalid,omitempty"`
	Function   string    `json:"function,omitempty"`
	Class      string
	Voltage    int       `json:"voltage,omitempty"`
	Current    int       `json:"current,omitempty"`
	Constant   int       `json:"constant,omitempty"`
	Description     string  `json:"description,omitempty"`
	Unit            string  `json:"unit,omitempty"`
	Port            int32   `json:"port,omitempty"`
	Enable          int
	Type            string  `json:"type,omitempty"`
}


type SensorMeasurements struct {
	Timestamp time.Time
	Value	  float64
}

type SensorPost struct {
	Config       SensorConfig         `json:"config,omitempty"`
	Measurements []SensorMeasurements `json:"measurements,omitempty"`
}


/*
// Heartbeat
type DeviceIfaceIPConfig struct {
	Protocol   string `json:"protocol,omitempty"`
	IP         string `json:"ip,omitempty"`
	Netmask    string `json:"netmask,omitempty"`
	Gateway    string `json:"gateway,omitempty"`
	Nameserver string `json:"nameserver,omitempty"`
}

type DeviceConfigNetLan struct {
	DeviceIfaceIPConfig

	Enabled bool `json:"enabled"`
}

type DeviceConfigNetWifi struct {
	DeviceIfaceIPConfig

	Enabled    bool   `json:"enabled"`
	SSID       string `json:"essid,omitempty"`
	Encryption string `json:"enc,omitempty"`
	PSK        string `json:"psk,omitempty"`
}

type DeviceConfigNetwork struct {
	LAN  *DeviceConfigNetLan  `json:"lan,omitempty"`
	Wifi *DeviceConfigNetWifi `json:"wifi,omitempty"`
}

type Heartbeat struct {
	Time	   time.Time
	Syslog	   string
	MemTotal   uint64
	MemBuffers uint64
	MemCached  uint64
	MemFree    uint64
	Uptime	   time.Time
	Resets	   uint64
	Firmware struct {
		Releasetime   string
		Build         string
		Version       string
		Tag           string
	}
	Config *DeviceConfigNetwork
//	Config struct {
//	}
	Type	   string
}


// Device Query 
type DeviceQueryResponse struct {
}

// Firmware upgrade
type FirmwareUpgradeResponse struct {
}


/*
{
"syslog":"H4sIAMtO3VYCA7WaS0/bUBCF9/0VV6hC7YL4vh9Zt4suUKvSroCFsW/AApzINqhU/fE1OKKURzzSzOyyiHKO555vnJz4sOyECEL6pXJLqcTq6uayXx8Y61UtRdWt20XTrtYPr+rjoOTpcnq9FD+PPn8X3Xo9iE1TCyVlNKK6rsWxOPgt9t5/2PTijzjv8kac7F3kX2Vd5uv1cXt6svdxT5yK/X1R5KEqmrYZFnXx+I5WdLkfym54d/joTUukN6XSg7fipu+Ks6Yd5UaFs1wOQpLqaMk5A/PM202fu8nb4/Uc35vQo8NvX49+iIth2PTLoig3zeL6rr8e33TeNfWizstorSnqfNtUuZg+ThoTqnpVl2XSQcqV1clWZX2W7VJoKSFecte9YuVLW627LleDqJvz8cJE04rhIv9753i5m6u7hfjU9FXZ1fdXv1m3fV481cTmUzm2fBp0bkKE5JNA5xkHq+oiV5ekElFyjtnCEIiOGYE3vbxA4MEKCQIGi4DWlutsLDY32oJWNF7HSc4ZwFa0dpY5nxa6oicrJPm06HxGO7OeCCQc1/E7bDSNVhAECHQ05wxgCBjtmRFwUAQmKyQIOGw+jWPKZ1hKdG62t/ad+STRUTvXAImE5hwz7FuKibwI7PDyAoFIhMCoqZBnY3X4//jzbW6H/qmEQ0skruNX2GhaZyGUEeg4zhnA7gLWcSOgoHeByQoJAuguxcY4swEJJNgQQNcoTjsIAgQ6nnMGsLuA04EZAQ29C0xWSBBA1zXOseUT3XG4CMongY6fWQMEEoFzzEAEYmRGAFzXTFZIEEDXNd4orrNB1yjeeQgCBDqBcwawfHrHnU8LzedkhSSf6C7FJzmznggk2BBA1yjh+e+U1xEg0ImcM4AhEHRiRsBBEZiskCCArmuCZ8pnxHccIQJWNIlO2LkGSCQi55iBCCTJikCE1zWTFQIEIr6uiUbvrGsivq6JxnIdP7pGiS5BKMPreMk5AxgC0StmBBQUgckKCQLoLiUmO7MBCSQc1/Gja5RkFAQBAh3NOQMYAskYZgTAdc1khQQBdF2TPFs+0R1HShKSTwIdNbMG7iU0TkJzjtnBEEjcCLzp5QUCiQwBbF2jpPFcZ4OtUcYf0qAVTaCjOWcA+lPpvjhjzif06ZqtFZJ8WnQ+k59ZTwQSgev4sTWKUsZAECDQsZwzgCGgjGNGAPp0zdYKCQLYukYpz5TPhO44lEp6Pp8kOmbnGiCRsJxjtjAEEi8CCVzXbK0QIJDQdY3SJu2saxK6rlHaKq7jx9YoSnsPoYxAJ3DOAHYX0D4yIwB9umZrhQQBbJeijJQzG5BAgg0BbI0yfg8KEAQIdCLnDGB3AWMSMwLQumZrhQQBjc5n0FxnY9C5SaB8EujEmTVAIJE4xwxDwErJjAD06ZqtFRIE0HXNeL1cZ4OuUayPEAQIdBLnDID5DNz5tOB8BrJ8orsUJ/XMeiKQYEMAXaM4kyAI4HUs8f+qfwFzxLyHUz8AAA==",
"time":1457344203,


"memtotal":30117888,
"membuffers":2453504,
"memcached":0,
"memfree":6598656,
"uptime":518125,
"reset":0
"firmware":{
	"releasetime":"20160127_1426",
	"build":"1f2eabedc4eddd6a4577f4811a9bd44f9bdf202c",
	"version":"2.3.0-4",
	"tag":"flukso-2.3.0-rc4"},
"config":{
	"network":{
		"wifi":{
			"enabled":1,
			"enc":"wpa2",
			"essid":"flukso_FRITZ!Box",
			"protocol":"dhcp",
			"psk":"9259109434019904"},
		"lan":{"enabled":0}
	}
},
"type":"amperix1",
}
* /



func (hb *Heartbeat) UnmarshalJSON(raw []byte) error {
	var data hbData
	if err := json.Unmarshal(raw, &data); err != nil {
		return err
	}
	*hb = Heartbeat{
		Time:     time.Unix(data.Time, 0),
		Memory:   data.Memory,
		Uptime:   time.Duration(data.Uptime) * time.Second,
		Resets:   data.Resets,
		Type:     data.Type,
		Syslog:   data.Syslog,
		Firmware: data.Firmware,
		Config:   data.Config,
	}
	return nil
}
*/





// request Data
type evData struct {
	Device string
}


// Response data
type hbrData struct {
	Upgrade    uint64
	Timestamp  int64
}

type drmrData struct {
	Response   string
}

type drrData struct {
	Upgrade    uint64
	Timestamp  int64
}

type scData struct {
	Device       string
	Externalid   string
	Function     string
	Class        string
	Voltage      int
	Current      int
	Constant     int
	Description  string
	Unit            string
	Port            int32
	Enable          int
	Type            string
}

type smData struct {
	item [][]int `json:"measurements,omitempty"`
}

type sensorData struct {
	Config       scData     `json:"config,omitempty"`
	Measurements [][]int    `json:"measurements,omitempty"`
}


// Device Query Response
type dqrData struct {
	Description    string
	Type           string
/*  sensors: [
    { meter:    <string(32) - sensor id>,
      function: <string(16) - sensor name> },
    ...
    { meter:    <string(32) - sensor id>,
      function: <string(16) - sensor name> }
  ]
}
*/
}
type furData struct {
	Data     string `json:"data,omitempty"`
}

type evrData struct {
	Timestamp   int64
}

// query requests data
type drData struct {
	Key	 string     `json:"key" binding:"required"`
	Type     string     `json:"type"`
	Firmware struct {
		Version     string
		ReleaseTime string
		Build       string
		Tag         string
	}
}

func (dr *DeviceRegistration) UnmarshalJSON(raw []byte) error {
	var data drData
	if err := json.Unmarshal(raw, &data); err != nil {
		log.Print("Unmarshal failed.")
		return err
	}
	if data.Key != "" {
		log.Print("Unmarshal succeeded with key: '", data.Key, "' .")

		*dr = DeviceRegistration{
			Key:      data.Key,
			Type:     data.Type,
			Firmware: data.Firmware,
		}
	} else { return badType }

	return nil
}


func (drr DeviceRegistrationResponse) MarshalJSON() ([]byte, error) {
	data := drrData{
		Upgrade:    drr.Upgrade,
		Timestamp:  drr.Time.Unix(),
	}
	return json.Marshal(data)
}

// Heartbeat
func (hbr HeartbeatResponse) MarshalJSON() ([]byte, error) {
	data := hbrData{
		Upgrade:    hbr.Upgrade,
		Timestamp:  hbr.Time.Unix(),
	}
	return json.Marshal(data)
}


// Event notification
func (ev *Event) UnmarshalJSON(raw []byte) error {
	var data evData
	if err := json.Unmarshal(raw, &data); err != nil {
		log.Print("Unmarshal failed.")
		return err
	}
	*ev = Event{
		Device:	data.Device,
	}
	return nil
}

func (evr EventResponse) MarshalJSON() ([]byte, error) {
	data := evrData{
		Timestamp:	evr.Time.Unix(),
	}
	return json.Marshal(data)
}

// Sensor Config
func (sp *SensorPost) UnmarshalJSON(raw []byte) error {
	var data sensorData
	if err := json.Unmarshal(raw, &data); err != nil {
		log.Print("Unmarshal failed.", err.Error())
		return err
	}
	if ( data.Config.Device != "" ) {
		log.Print("   Have SensorConfig data.")
		sc := SensorConfig{
			Device:	    data.Config.Device,
			Externalid: data.Config.Externalid,
			Function:    data.Config.Function,
			Class:       data.Config.Class,
			Voltage:     data.Config.Voltage,
			Current:     data.Config.Current,
			Constant:    data.Config.Constant,
			Description: data.Config.Description,
			Unit:        data.Config.Unit,
			Port:        data.Config.Port,
			Enable:      data.Config.Enable,
			Type:        data.Config.Type,
		}
		*sp = SensorPost{Config: sc}
	}
	if ( data.Measurements != nil ) {
		log.Print("   Have SensorMeasurements data - ", len(data.Measurements))
		log.Print("   Have SensorMeasurements ", data.Measurements[0][0], " - ", data.Measurements[0][1])
		l := len(data.Measurements)
		var sm []SensorMeasurements
		for i:=0; i<l; i++ {
			sm = append(sm, SensorMeasurements{Timestamp: time.Unix(int64(data.Measurements[i][0]), 0), Value: float64(data.Measurements[i][1])})
		}
		*sp = SensorPost{Measurements: sm}
	}
	return nil
}

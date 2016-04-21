package oldapi

import (
//        "crypto/aes"
//        "crypto/cipher"
        "crypto/hmac"
//        "crypto/rand"
        "crypto/sha1"
        "encoding/hex"
        "encoding/json"
        "errors"
        "github.com/gorilla/mux"
        "io/ioutil"
//        "math"
	"log"
        "net/http"
        "strconv"
        "time"
	"github.com/mysmartgrid/msg-prototype-2/regdev"
	"github.com/mysmartgrid/msg-prototype-2/db"
	"github.com/mysmartgrid/msg-prototype-2/oldapidb"
)

type OldApiServer struct {
	Db regdev.Db
	Udb db.Db
	Sdb oldapidb.Db
}

/*
  Queries implemented:
POST /device/<device_id>    Device Registration            implemented
DELETE /device/<device_id>  Device Removal                 not yet
POST /device/<device_id>    Device Heartbeat               implemented/without remoteSupport
GET  /device/<device_id>    Device Query                   not yet

GET  /firmware/<device_id>  Firmware Upgrade File Download not yet

POST /event/<event_id>      Device Event Notification      partial

POST /sensor/<sensor_id>    Sensor Configuration             implemented
GET  /sensor/<sensor_id>    Sensor Configuration Query       not yet
POST /sensor/<sensor_id>    Sensor Measurements Registration implemented
DELETE /sensor/<sensor_id>  Sensor Removal                   not yet
GET  /sensor/<sensor_id>?<attrib> Sensor Measurements Query  not yet


*/
var (
	InvalidAuth     = errors.New("invalid authorisation")
)

// checkSignature
func (s *OldApiServer) CheckSignature(r *http.Request, body []byte, device string, key []byte) (error,int) {
	// check signature
	log.Print("   CheckSignature for device '", device, "' !")
	digests, hasKeys := r.Header["X-Digest"]
	log.Print("Check Request for device: %", device)
	if !hasKeys {
		//http.Error(w, "digest missing", 401)
		return httpErrorUnauthorized,httpErrorUnauthorizedNo
	}
	if len(digests) != 1 {
		//http.Error(w, "multiple X-Digest headers", 402)
		return httpErrorUnauthorized,httpErrorUnauthorizedNo
	}
	sig, err := hex.DecodeString(digests[0])
		if err != nil {
		return httpErrorUnauthorized,httpErrorUnauthorizedNo
		//http.Error(w, err.Error(), 403)
	}
		mac := hmac.New(sha1.New,  key)
	mac.Write(body)
	if !hmac.Equal(mac.Sum(nil), sig) {
		return httpErrorUnauthorized,httpErrorUnauthorizedNo
		//http.Error(w, "bad request", 404)
	}
	mac.Reset()
	return nil,200
}

// decodeRequest
func (s *OldApiServer) CheckRequest(w http.ResponseWriter, r *http.Request, body []byte, device string) (error) {
	// fetch device from database
	log.Print("   CheckRequest for device '", device, "' !")
	s.Db.View(func(tx regdev.Tx) error {
		dev := tx.Device(device)
		if dev == nil {
			log.Print("Device '", device, "' not found: ")
			http.Error(w, httpErrorNotFound.Error(), httpErrorNotFoundNo)
			return InvalidAuth
		}

		log.Print("Device '", device, "' found!")
		// check signature
		if err, errno := s.CheckSignature(r, body, device, dev.Key()); err != nil {
			log.Print("Check failed")
			http.Error(w, err.Error(), errno)
			return InvalidAuth
		}
		return nil
	})
	return nil
}

/*
func (s *OldApiServer) handleDeviceRegistration(dr DeviceRegistration) (error, int){
	s.Db.View(func(tx regdev.Tx) error {
		dev := tx.Device(device)
		if dev == nil {
			log.Print("Device '", device, "' not found: ")
			if err, errno := s.CheckSignature(r, body, device, dr.Key); err != nil {
				log.Print("Check failed")
				http.Error(w, err.Error(), errno)
				return InvalidAuth
			}
			// register device now
			s.Db.Update(func(tx regdev.Tx) error {
				err := tx.AddDevice(device, []byte(key))
			})
		} else {
		}

	})
}
*/

// device query
func (s *OldApiServer) Device_Get(w http.ResponseWriter, r *http.Request) {
	log.Print("Got Device-GET...")
	// errors: httpErrorNotFound

	device := mux.Vars(r)["device"]
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// check first for X-Token
	accessToken, hasKeys := r.Header["X-Token"]
	if !hasKeys {
		if err := s.CheckRequest(w, r, body, device); err != nil {
			log.Print("Check Request failed...")
			return
		}
	}
        log.Print("Check token: ", accessToken)
	log.Print(string(body))
}

// device registration, heartbeat
func (s *OldApiServer) Device_Post(w http.ResponseWriter, r *http.Request) {
	log.Print("Got Device-POST...")
	device := mux.Vars(r)["device"]
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	log.Print(string(body))

	// test for registration
	var reg DeviceRegistration
	if err := json.Unmarshal(body, &reg); err != nil {
		if( err != badType ) {
			http.Error(w, err.Error(), 400)
			return
		}

		if err := s.CheckRequest(w, r, body, device); err != nil {
			log.Print("Check Request failed...")
			return
		}

		log.Print("==> got a device post, but not a registration.")
	
		// test for heartbeat
		var hb regdev.Heartbeat
		if err := json.Unmarshal(body, &hb); err != nil {
			http.Error(w, err.Error(), 400)
			log.Print("==> got a device post, but not a heartbeat.")
			return
		}
		log.Print("got an heartbeat.")
		s.Db.View(func(tx regdev.Tx) error {
			dev := tx.Device(device)
			if dev == nil {
				log.Print("Device '", device, "' not found: ")
				http.Error(w, httpErrorNotFound.Error(), httpErrorNotFoundNo)
				return nil
			}
			username,err := dev.UserLink()
			if (err) {
				log.Print("    Device linkto '", username, "' ")
				resp := HeartbeatResponse{
					Upgrade:  0,
					Time:     time.Now(),
				}
				data, err := json.Marshal(resp)
				if err != nil {
					http.Error(w, err.Error(), 500)
					return nil
				}
				w.Write(data)
			} else {
				http.Error(w, httpErrorNotFound.Error(), httpErrorNotFoundNo)
			}
			return nil		
		})
	} else {
		log.Print("got an registration request.")
		// register device now
		s.Db.Update(func(tx regdev.Tx) error {
			dev := tx.Device(device)
			if dev == nil {
				log.Print("Device '", device, "' not found: ")
				if err, errno := s.CheckSignature(r, body, device, []byte(reg.Key)); err != nil {
					log.Print("Check failed")
					http.Error(w, err.Error(), errno)
					return err
				}
				log.Print("Insert device into database")
				log.Print("	Insert device into database - 1")
				err := tx.AddDevice(device, []byte(reg.Key))
				if err != nil {
					log.Print("	Insert device into database - failed")
					http.Error(w, httpErrorBadRequest.Error(),  httpErrorBadRequestNo)
					return httpErrorBadRequest
				}
				log.Print("	Insert device into database - success")
				resp := DeviceRegistrationResponse{
					Upgrade:  0,
					Time:     time.Now(),
				}
				log.Print("	Insert device into database - success2")
				data, err := json.Marshal(resp)
				if err != nil {
					http.Error(w, err.Error(), 500)
					return err
				}
				log.Print("	Insert device into database - success3")
				w.Write(data)
				log.Print("	Insert device into database - success4")
			} else {
				log.Print("Device '", device, "' found: ")
			}
			return nil	
		})
		log.Print("  Device view finished '", device, "' found: ")
	}
	log.Print("Device POST finished")
}

// device deletion
func (s *OldApiServer) Device_Delete(w http.ResponseWriter, r *http.Request) {
	log.Print("Got Device-DELETE...")
	device := mux.Vars(r)["device"]
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err := s.CheckRequest(w, r, body, device); err != nil {
		log.Print("Check Request failed...")
		return
	}

	log.Print(string(body))
}

// Sensor Configuration Query
// Sensor Measurements Query
func (s *OldApiServer) Sensor_Get(w http.ResponseWriter, r *http.Request) {
	log.Print("Got Sensor-GET...")
	// possible errors: httpErrorBadRequest, httpErrorForbidden, httpErrorInvalidTimestamp
	// httpErrorInvalidUnit, 

	//sensor := mux.Vars(r)["sensor"]
	device := ""
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err := s.CheckRequest(w, r, body, device); err != nil {
		log.Print("Check Request failed...")
		return
	}

	log.Print(string(body))
}

// Sensor Configuration
// Sensor Measurements Registration
func (s *OldApiServer) Sensor_Post(w http.ResponseWriter, r *http.Request) {
	// can be sensorconfigration or sensor measurements
	sensorId := mux.Vars(r)["sensor"]

	log.Print("Got Sensor-POST...%", sensorId)
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	log.Print(string(body))
	var sc SensorPost
	if err := json.Unmarshal(body, &sc); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var device string
	if (sc.Config.Device != "") {
		device = sc.Config.Device		
	} else {
		// these are measurements values
		s.Udb.View(func(tx db.Tx) error {
			//i := 0
			sensor := tx.Sensor(sensorId)
			if sensor != nil {
				log.Print(" got the sensorreturn...")
				// get timestamp an value

				//
				l := len(sc.Measurements)
				for i:=1; i<l; i++ {
					deltaV := sc.Measurements[i].Value - sc.Measurements[i-1].Value
					deltaT := sc.Measurements[i].Timestamp.Unix() - sc.Measurements[i-1].Timestamp.Unix()
					value := ( 3600 * deltaV) / (float64)(deltaT)
					s.Udb.AddReading(sensor, sc.Measurements[i].Timestamp,value)
					log.Printf("    Timestamp: %d  Value: %f, %f ", sc.Measurements[i].Timestamp.Unix(), sc.Measurements[i].Value, value)
				}
				
				log.Print("  added successfully the readings: ", sensor.Name())
			} else {
				log.Print("  sensor was not found...: ", sensorId)
				http.Error(w, httpErrorNotFound.Error(), httpErrorNotFoundNo)
			}
			return nil
		})
		return		
	}

	if err := s.CheckRequest(w, r, body, device); err != nil {
		log.Print("Check Request failed...")
		return
	}

	if( device != "" ) {
		// add the sensor config
		//{"config":{"type":"electricity","constant":1000,
	        //            "device":"3461d00337cdfdaa92700f4294cadbe4","class":"pulse",
        	//            "port":5,"enable":0}}
		s.Db.View(func(tx regdev.Tx) error {
			log.Print("    Search for device '", sc.Config.Device, "'")
			dev := tx.Device(sc.Config.Device)
			if dev == nil {
				log.Print("Device '", sc.Config.Device, "' not found: ")
				http.Error(w, httpErrorNotFound.Error(), httpErrorNotFoundNo)
				return nil
			}
			s.Udb.Update(func(tx db.Tx) error {
				username,err := dev.UserLink()
				log.Print("    Search for User '", username, "'")
				if( err) {
					log.Print("    Search for User '", username, "'  - found")
					user := tx.User(username)
					if user == nil {
						http.Error(w, httpErrorNotFound.Error(), httpErrorNotFoundNo)
						return nil
					}
					udev := user.Device(dev.ID())
					s, err := udev.AddSensor(sensorId, sc.Config.Unit, sc.Config.Port, 1.0)
					if err != nil  && s==nil{
						log.Print("    Failed to add sensor!", err.Error())
						http.Error(w, httpErrorInternalServerError.Error(), httpErrorInternalServerErrorNo)
						}
				} else {
					log.Print("    Search for User '", username, "'  - not found")
					http.Error(w, httpErrorNotFound.Error(), httpErrorNotFoundNo)
					return nil
				}
				return nil
			})
			return nil
		})
	}

}

// Sensor Removal
func (s *OldApiServer) Sensor_Delete(w http.ResponseWriter, r *http.Request) {
	//sensor := mux.Vars(r)["sensor"]

	log.Print("Got Sensor-DELETE...")
	device := ""
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err := s.CheckRequest(w, r, body, device); err != nil {
		log.Print("Check Request failed...")
		return
	}

	log.Print(string(body))
}

// Device Event Notification
func (s *OldApiServer) Event_Post(w http.ResponseWriter, r *http.Request) {
	eventstr := mux.Vars(r)["eventid"]
	eventid,err := strconv.Atoi(eventstr)

	log.Print("Got Event-POST...")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	var ev Event
	if err := json.Unmarshal(body, &ev); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err := s.CheckRequest(w, r, body, ev.Device); err != nil {
		log.Print("Check Request failed...")
		return
	}

	if ( (eventid<104) || (eventid>108) ) {
		http.Error(w, httpErrorInvalidEvent.Error(), httpErrorInvalidEventNo)
		return
	}

	log.Print(string(body))
	
}

// Firmware Upgrade File Download
func (s *OldApiServer) Firmware_Get(w http.ResponseWriter, r *http.Request) {
	device := mux.Vars(r)["device"]
	// possible errors: httpErrorForbidden

	log.Print("Got Firmware-GET...")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err := s.CheckRequest(w, r, body, device); err != nil {
		log.Print("Check Request failed...")
		return
	}
}

func (s *OldApiServer) RegisterRoutes(r *mux.Router) {

	r.HandleFunc("/device/{device}", s.Device_Get).Methods("GET")
	r.HandleFunc("/device/{device}", s.Device_Post).Methods("POST")
	r.HandleFunc("/device/{device}", s.Device_Delete).Methods("DELETE")
	r.HandleFunc("/sensor/{sensor}", s.Sensor_Get).Methods("GET")
	r.HandleFunc("/sensor/{sensor}", s.Sensor_Post).Methods("POST")
	r.HandleFunc("/sensor/{sensor}", s.Sensor_Delete).Methods("DELETE")
	r.HandleFunc("/event/{eventid}", s.Event_Post).Methods("POST")
	r.HandleFunc("/firmware/{device}", s.Firmware_Get).Methods("GET")
}

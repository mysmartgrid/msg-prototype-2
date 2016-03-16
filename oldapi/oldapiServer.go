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
)

type OldApiServer struct {
	Db regdev.Db
}

var (
	InvalidAuth     = errors.New("invalid authorisation")
)

// checkSignature
func (s *OldApiServer) CheckSignature(r *http.Request, body []byte, device string, key []byte) (error,int) {
	// check signature
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

	if err := s.CheckRequest(w, r, body, device); err != nil {
		log.Print("Check Request failed...")
		return
	}

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
		resp := HeartbeatResponse{
			Upgrade:  0,
			Time:     time.Now(),
		}
		data, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write(data)
	} else {
		log.Print("got an registration request.")
		s.Db.View(func(tx regdev.Tx) error {
			dev := tx.Device(device)
			if dev == nil {
				log.Print("Device '", device, "' not found: ")
				if err, errno := s.CheckSignature(r, body, device, []byte(reg.Key)); err != nil {
					log.Print("Check failed")
					http.Error(w, err.Error(), errno)
					return err
				}
				log.Print("Insert device into database")
				// register device now
				s.Db.Update(func(tx regdev.Tx) error {
					log.Print("	Insert device into database - 1")
					err := tx.AddDevice(device, []byte(reg.Key))
					if err != nil {
						log.Print("	Insert device into database - failed")
						http.Error(w, httpErrorBadRequest.Error(),  httpErrorBadRequestNo)
						return httpErrorBadRequest
					}
					log.Print("	Insert device into database - success")
					resp := DeviceRegistrationResponse{
						Upgrade:  1,
						Time:     time.Now(),
					}
					data, err := json.Marshal(resp)
					if err != nil {
						http.Error(w, err.Error(), 500)
						return err
					}
					w.Write(data)
					return nil
				})
			} else {
			}
			return nil

		})

	}
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
	sensor := mux.Vars(r)["sensor"]

	log.Print("Got Sensor-POST...%", sensor)
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var sc SensorConfig
	if err := json.Unmarshal(body, &sc); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err := s.CheckRequest(w, r, body, sc.Device); err != nil {
		log.Print("Check Request failed...")
		return
	}

	log.Print(string(body))
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

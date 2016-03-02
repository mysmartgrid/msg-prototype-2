package oldapi

type OldApiServer struct {
	Db Db
}

// device query
func (s *OldApiServer) Device_Get(w http.ResponseWriter, r *http.Request) {
}

// device registration, heartbeat
func (s *OldApiServer) Device_Post(w http.ResponseWriter, r *http.Request) {
}

// device deletion
func (s *OldApiServer) Device_Delete(w http.ResponseWriter, r *http.Request) {
}

// Sensor Configuration Query
// Sensor Measurements Query
func (s *OldApiServer) Sensor_Get(w http.ResponseWriter, r *http.Request) {
}

// Sensor Configuration
// Sensor Measurements Registration
func (s *OldApiServer) Sensor_Post(w http.ResponseWriter, r *http.Request) {
}

// Sensor Removal
func (s *OldApiServer) Sensor_Delete(w http.ResponseWriter, r *http.Request) {
}

// Device Event Notification
func (s *OldApiServer) Event_Post(w http.ResponseWriter, r *http.Request) {
}

// Firmware Upgrade File Download
func (s *OldApiServer) Firmware_Get(w http.ResponseWriter, r *http.Request) {
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

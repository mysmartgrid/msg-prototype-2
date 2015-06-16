package regdev

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
)

type DeviceServer struct {
	Db Db

	router *mux.Router
}

func (s *DeviceServer) registerDevice(w http.ResponseWriter, r *http.Request) {
	keys, hasKeys := r.Header["X-Key"]
	if !hasKeys {
		http.Error(w, "key missing", 400)
		return
	}
	if len(keys) != 1 {
		http.Error(w, "multiple X-Key headers", 400)
		return
	}
	key, err := hex.DecodeString(keys[0])
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	err = s.Db.Update(func(tx Tx) error {
		return tx.AddDevice(mux.Vars(r)["device"], key)
	})
	if err != nil {
		http.Error(w, err.Error(), 400)
	}
}

func (s *DeviceServer) getDeviceInfo(w http.ResponseWriter, r *http.Request) {
	s.Db.View(func(tx Tx) error {
		dev := tx.Device(mux.Vars(r)["device"])
		if dev == nil {
			http.Error(w, "not found", 404)
			return nil
		}

		user, _ := dev.UserLink()
		net := dev.GetNetworkConfig()
		resp := DeviceConfiguration{user, &net}
		data, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return nil
		}

		mac := hmac.New(sha256.New, dev.Key())

		var iv [16]byte
		if _, err := rand.Read(iv[:]); err != nil {
			http.Error(w, err.Error(), 500)
			return nil
		}

		var nonce [16]byte
		if _, err := rand.Read(nonce[:]); err != nil {
			http.Error(w, err.Error(), 500)
			return nil
		}

		mac.Write(nonce[:])
		key := mac.Sum(nil)[:16]
		mac.Reset()

		cinst, _ := aes.NewCipher(key)
		transform := cipher.NewCFBEncrypter(cinst, iv[:])

		transform.XORKeyStream(data, data)
		mac.Write(data)

		w.Header()["X-Nonce"] = []string{hex.EncodeToString(nonce[:])}
		w.Header()["X-IV"] = []string{hex.EncodeToString(iv[:])}
		w.Header()["X-HMAC"] = []string{hex.EncodeToString(mac.Sum(nil))}

		w.Write([]byte(hex.EncodeToString(data[:])))

		return nil
	})
}

func (s *DeviceServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.router == nil {
		s.router = mux.NewRouter()

		s.router.HandleFunc("/regdev/v1/{device}", s.registerDevice).Methods("POST")
		s.router.HandleFunc("/regdev/v1/{device}", s.getDeviceInfo).Methods("GET")
	}

	s.router.ServeHTTP(w, r)
}

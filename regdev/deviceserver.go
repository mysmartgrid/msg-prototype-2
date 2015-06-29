package regdev

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"time"
)

type DeviceServer struct {
	Db Db
}

var badHeartbeat = errors.New("invalid heartbeat")
var badArgs = errors.New("bad arguments")

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

func parseHeartbeatParams(r *http.Request) (ts time.Time, tsRaw []byte, sig []byte, err error) {
	args := r.URL.Query()
	if len(args["ts"]) != 1 || len(args["sig"]) != 1 {
		err = badArgs
		return
	}

	tsArg, err := strconv.ParseInt(args["ts"][0], 10, 64)
	if err != nil {
		return
	}
	ts = time.Unix(tsArg, 0)
	tsRaw = []byte(args["ts"][0])

	sig, err = hex.DecodeString(args["sig"][0])
	return
}

func (s *DeviceServer) heartbeat(w http.ResponseWriter, r *http.Request) {
	s.Db.Update(func(tx Tx) error {
		dev := tx.Device(mux.Vars(r)["device"])
		if dev == nil {
			http.Error(w, "not found", 404)
			return badHeartbeat
		}

		mac := hmac.New(sha256.New, dev.Key())

		ts, tsRaw, sig, err := parseHeartbeatParams(r)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return err
		}
		if math.Abs(ts.Sub(time.Now()).Hours()) > 4 {
			http.Error(w, "bad timestamp", 400)
			return err
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return err
		}

		mac.Write(tsRaw)
		mac.Write(body)
		if !hmac.Equal(mac.Sum(nil), sig) {
			http.Error(w, "bad request", 400)
			return badHeartbeat
		}
		mac.Reset()

		var hb Heartbeat
		if err := json.Unmarshal(body, &hb); err != nil {
			http.Error(w, err.Error(), 400)
			return err
		}

		hb.Time = ts

		if err := dev.RegisterHeartbeat(hb); err != nil {
			http.Error(w, err.Error(), 500)
			return err
		}

		user, _ := dev.UserLink()
		net := dev.GetNetworkConfig()
		resp := DeviceConfiguration{user, &net}
		data, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return err
		}

		var iv [16]byte
		if _, err := rand.Read(iv[:]); err != nil {
			http.Error(w, err.Error(), 500)
			return err
		}

		var nonce [16]byte
		if _, err := rand.Read(nonce[:]); err != nil {
			http.Error(w, err.Error(), 500)
			return err
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

func (s *DeviceServer) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/v1/{device}", s.registerDevice).Methods("POST")
	r.HandleFunc("/v1/{device}/status", s.heartbeat).Methods("POST")
}

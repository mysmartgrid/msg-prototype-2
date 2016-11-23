package main

import (
       "fmt"
        "bytes"
	"regexp"
	"bufio"
        "crypto/aes"
        "crypto/cipher"
        "crypto/hmac"
        "crypto/sha256"
        "crypto/tls"
        "os"
        "encoding/hex"
        "encoding/json"
        "errors"
        msgp "github.com/mysmartgrid/msg2api"
        uci  "github.com/mysmartgrid/msg-prototype-2/uciconf"
        "io/ioutil"
        "log"
        "log/syslog"
        "net/http"
        "net/url"
        "strconv"
	"time"
	"sync"
	"strings"
)

/* 
1. Size readbuffer:
   format [60* '[1234567890,x],']
   60 * 14 + 2 = 842
   with size(x) = 5 requires 60 * 5 = 300 bytes
   sum: 1142 bytes required for read-buffer

2. Restart this daemon, every time when the fluksod is restarted. This
   avoids to check the recreations of the fifo files.q
*/
   

var DIR = "/var/run/fluksod/fifo/"

var useSSL bool
var tlsConfig tls.Config
var dev *device
var debug = 0

type sensor struct {
        Name                string
        Unit                string
        Port                int32
        Factor              float64
        LastRealtimeRequest time.Time
	Id                  string
	Enabled             bool
	FiFoFile            *os.File
}

type device struct {
        ID  string
        Key []byte

        User string

	mutex * sync.Mutex

        Sensors map[string]sensor

        api    string
        client *msgp.DeviceClient

        regdevAPI string
}

var errDeviceNotRegistered = errors.New("device not registered")

type AmperixValue struct {
     Timestamp int
     Value     int
}

type AmperixValues struct {
     Measurements []AmperixValue
}

type avData struct {
     Measurements []AmperixValue `json:"m"`
}

func (av *AmperixValue) UnmarshalJSON(raw []byte) error {
     if debug>1 {
	log.Printf("avPair.UnmarshalJSON()..'%s'\n", raw)
     }
     var d []int
     if err := json.Unmarshal(raw, &d); err != nil {
             return err
     }
     //log.Printf("==%d, %d === \n", d[0], d[1])
     *av = AmperixValue{
		Timestamp : d[0],
		Value     : d[1],
     }
     return nil
}

func (av *AmperixValues) UnmarshalJSON(raw []byte) error {
     var data avData
     if err := json.Unmarshal(raw, &data); err != nil {
//             return err
     }
     if debug>1 {
	log.Printf("AmperixValues.UnmarshalJSON()..\n")
	log.Printf("--Data:'%s', %d", raw, len(data.Measurements))
     }
     n := len(data.Measurements)
     var mx = 0
     for i:=0; i<n; i++ { 
	 if(data.Measurements[i].Timestamp > 0) { mx++ }
     }
     *av = AmperixValues{
          Measurements : data.Measurements[:mx],
     }

     if debug>1 {
          log.Printf("AmperixValues.UnmarshalJSON() - Done..\n")
     }
     return nil
}

func (av *AmperixValues)convValues() []msgp.Measurement{
     n := len(av.Measurements)
     slice := make([]msgp.Measurement, n)
     if debug>1 {
	log.Printf("Convert: %d\n", n)
     }
     for i:=0; i<n; i++ {
	 if(int64(av.Measurements[i].Timestamp)>0) {
	 slice[i] = msgp.Measurement{
		  Time:  time.Unix(int64(av.Measurements[i].Timestamp), 0),
		  Value: float64(av.Measurements[i].Value),
		  }
	 }
     }
     return slice
}

func testConvert(raw []byte) {
     var ap AmperixValues
     err := json.Unmarshal(raw, &ap)
     if err != nil {
	log.Printf("%#v%s\n", err, err.Error())
     }
     if debug>0 {
          log.Printf("===> %d ===\n", len(ap.Measurements))
     }
     n := len(ap.Measurements)
     for i:=0; i<n; i++ {
	 log.Printf("%02d - %d , %d\n", i, ap.Measurements[i].Timestamp,
			  ap.Measurements[i].Value)
     }     
}


func (dev *device) readConfig() error {
     dev.Sensors = make(map[string]sensor, 6) // Read 4MB at a time
     for i := 1 ; i<6 ; i++  {
	 option := fmt.Sprintf("sensor.%d", i)
	 id     := uci.Get("flukso", option, "id")
	 enable := uci.Get("flukso", option, "enable")
	 name   := uci.Get("flukso", option, "function")
	 enabled := false
	 if enable != "0" {
	    enabled = true
	 }
	 data := sensor{ Name: name, Unit :"W",
			 Port: int32(i), Factor: 1,
			 Id: id, Enabled: enabled }
	 dev.Sensors[id] = data
     }

     if debug>0 {
         log.Printf("readConfig done\n")
     }
     return nil
}

func (dev *device) Run() {
    log.Printf("Run...\n")
    c := len(dev.Sensors)

    dev.mutex = &sync.Mutex{}
    var wg sync.WaitGroup
    wg.Add(c)

    for sensor, data := range dev.Sensors {
	 if data.Enabled != false {
	    go dev.HandleSensor(sensor)
	 }
    }
    wg.Wait()
}

func Readln(r *bufio.Reader) (string, error) {
  var (isPrefix bool = true
       err error = nil
       line, ln []byte
      )
  for isPrefix && err == nil {
      line, isPrefix, err = r.ReadLine()
      ln = append(ln, line...)
  }
  return string(ln),err
}

func (dev *device) HandleSensor(sensorId string) error {

     //     defer dev.wg.Done()

     log.Printf("Sensor: %s\n", sensorId)
     file, err := os.Open(DIR + sensorId)
     log.Printf("Sensor: %s - done\n", sensorId)
     if err != nil {
	return err
     }
     
     values := make(map[string][]msgp.Measurement, 1)

     r := bufio.NewReader(file)

     for {
	 data, err := Readln(r)
	 if err != nil {
	    if err.Error() != "EOF" {
	       log.Printf("%#v%s\n", err, err.Error())
	    }
   	  } else { 
	      n := len(data)
	      if n > 0 {
	         raw := make([]byte, n)
		 copy(raw[0:n], data[:])
		 p := fmt.Sprintf("{ \"m\" : %s }", raw)
		 if debug>1 {
		    log.Printf("D:%s - '%s'\n", sensorId, p)
		 }
		 raw  = []byte(p)
		 var ap AmperixValues
		 err = json.Unmarshal(raw, &ap)
   	         if err != nil {
	          log.Printf("%#v%s\n", err, err.Error())
		 }
		 values[sensorId] = ap.convValues()
		 dev.mutex.Lock()
		 if debug>0 {
		    log.Printf("== %s => %d ===\n", sensorId, len(ap.Measurements))
                 }
		 if err := dev.getClient().Update(values); err != nil {
	            log.Println("Sensor err. (%d)", len(values[sensorId]))
		 }
		 dev.mutex.Unlock()
	  } else {
		 time.Sleep(1000 * time.Millisecond)
	  }}
     }
     file.Close()
     return nil
}


func (dev *device) getClient() *msgp.DeviceClient {
        if dev.client == nil {
                client, err := msgp.NewDeviceClient(dev.api+"/"+dev.User+"/"+dev.ID, dev.Key, &tlsConfig)
                if err != nil {
                        log.Fatalf("Device::client: %v", err.Error())
                }

                client.RequestRealtimeUpdates = func(sensors []string) {
			//if debug>1 {
			log.Printf("server requested realtime updates for %v", sensors)
                        //}
			for _, sensor := range sensors {
                                if s, ok := dev.Sensors[sensor]; ok {
                                        s.LastRealtimeRequest = time.Now()
                                }
                        }
                }

                dev.client = client
        }

        return dev.client
}

func getMemInfo() map[string]uint64 {
	data, err := ioutil.ReadFile("/proc/meminfo")
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(data), "\n")
	result := make(map[string]uint64)
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Split(line, ":")
		key := fields[0]
		value, err := strconv.ParseUint(strings.Fields(fields[1])[0], 10, 64)
		if err != nil {
			panic(err)
		}

		switch key {
		case "MemTotal":
			result["Total"] = value
		case "MemFree":
			result["Free"] = value
		case "Cached":
			result["Cached"] = value
		case "Buffers":
			result["Buffered"] = value
		}
	}

	return result
}

func getUptime() uint64 {
	data, err := ioutil.ReadFile("/proc/uptime")
	if err != nil {
		panic(err.Error())
	}
	uptime, err := strconv.ParseFloat(strings.Fields(string(data))[0], 64)
	if err != nil {
		panic(err)
	}
	return uint64(uptime)
}

func (dev *device) heartbeat() (map[string]interface{}, error) {
        mac := hmac.New(sha256.New, dev.Key)
        hbInfo := map[string]interface{}{
                "Time":   time.Now().Unix(),
		"Memory": getMemInfo(),
		"Uptime": getUptime(),
                "Resets": 0,
                "Type":   "Amperix",
                "Syslog": "",
                "Firmware": map[string]string{
                        "Version":     "0.1",
                        "ReleaseTime": "not yet",
                        "Build":       "from git",
                        "Tag":         "<unknown>",
                },
                "config": map[string]interface{}{
                        "lan": map[string]interface{}{
                                "enabled":  true,
                                "protocol": "dhcp",
                        },
                },
        }
        hbData, err := json.Marshal(hbInfo)
        if err != nil {
	       log.Printf("Error-1\n")
                return nil, err
        }

        hbURL, _ := url.Parse(dev.regdevAPI + "/" + dev.ID + "/status")
        params := url.Values{
                "ts": []string{strconv.FormatInt(time.Now().Unix(), 10)},
        }
        mac.Write([]byte(params["ts"][0]))
        mac.Write(hbData)
        params["sig"] = []string{hex.EncodeToString(mac.Sum(nil))}
        hbURL.RawQuery = params.Encode()
        mac.Reset()

        req, err := http.NewRequest("POST", hbURL.String(), bytes.NewReader(hbData))
        if err != nil {
	       log.Printf("Error-2\n")
                return nil, err
        }

        resp, err := http.DefaultClient.Do(req)
        if err != nil {
	       log.Printf("Error-3\n")
                return nil, err
        }
        defer resp.Body.Close()

	log.Println(">>> Heartbeat: %d", resp.StatusCode)
        switch resp.StatusCode {
        case 200:
                var err error
                var body, nonce, iv, macValue []byte

                if body, err = ioutil.ReadAll(resp.Body); err != nil {
		      log.Printf("Error-4\n")
                        return nil, err
                }
                if body, err = hex.DecodeString(string(body)); err != nil {
		      log.Printf("Error-5\n")
                        return nil, err
                }

                if nonce, err = hex.DecodeString(resp.Header.Get("X-Nonce")); err != nil {
		      log.Printf("Error-6\n")
                        return nil, err
                }
                if iv, err = hex.DecodeString(resp.Header.Get("X-IV")); err != nil {
		      log.Printf("Error-7\n")
                        return nil, err
                }
                if macValue, err = hex.DecodeString(resp.Header.Get("X-HMAC")); err != nil {
		      log.Printf("Error-8\n")
                        return nil, err
                }

                mac.Write(body)

                if !hmac.Equal(mac.Sum(nil), macValue) {
                        return nil, errors.New("bad hmac")
                }

                mac.Reset()
                mac.Write(nonce)
                key := mac.Sum(nil)[:16]

                cinst, err := aes.NewCipher(key)
                if err != nil {
		      log.Printf("Error-9\n")
                        return nil, err
                }
                transform := cipher.NewCFBDecrypter(cinst, iv[:])
                transform.XORKeyStream(body, body)

		log.Printf("Body: %s\n", body)
                var content map[string]interface{}
                if err := json.Unmarshal(body, &content); err != nil {
                        log.Panic(err)
                }
                if user, ok := content["linkedTo"].(string); ok {
                        dev.User = user
                } else {
                        dev.User = ""
                }
                return content, nil

        case 404:
                return nil, errDeviceNotRegistered

        default:
                body, _ := ioutil.ReadAll(resp.Body)
		      log.Printf("Error-10\n")
                return nil, errors.New(string(body))
        }
}

func (dev *device) registerSensors() error {
	for id, sens := range dev.Sensors {
		if err := dev.getClient().AddSensor(id, sens.Unit, sens.Port, sens.Factor); err != nil {
			return err
		}
		md := msgp.SensorMetadata{
			Name: &sens.Name,
		}
		if err := dev.getClient().UpdateSensor(id, md); err != nil {
			return err
		}
	}

	return nil
}

func hex2bytes(hex string) string {
    var s []byte
    if m, _ := regexp.MatchString("^[0-9a-f]+$", hex); !m {
        log.Println("not hex")
        return hex
    }

    for i := 0; i < len(hex); i = i + 2 {
        var a int64
        if i+1 >= len(hex) {
            a, _ = strconv.ParseInt(string(hex[i]), 16, 10)
        } else {
            a, _ = strconv.ParseInt(string(hex[i:i+2]), 16, 10)
        }
        v := byte(a)
        s = append(s, v)
    }
    return string(s)

}

func initAmperixDevice() *device {
     devKey    := uci.Get("system", "system", "key")
     devID     := uci.Get("system", "system", "device")
     msgAPI    := uci.Get("msgampc", "settings.main", "api")
     msgRegAPI := uci.Get("msgampc", "settings.main", "regdevapi")

     log.Printf("ID    : '%s'\n", devID);
     log.Printf("Key   : '%s'\n", devKey);
     log.Printf("API   : '%s'\n", msgAPI);
     log.Printf("RegAPI: '%s'\n", msgRegAPI);
        return &device{
                ID:  devID,
                Key: []byte(devKey),
                api: msgAPI,
                regdevAPI: msgRegAPI,
        }
}

func main() {
     tlsConfig.InsecureSkipVerify = true
     http.DefaultTransport.(*http.Transport).TLSClientConfig = &tlsConfig
     
     for i := 1; i < len(os.Args); i++ {
	cmdName := os.Args[i]

	switch os.Args[i] {
	case "-d":
	     debug++

	case "-t":
	     dummy := "{ \"m\" : [[1476088596,34],[1476088597,33],[1476088598,34],[1476088599,34],[1476088600,34],[1476088601,34],[1476088602,34],[1476088603,33],[1476088604,34],[1476088605,34],[1476088606,34],[1476088607,36],[1476088608,34],[1476088609,32],[1476088610,36],[1476088611,36],[1476088612,34],[1476088613,34],[1476088614,33],[1476088615,32],[1476088616,34],[1476088617,33],[1476088618,34],[1476088619,33],[1476088620,34],[1476088621,34],[1476088622,34],[1476088623,34],[1476088624,34],[1476088625,33],[1476088626,34],[1476088627,33],[1476088628,33],[1476088629,33],[1476088630,34],[1476088631,33],[1476088632,33],[1476088633,30],[1476088634,30],[1476088635,33],[1476088636,34],[1476088637,33],[1476088638,33],[1476088639,33],[1476088640,34],[1476088641,34],[1476088642,33],[1476088643,33],[1476088644,34],[1476088645,33],[1476088646,33],[1476088647,33],[1476088648,33],[1476088649,33],[1476088650,34],[1476088651,33],[1476088652,34],[1476088653,33],[1476088654,34],[1476088655,\"nan\"]] }"
	     testConvert(([]byte)(dummy))
             return
	default:
	     log.Fatalf("bad command %v", cmdName)
	}
     }     
     if debug == 0 {
	logger, err := syslog.New(syslog.LOG_INFO, "msgampc")
	defer logger.Close()
	if err != nil {
		log.Fatal("error")
	}
        log.SetOutput(logger)
     }

     log.Printf("Debug set to: %d\n", debug)
     dev = initAmperixDevice()
     dev.readConfig()
     info, err := dev.heartbeat()
     if err := dev.registerSensors(); err != nil {
          log.Println("registerSensors err.")
          log.Printf("%#v%s\n", err, err.Error())
     }
     if err != nil {
	          log.Println("Heartbeat err.")
	          log.Printf("%#v%s\n", err, err.Error())
     }
     log.Println(info)

     log.Println("Call run..")
     dev.Run()
}


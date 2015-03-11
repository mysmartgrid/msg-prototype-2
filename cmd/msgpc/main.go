package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"msgp"
	"net/http"
	"os"
	"time"
)

func createUser(name string) {
	resp, err := http.PostForm("http://localhost:8080/admin/"+name, nil)
	if err != nil {
		log.Println(resp)
		panic(err)
	}
}

func createDevice(user, name string) {
	resp, err := http.PostForm("http://localhost:8080/admin/"+user+"/"+name, nil)
	if err != nil {
		log.Println(resp)
		panic(err)
	}
}

type clientState map[string]map[string][]string

func (cs clientState) save() error {
	data, err := json.MarshalIndent(cs, "", " ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile("msgpc-state.json", data, 0666)
}

func loadState() clientState {
	data, err := ioutil.ReadFile("msgpc-state.json")
	if err != nil {
		log.Panic(err)
	}
	var state clientState
	err = json.Unmarshal(data, &state)
	if err != nil {
		log.Panic(err)
	}
	return state
}

func createUsers(users, devicesPerUser, sensorsPerDev int) clientState {
	result := make(map[string]map[string][]string)
	for i := 0; i < users; i++ {
		un := fmt.Sprintf("u%v", i)
		result[un] = make(map[string][]string)
		createUser(un)
		for j := 0; j < devicesPerUser; j++ {
			dn := fmt.Sprintf("d%v", j)
			result[un][dn] = make([]string, 0, 10)
			createDevice(un, dn)
			client, err := msgp.NewWSClientDevice("ws://[::1]:8080", un, dn, []byte(dn))
			if err != nil {
				log.Panic(err)
			}
			for k := 0; k < sensorsPerDev; k++ {
				sn := fmt.Sprintf("s%v", k)
				err = client.AddSensor(sn)
				result[un][dn] = append(result[un][dn], sn)
			}
		}
	}
	return result
}

func registerDevice(device string, key []byte) error {
	req, err := http.NewRequest("POST", "http://[::1]:8080/ws/regdevice/"+device, nil)
	if err != nil {
		return err
	}
	req.Header["X-Key"] = []string{hex.EncodeToString(key)}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return errors.New(string(body))
	}
	return nil
}

func runRegisteredDevice(device string, key []byte) {
	resp, err := http.Get("http://[::1]:8080/ws/regdevice/" + device)
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200:
		body, _ := ioutil.ReadAll(resp.Body)
		content := make(map[string]string)
		if err := json.Unmarshal(body, &content); err != nil {
			log.Panic(err)
		}
		if user, linked := content["linkedTo"]; linked {
			log.Printf("device registered for user %v", user)
		} else {
			log.Printf("device not linked")
		}

	case 404:
		err := registerDevice(device, key)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("created device on server")
		runRegisteredDevice(device, key)

	default:
		body, _ := ioutil.ReadAll(resp.Body)
		log.Println(string(body))
	}
}

func runDevice(state clientState, user, device string, newSensorProb float64) {
	client, err := msgp.NewWSClientDevice("ws://[::1]:8080", user, device, []byte(device))
	if err != nil {
		log.Panic(err)
	}

	for {
		sensors := state[user][device]

		if rand.Float64() < newSensorProb {
			maxId := 0
			for _, sid := range sensors {
				id := 0
				fmt.Sscanf(sid, "s%v", &id)
				if id > maxId {
					maxId = id
				}
			}
			victim := rand.Int31n(int32(len(sensors)))
			err = client.RemoveSensor(sensors[victim])
			if err != nil {
				log.Panic(err)
			}
			newSensors := make([]string, 0, len(sensors))
			newSensors = append(newSensors, sensors[0:victim]...)
			newSensors = append(newSensors, sensors[victim+1:]...)
			newSid := fmt.Sprintf("s%v", maxId+1)
			newSensors = append(newSensors, newSid)
			err = client.AddSensor(newSid)
			if err != nil {
				log.Panic(err)
			}
			state[user][device] = newSensors
			sensors = newSensors
			state.save()
		}

		values := make(map[string][]msgp.Measurement, len(sensors))
		for _, name := range sensors {
			values[name] = []msgp.Measurement{{time.Now(), rand.Float64()}}
		}
		err = client.Update(values)
		if err != nil {
			log.Panic(err)
		}
		err = client.Rename(fmt.Sprintf("%v-%v", device, rand.Int31n(1000)))
		if err != nil {
			log.Panic(err)
		}
		for _, id := range sensors {
			err = client.RenameSensor(id, fmt.Sprintf("%v-%v", id, rand.Int31n(1000)))
			if err != nil {
				log.Panic(err)
			}
		}
		time.Sleep(1000 * time.Millisecond)
	}
	client.Close()
}

func main() {
	if len(os.Args) < 2 {
		log.Println("usage: msgpc (init uc,dc,sc | run)")
		return
	}

	switch os.Args[1] {
	case "init":
		uc, dc, sc := 0, 0, 0
		fmt.Sscanf(os.Args[2], "%v,%v,%v", &uc, &dc, &sc)
		state := createUsers(uc, dc, sc)
		state.save()

	case "run":
		state := loadState()
		for user, ds := range state {
			if len(os.Args) > 2 && os.Args[2] != user {
				continue
			}
			for device, _ := range ds {
				go runDevice(state, user, device, 0)
			}
		}
		<-make(chan int)

	case "runreg":
		runRegisteredDevice(os.Args[2], []byte(os.Args[3]))
	}
}

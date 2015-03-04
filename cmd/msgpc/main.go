package main

import (
	"encoding/json"
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

func createUsers(users, devicesPerUser, sensorsPerDev int) map[string]map[string][]string {
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

func runDevice(user, device string, sensors []string) {
	client, err := msgp.NewWSClientDevice("ws://[::1]:8080", user, device, []byte(device))
	if err != nil {
		log.Panic(err)
	}

	for {
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
		data, _ := json.MarshalIndent(state, "", " ")
		ioutil.WriteFile("msgpc-state.json", data, 0666)

	case "run":
		data, err := ioutil.ReadFile("msgpc-state.json")
		if err != nil {
			log.Println(err)
			return
		}
		var state map[string]map[string][]string
		err = json.Unmarshal(data, &state)
		if err != nil {
			log.Println(err)
			return
		}
		for user, ds := range state {
			if len(os.Args) > 2 && os.Args[2] != user {
				continue
			}
			for device, sensors := range ds {
				go runDevice(user, device, sensors)
			}
		}
		<-make(chan int)
	}
}

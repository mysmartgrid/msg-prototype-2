package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"msgp"
	"net/http"
	"net/url"
	"math/rand"
	"time"
)

func createUser(id int) {
	resp, err := http.PostForm("http://localhost:8080/user/register", url.Values{
		"name": []string{fmt.Sprintf("%v", id)},
	})
	if err != nil {
		log.Println(resp)
		panic(err)
	}
}

func createSensor(user, sensor int) string {
	url, err := url.Parse(fmt.Sprintf("http://localhost:8080/api/value/%v/%v", user, sensor))
	if err != nil {
		panic(err)
	}
	req := &http.Request{
		Method: "PUT",
		URL:    url,
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(resp)
		panic(err)
	}
	var line [64]byte
	n, err := resp.Body.Read(line[:])
	if err != nil && err != io.EOF {
		log.Println(err)
		panic(err)
	}
	resp.Body.Close()
	return string(line[:n])
}

func createSensors() {
	allSensors := make(map[string]map[string]string)

	for i := 0; i < 10; i++ {
		createUser(i)
		is := fmt.Sprintf("%v", i)
		allSensors[is] = make(map[string]string)
		for j := 0; j < 10; j++ {
			st := createSensor(i, j)
			allSensors[is][fmt.Sprintf("%v", j)] = st
			log.Println(i, j, st)
		}
	}

	stream, err := json.MarshalIndent(allSensors, "", "	")
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile("allsensors.json", stream, 0660)
	if err != nil {
		panic(err)
	}
}

func runUser(user string, sensors map[string]string) {
	client, err := msgp.NewWSClientDevice("ws://[::1]:8080", "a", "a", []byte("a"))
	if err != nil {
		log.Panic(err)
	}

	//	err = client.AddSensor("a")
	//	if err != nil {
	//		log.Panic(err)
	//	}
	//
	//	err = client.AddSensor("b")
	//	if err != nil {
	//		log.Panic(err)
	//	}

	for {
		err = client.Update(map[string][]msgp.Measurement{
			"a": []msgp.Measurement{{time.Now().Add(time.Second), rand.Float64()}},
			"b": []msgp.Measurement{{time.Now().Add(time.Second), rand.Float64()}},
		})
		if err != nil {
			log.Panic(err)
		}
		time.Sleep(1 * time.Second)
	}
	client.Close()
}

func main() {
	runUser("a", nil)

	//	createSensors()

	//	stream, err := ioutil.ReadFile("allsensors.json")
	//	if err != nil {
	//		panic(err)
	//	}
	//
	//	var allSensors map[string]map[string]string
	//	err = json.Unmarshal(stream, &allSensors)
	//	if err != nil {
	//		panic(err)
	//	}
	//
	//	for user, sensors := range allSensors {
	//		go runUser(user, sensors)
	//	}
	//
	//	time.Sleep(120 * time.Second)
}

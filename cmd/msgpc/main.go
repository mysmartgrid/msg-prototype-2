package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
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
	headers := http.Header{
		"Sec-WebSocket-Protocol": []string{"msgp-1"},
	}
	ws, resp, err := websocket.DefaultDialer.Dial("ws://[::1]:8080/ws/"+user, headers)
	if err != nil {
		log.Panic(resp, err)
	}

	go func() {
		for {
			for sensor, token := range sensors {
				msg := fmt.Sprintf(`{"cmd": "update", "args": {"sensor": %q, "values": [[%v, %v]], "token": %q}}`,
					sensor, time.Now().UnixNano()/1e6, rand.Float64(), token)
				ws.WriteMessage(websocket.TextMessage, []byte(msg))
				waitTime := 90 + rand.Float64()*20
				time.Sleep(time.Duration(waitTime) * time.Millisecond)
			}
		}
	}()

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			panic(err)
		}
		// TODO measure things
	}
}

func main() {
	stream, err := ioutil.ReadFile("allsensors.json")
	if err != nil {
		panic(err)
	}

	var allSensors map[string]map[string]string
	err = json.Unmarshal(stream, &allSensors)
	if err != nil {
		panic(err)
	}

	for user, sensors := range allSensors {
		go runUser(user, sensors)
	}

	time.Sleep(120 * time.Second)
}

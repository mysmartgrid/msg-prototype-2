package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"time"
	"msgp/ws"
)

type cmdlineArgs struct {
	listen, assets, templates *string
}

var args = cmdlineArgs{
	listen:    flag.String("listen", ":8080", "listen address"),
	assets:    flag.String("assets", "./assets", "assets path"),
	templates: flag.String("templates", "./templates", "template path"),
}

var templates *template.Template

func init() {
	flag.Parse()

	switch fi, err := os.Stat(*args.assets); true {
	case err != nil:
		log.Fatal("bad -assets: ", err)
		os.Exit(1)

	case !fi.IsDir():
		log.Fatal("-assets is not a directory")
		os.Exit(1)
	}

	var err error
	templates, err = template.ParseGlob(path.Join(*args.templates, "*.html"))
	if err != nil {
		log.Fatal("error parsing templates: ", err)
		os.Exit(1)
	}
}

func staticTemplate(name string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		var scheme = "ws://"
		if r.TLS != nil {
			scheme = "wss://"
		}
		var url = scheme + r.Host + "/ws"
		templates.ExecuteTemplate(w, name, url)
	}
}

var upgrader = websocket.Upgrader{
	HandshakeTimeout: 10 * time.Second,
	Subprotocols:     []string{"msgp-1"},
}

var conn *ws.Dispatcher
var send chan string

func wsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	wss, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		switch err.(type) {
		case websocket.HandshakeError:
			http.Error(w, "Handshake failed", 400)

		default:
			log.Println(err)
		}
		return
	}

	send = make(chan string)

	wss.SetReadLimit(4096)

	conn = &ws.Dispatcher{
		Socket: wss,
	}

	go func() {
		for {
			val := <-send
			ts := time.Now().Unix()
			line := fmt.Sprintf("[%v, %v]", ts, val)
			conn.Write(line)
		}
	}()

	conn.Run()
}

func postHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	val, present := r.PostForm["value"]
	if !present {
		http.Error(w, "value missing", 400)
		return
	}

	send <- val[0]
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", staticTemplate("index"))
	router.HandleFunc("/ws", wsHandler)
	router.HandleFunc("/api/value", postHandler).Methods("POST")
	router.PathPrefix("/").Handler(http.FileServer(http.Dir(*args.assets)))

	http.Handle("/", router)

	log.Print("Listening on ", *args.listen)
	if err := http.ListenAndServe(*args.listen, nil); err != nil {
		log.Fatal("failed: ", err)
	}
}

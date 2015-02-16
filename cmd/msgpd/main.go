package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"html/template"
	"log"
	"msgp"
	"msgp/hub"
	"msgp/ws"
	"net/http"
	"os"
	"path"
	"time"
)

type cmdlineArgs struct {
	listen, assets, templates, udbPath *string
}

var args = cmdlineArgs{
	listen:    flag.String("listen", ":8080", "listen address"),
	assets:    flag.String("assets", "./assets", "assets path"),
	templates: flag.String("templates", "./templates", "template path"),
	udbPath:   flag.String("userdb", "", "path to user database"),
}

var templates *template.Template
var db *msgp.Db

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

	if *args.udbPath == "" {
		log.Fatal("-userdb missing")
		os.Exit(1)
	}

	templates = template.New("")

	templates.Funcs(template.FuncMap{
		"activeIfEq": func(this, page string) template.HTMLAttr {
			if this == page {
				return `class="active"`
			} else {
				return ""
			}
		},
		"alertIfMissing": func(this string, missing []string) template.HTMLAttr {
			if missing == nil {
				return ""
			}

			for _, v := range missing {
				if this == v {
					return `style="color: red"`
				}
			}

			return ""
		},
	})

	_, err := templates.ParseGlob(path.Join(*args.templates, "*.html"))
	if err != nil {
		log.Fatal("error parsing templates: ", err)
		os.Exit(1)
	}

	db, err = msgp.OpenDb(*args.udbPath)
	if err != nil {
		log.Fatal("error opening user db: ", err)
		os.Exit(1)
	}
}

func defaultHeaders(fn func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fn(w, r)
	}
}

func wsTemplate(name string) func(http.ResponseWriter, *http.Request) {
	return defaultHeaders(func(w http.ResponseWriter, r *http.Request) {
		var scheme = "ws://"
		if r.TLS != nil {
			scheme = "wss://"
		}
		type ctx struct {
			Ws      string
			Missing []string
			Sensors map[string]bool
		}
		user := db.Find(r.PostFormValue("user"))
		if user == nil {
			templates.ExecuteTemplate(w, name, ctx{Missing: []string{"user"}})
			return
		}
		var url = scheme + r.Host + "/ws/" + user.Name
		templates.ExecuteTemplate(w, name, ctx{Ws: url, Sensors: user.Sensors})
	})
}

func staticTemplate(name string) func(http.ResponseWriter, *http.Request) {
	return defaultHeaders(func(w http.ResponseWriter, r *http.Request) {
		templates.ExecuteTemplate(w, name, nil)
	})
}

var upgrader = websocket.Upgrader{
	HandshakeTimeout: 10 * time.Second,
	Subprotocols:     []string{"msgp-1"},
}

var h = hub.New()

func wsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	user := db.Find(mux.Vars(r)["user"])
	if user == nil {
		http.Error(w, "not found", 404)
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

	wss.SetReadLimit(4096)

	conn := &ws.Dispatcher{
		Socket: wss,
	}
	hconn := h.Connect()
	hconn.Subscribe(user.Name)

	go func() {
		for {
			val, open := <-hconn.R
			if !open {
				return
			}
			ts := time.Now().Unix()
			line := fmt.Sprintf("[%v, %v]", ts, val.Value)
			conn.Write(line)
		}
	}()

	conn.Run()
	hconn.Close()
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

	user := mux.Vars(r)["user"]
	sensor := mux.Vars(r)["sensor"]

	db.Update(user, func(u *msgp.User) error {
		u.Sensors[sensor] = true
		return nil
	})

	h.Publish(user, fmt.Sprintf("%q, %v", sensor, val[0]))
}

func userRegister(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")

	type ctx struct {
		Missing []string
		Error   string
	}

	switch {
	case name == "":
		templates.ExecuteTemplate(w, "register", ctx{Missing: []string{"name"}})

	default:
		user, err := db.Add(name)
		if err != nil {
			templates.ExecuteTemplate(w, "register", ctx{Error: err.Error()})
			return
		}
		templates.ExecuteTemplate(w, "register_done", user.AuthToken)
	}
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	db.ForEach(func(u *msgp.User) error {
		w.Write([]byte(fmt.Sprintf("<div>User %v Token <b>%v</b></div>", u.Name, u.AuthToken)))
		for s := range u.Sensors {
			w.Write([]byte(fmt.Sprintf("<div>&nbsp;&nbsp;Sensor %v</div>", s)))
		}
		return nil
	})
}

func main() {
	router := mux.NewRouter()

	router.HandleFunc("/", wsTemplate("index")).Methods("POST")
	router.HandleFunc("/", staticTemplate("index")).Methods("GET")
	router.HandleFunc("/user/register", staticTemplate("register")).Methods("GET")
	router.HandleFunc("/user/register", defaultHeaders(userRegister)).Methods("POST")

	router.HandleFunc("/admin", defaultHeaders(adminHandler))

	router.HandleFunc("/ws/{user}", wsHandler)
	router.HandleFunc("/api/value/{user}/{sensor}", postHandler).Methods("POST")
	router.PathPrefix("/").Handler(http.FileServer(http.Dir(*args.assets)))

	http.Handle("/", router)

	log.Print("Listening on ", *args.listen)
	if err := http.ListenAndServe(*args.listen, nil); err != nil {
		log.Fatal("failed: ", err)
	}
}

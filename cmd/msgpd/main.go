package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"msgp"
	"msgp/hub"
	"net/http"
	"os"
	"path"
)

type cmdlineArgs struct {
	listen                                       *string
	assets, templates                            *string
	udbPath                                      *string
	influxAddr, influxDb, influxUser, influxPass *string
}

var args = cmdlineArgs{
	listen:     flag.String("listen", ":8080", "listen address"),
	assets:     flag.String("assets", "./assets", "assets path"),
	templates:  flag.String("templates", "./templates", "template path"),
	udbPath:    flag.String("userdb", "", "path to user database"),
	influxAddr: flag.String("influx-addr", "", "address of influxdb"),
	influxDb:   flag.String("influx-db", "", "influxdb database name"),
	influxUser: flag.String("influx-user", "", "username for influxdb"),
	influxPass: flag.String("influx-pass", "", "password for influxdb"),
}

var templates *template.Template
var db msgp.Db

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

	bailIfMissing := func(value *string, flag string) {
		if *value == "" {
			log.Fatal(flag + " missing")
			os.Exit(1)
		}
	}

	bailIfMissing(args.udbPath, "-userdb")
	bailIfMissing(args.influxAddr, "-influx-addr")
	bailIfMissing(args.influxDb, "-influx-db")
	bailIfMissing(args.influxUser, "-influx-user")
	bailIfMissing(args.influxPass, "-influx-pass")

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

	db, err = msgp.OpenDb(*args.udbPath, *args.influxAddr, *args.influxDb, *args.influxUser, *args.influxPass)
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
			User    msgp.User
		}
		db.View(func(tx msgp.DbTx) error {
			user := tx.User(r.PostFormValue("user"))
			if user == nil {
				templates.ExecuteTemplate(w, name, ctx{Missing: []string{"user"}})
				return nil
			}
			var url = scheme + r.Host + "/ws/user/" + r.PostFormValue("user")
			return templates.ExecuteTemplate(w, name, ctx{Ws: url, User: user})
		})
	})
}

func staticTemplate(name string) func(http.ResponseWriter, *http.Request) {
	return defaultHeaders(func(w http.ResponseWriter, r *http.Request) {
		templates.ExecuteTemplate(w, name, nil)
	})
}

var h = hub.New()

func wsHandlerUser(w http.ResponseWriter, r *http.Request) {
	x := msgp.WSAPI{
		Db:      db,
		Hub:     h,
		User:    mux.Vars(r)["user"],
		Writer:  w,
		Request: r,
	}
	defer x.Close()
	x.RunUser()
}

func wsHandlerDevice(w http.ResponseWriter, r *http.Request) {
	x := msgp.WSAPI{
		Db:      db,
		Hub:     h,
		User:    mux.Vars(r)["user"],
		Writer:  w,
		Request: r,
	}
	defer x.Close()
	x.RunDevice(mux.Vars(r)["device"])
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
		db.Update(func(tx msgp.DbTx) error {
			_, err := tx.AddUser(name)
			if err != nil {
				templates.ExecuteTemplate(w, "register", ctx{Error: err.Error()})
				return err
			}
			templates.ExecuteTemplate(w, "register_done", nil)
			return nil
		})
	}
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	db.View(func(tx msgp.DbTx) error {
		for name, user := range tx.Users() {
			w.Write([]byte(fmt.Sprintf("<div>User %v</div>", name)))
			for name, device := range user.Devices() {
				w.Write([]byte(fmt.Sprintf("<div>&nbsp;&nbsp;Device %v, key %v</div>", name, device.Key())))
				for name, _ := range device.Sensors() {
					w.Write([]byte(fmt.Sprintf("<div>&nbsp;&nbsp;&nbsp;&nbsp;Sensor %v</div>", name)))
				}
			}
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

	router.HandleFunc("/ws/user/{user}", wsHandlerUser)
	router.HandleFunc("/ws/device/{user}/{device}", wsHandlerDevice)
	router.PathPrefix("/").Handler(http.FileServer(http.Dir(*args.assets)))

	http.Handle("/", router)

	log.Print("Listening on ", *args.listen)
	if err := http.ListenAndServe(*args.listen, nil); err != nil {
		log.Fatal("failed: ", err)
	}
}

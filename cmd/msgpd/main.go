package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"html/template"
	"log"
	"msgp"
	msgpdb "msgp/db"
	"msgp/hub"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

type cmdlineArgs struct {
	listen                                       *string
	assets, templates                            *string
	udbPath                                      *string
	influxAddr, influxDb, influxUser, influxPass *string
}

const (
	sessionCookieVersion = 1
)

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
var db msgpdb.Db
var cookieStore = sessions.NewCookieStore([]byte("test-key"))

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
		"activeIfAt": func(this, page string) template.HTMLAttr {
			if strings.SplitN(page, ":", 2)[0] == this {
				return `class="active"`
			} else {
				return ""
			}
		},
		"sessionFlag": func(flag, set string) bool {
			for _, sflag := range strings.Split(set, ":")[1:] {
				if flag == sflag {
					return true
				}
			}
			return false
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

	db, err = msgpdb.OpenDb(*args.udbPath, *args.influxAddr, *args.influxDb, *args.influxUser, *args.influxPass)
	if err != nil {
		log.Fatal("error opening user db: ", err)
		os.Exit(1)
	}
}

func getSession(w http.ResponseWriter, r *http.Request) *sessions.Session {
	session, _ := cookieStore.Get(r, "msgp-session")
	version, good := session.Values["-session-version"].(int)
	if !good || version != sessionCookieVersion {
		session.Values = make(map[interface{}]interface{})
		session.Values["-session-version"] = sessionCookieVersion
		session.Save(r, w)
	}
	return session
}

func defaultHeaders(fn func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fn(w, r)
	}
}

func wsTemplate(name string) func(http.ResponseWriter, *http.Request) {
	return defaultHeaders(func(w http.ResponseWriter, r *http.Request) {
		session := getSession(w, r)
		userId, found := session.Values["user"]
		if !found {
			http.Error(w, "bad request", 400)
			return
		}
		var scheme = "ws://"
		if r.TLS != nil {
			scheme = "wss://"
		}
		type ctx struct {
			Ws      string
			Missing []string
			User    msgpdb.User
		}
		db.View(func(tx msgpdb.Tx) error {
			user := tx.User(userId.(string))
			if user == nil {
				http.Error(w, "not found", 404)
				return nil
			}
			var url = scheme + r.Host + "/ws/user/" + user.Id() + "/" + session.Values["wsToken"].(string)
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
	session := getSession(w, r)

	token, good := session.Values["wsToken"].(string)
	if !good || token != mux.Vars(r)["token"] {
		http.Error(w, "bad request", 400)
		return
	}

	x := msgp.WSAPI{
		Db:      db,
		Hub:     h,
		Writer:  w,
		Request: r,
		User:    mux.Vars(r)["user"],
	}
	defer x.Close()
	x.RunUser()
}

func wsHandlerDevice(w http.ResponseWriter, r *http.Request) {
	x := msgp.WSAPI{
		Db:      db,
		Hub:     h,
		Writer:  w,
		Request: r,
		User:    mux.Vars(r)["user"],
	}
	defer x.Close()
	x.RunDevice(mux.Vars(r)["device"])
}

func handlerRegisteredDevice(w http.ResponseWriter, r *http.Request) {
	db.View(func(tx msgpdb.Tx) error {
		dev := tx.Device(mux.Vars(r)["device"])
		if dev == nil {
			http.Error(w, "not found", 404)
		}
		return nil
	})
	// TODO
}

func registerDevice(w http.ResponseWriter, r *http.Request) {
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
	err = db.Update(func(tx msgpdb.Tx) error {
		return tx.AddDevice(mux.Vars(r)["device"], key)
	})
	if err != nil {
		http.Error(w, err.Error(), 400)
	}
}

func doLogin(w http.ResponseWriter, r *http.Request) {
	session, _ := cookieStore.Get(r, "msgp-session")
	session.Values["user"] = r.PostFormValue("user")
	session.Values["wsToken"] = fmt.Sprintf("%x", sha256.Sum256([]byte(time.Now().String())))
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func doLogout(w http.ResponseWriter, r *http.Request) {
	session, _ := cookieStore.Get(r, "msgp-session")
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
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
		db.Update(func(tx msgpdb.Tx) error {
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
	tstr := `
<div>Users:</div>
<ul>
{{range $name, $user := (.Users)}}
	<li>
		<div>{{$name}}</div>
		<ul>
		{{range $name, $dev := $user.Devices}}
			<li>
				<div>{{$dev.Id}}({{$dev.Name}}) key 0x{{$dev.Key | printf "%x"}}</div>
				<ul>
				{{range $name, $sens := $dev.Sensors}}
					<li>{{$sens.Id}}({{$sens.Name}})</li>
				{{end}}
				</ul>
			</li>
		{{end}}
		</ul>
	</li>
{{end}}
</ul>

<div>Registered devices:</div>
<ul>
{{range $id, $link := .Devices}}
	<li>{{$id}} 0x{{$link.Key | printf "%x"}} -> {{$link.UserLink}}</li>
{{end}}
</ul>
`

	db.View(func(tx msgpdb.Tx) error {
		t, err := template.New("").Parse(tstr)
		if err != nil {
			w.Write([]byte(err.Error()))
			return nil
		}
		t.Execute(w, tx)
		return nil
	})
}

func adminAddUser(w http.ResponseWriter, r *http.Request) {
	err := db.Update(func(tx msgpdb.Tx) error {
		_, err := tx.AddUser(mux.Vars(r)["user"])
		return err
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func adminAddDevice(w http.ResponseWriter, r *http.Request) {
	err := db.Update(func(tx msgpdb.Tx) error {
		u := tx.User(mux.Vars(r)["user"])
		_, err := u.AddDevice(mux.Vars(r)["device"], []byte(mux.Vars(r)["device"]))
		return err
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func loggedInSwitch(in, out func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session := getSession(w, r)
		if _, found := session.Values["user"]; found {
			in(w, r)
		} else {
			out(w, r)
		}
	}
}

func main() {
	router := mux.NewRouter()

	router.HandleFunc("/", loggedInSwitch(wsTemplate("index_user"), staticTemplate("index_nouser"))).Methods("GET")
	router.HandleFunc("/user/login", staticTemplate("user-login")).Methods("GET")
	router.HandleFunc("/user/login", doLogin).Methods("POST")
	router.HandleFunc("/user/logout", doLogout).Methods("GET")
	router.HandleFunc("/user/register", staticTemplate("register")).Methods("GET")
	router.HandleFunc("/user/register", defaultHeaders(userRegister)).Methods("POST")

	router.HandleFunc("/admin", defaultHeaders(adminHandler))
	router.HandleFunc("/admin/{user}", defaultHeaders(adminAddUser)).Methods("POST")
	router.HandleFunc("/admin/{user}/{device}", defaultHeaders(adminAddDevice)).Methods("POST")

	router.HandleFunc("/ws/user/{user}/{token}", wsHandlerUser)
	router.HandleFunc("/ws/device/{user}/{device}", wsHandlerDevice)
	router.HandleFunc("/ws/regdevice/{device}", handlerRegisteredDevice).Methods("GET")
	router.HandleFunc("/ws/regdevice/{device}", registerDevice).Methods("POST")
	router.PathPrefix("/").Handler(http.FileServer(http.Dir(*args.assets)))

	http.Handle("/", router)

	log.Print("Listening on ", *args.listen)
	if err := http.ListenAndServe(*args.listen, nil); err != nil {
		log.Fatal("failed: ", err)
	}
}

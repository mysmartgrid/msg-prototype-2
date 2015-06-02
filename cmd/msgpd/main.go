package main

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	msgp "github.com/mysmartgrid/msg-prototype-2"
	msgpdb "github.com/mysmartgrid/msg-prototype-2/db"
	"github.com/mysmartgrid/msg-prototype-2/hub"
	"github.com/mysmartgrid/msg-prototype-2/regdev"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

type cmdlineArgs struct {
	listen                                       *string
	assets, templates                            *string
	udbPath, devdbPath                           *string
	influxAddr, influxDb, influxUser, influxPass *string
	sslCert, sslKey                              *string
	deviceProxyConfig                            *string
	motherlode                                   *bool
}

const (
	sessionCookieVersion = 1
)

var args = cmdlineArgs{
	listen:            flag.String("listen", ":8080", "listen address"),
	assets:            flag.String("assets", "./assets", "assets path"),
	templates:         flag.String("templates", "./templates", "template path"),
	udbPath:           flag.String("userdb", "", "path to user database"),
	devdbPath:         flag.String("devdb", "", "path to device database"),
	influxAddr:        flag.String("influx-addr", "", "address of influxdb"),
	influxDb:          flag.String("influx-db", "", "influxdb database name"),
	influxUser:        flag.String("influx-user", "", "username for influxdb"),
	influxPass:        flag.String("influx-pass", "", "password for influxdb"),
	sslCert:           flag.String("ssl-cert", "", "ssl certificate file"),
	sslKey:            flag.String("ssl-key", "", "ssl key file"),
	motherlode:        flag.Bool("motherlode", false, ""),
	deviceProxyConfig: flag.String("dev-proxy-conf", "", "device->msg proxy config"),
}

var templates *template.Template
var cookieStore = sessions.NewCookieStore([]byte("test-key"))
var proxyConf struct {
	PostUrl    string
	CertPath   string
	DeviceKeys map[string]string
}
var oldApiPostClient *http.Client

var db msgpdb.Db
var devdb regdev.Db
var h = hub.New()

var apiCtx msgp.WsApiContext

func init() {
	flag.Parse()

	if *args.deviceProxyConfig != "" {
		fcontents, err := ioutil.ReadFile(*args.deviceProxyConfig)
		if err != nil {
			log.Fatalf("could not read device key map: %v", err.Error())
		}
		if err := json.Unmarshal(fcontents, &proxyConf); err != nil {
			log.Fatalf("could not load device key map: %v", err.Error())
		}
		oldApiPostClient = &http.Client{
			Timeout: 2 * time.Second,
		}
		if proxyConf.CertPath != "" {
			cert, err := ioutil.ReadFile(proxyConf.CertPath)
			if err != nil {
				log.Fatalf("could not load proxy CA: %v", err)
			}

			certPool := x509.NewCertPool()
			if !certPool.AppendCertsFromPEM(cert) {
				log.Fatal("could not parse proxy cert")
			}

			tlsConfig := &tls.Config{
				RootCAs: certPool,
			}
			oldApiPostClient.Transport = &http.Transport{TLSClientConfig: tlsConfig}
		}
	}

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
			log.Fatalf("%v missing", flag)
		}
	}

	bailIfMissing(args.udbPath, "-userdb")
	bailIfMissing(args.devdbPath, "-devdb")
	bailIfMissing(args.influxAddr, "-influx-addr")
	bailIfMissing(args.influxDb, "-influx-db")
	bailIfMissing(args.influxUser, "-influx-user")
	bailIfMissing(args.influxPass, "-influx-pass")

	if *args.sslCert != "" {
		bailIfMissing(args.sslKey, "-ssl-key")
	}
	if *args.sslKey != "" {
		bailIfMissing(args.sslCert, "-ssl-cert")
	}

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
	}

	db, err = msgpdb.OpenDb(*args.udbPath, *args.influxAddr, *args.influxDb, *args.influxUser, *args.influxPass)
	if err != nil {
		log.Fatal("error opening user db: ", err)
	}

	devdb, err = regdev.Open(*args.devdbPath)
	if err != nil {
		log.Fatal("error opening device db: ", err)
	}

	apiCtx = msgp.WsApiContext{
		Db:  db,
		Hub: h,
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

func removeSessionAndNotifyUser(w http.ResponseWriter, r *http.Request, session *sessions.Session) {
	session.Options.MaxAge = -1
	session.Save(r, w)
	templates.ExecuteTemplate(w, "index_nouser", struct{ Error string }{"Your session has expired"})
}

func wsTemplate(name string) func(http.ResponseWriter, *http.Request) {
	return defaultHeaders(func(w http.ResponseWriter, r *http.Request) {
		session := getSession(w, r)
		userId, found := session.Values["user"]
		if !found {
			removeSessionAndNotifyUser(w, r, session)
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
				removeSessionAndNotifyUser(w, r, session)
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

func wsHandlerUser(w http.ResponseWriter, r *http.Request) {
	session := getSession(w, r)

	token, good := session.Values["wsToken"].(string)
	if !good || token != mux.Vars(r)["token"] {
		http.Error(w, "bad request", 400)
		return
	}

	x := msgp.WsUserApi{
		Ctx:     &apiCtx,
		User:    mux.Vars(r)["user"],
		Writer:  w,
		Request: r,
	}
	defer x.Close()
	x.Run()
}

func wsHandlerDevice(w http.ResponseWriter, r *http.Request) {
	x := msgp.WsDevApi{
		User:    mux.Vars(r)["user"],
		Device:  mux.Vars(r)["device"],
		Writer:  w,
		Request: r,
	}
	if oldApiPostClient != nil {
		x.Key = []byte(proxyConf.DeviceKeys[x.Device])
		x.PostUrl = proxyConf.PostUrl
		x.PostClient = oldApiPostClient
	}
	apiCtx.RegisterDevice(&x)
	defer func() {
		apiCtx.RemoveDevice(&x)
		x.Close()
	}()
	x.Run()
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
{{range $name, $user := (.U.Users)}}
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
{{range $id, $link := .D.Devices}}
	<li>{{$id}} 0x{{$link.Key | printf "%x"}} -> {{userOfLink $link}}</li>
{{end}}
</ul>
<strong>done</strong>
`

	db.View(func(utx msgpdb.Tx) error {
		return devdb.View(func(dtx regdev.Tx) error {
			t := template.New("")
			t.Funcs(template.FuncMap{
				"userOfLink": func(link regdev.RegisteredDevice) template.HTML {
					user, linked := link.UserLink()
					if linked {
						return template.HTML(template.HTMLEscapeString(user))
					}
					return "<i>none</i>"
				},
			})
			_, err := t.Parse(tstr)
			if err != nil {
				w.Write([]byte(err.Error()))
				return nil
			}
			type ctx struct {
				U msgpdb.Tx
				D regdev.Tx
			}
			err = t.Execute(w, ctx{utx, dtx})
			if err != nil {
				w.Write([]byte("<br/>" + err.Error()))
			}
			return nil
		})
	})
}

func adminAddUser(w http.ResponseWriter, r *http.Request) {
	user := r.FormValue("user")

	err := db.Update(func(tx msgpdb.Tx) error {
		_, err := tx.AddUser(user)
		return err
	})
	if err != nil {
		http.Error(w, err.Error(), 400)
	}
}

func adminUser_AddDev(w http.ResponseWriter, r *http.Request) {
	user := mux.Vars(r)["user"]
	device := r.FormValue("device")
	key := r.FormValue("key")
	rawKey, err := hex.DecodeString(key)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	err = db.Update(func(tx msgpdb.Tx) error {
		user := tx.User(user)
		_, err := user.AddDevice(device, rawKey)
		return err
	})
	if err != nil {
		http.Error(w, err.Error(), 400)
	}
}

func adminDevice_AddSensor(w http.ResponseWriter, r *http.Request) {
	user := mux.Vars(r)["user"]
	device := mux.Vars(r)["device"]
	sensor := r.FormValue("sensor")

	err := db.Update(func(tx msgpdb.Tx) error {
		user := tx.User(user)
		device := user.Device(device)
		_, err := device.AddSensor(sensor)
		return err
	})
	if err != nil {
		http.Error(w, err.Error(), 400)
	}
}

func adminDevice_RemoveSensor(w http.ResponseWriter, r *http.Request) {
	user := mux.Vars(r)["user"]
	device := mux.Vars(r)["device"]
	sensor := r.FormValue("sensor")

	err := db.Update(func(tx msgpdb.Tx) error {
		user := tx.User(user)
		device := user.Device(device)
		return device.RemoveSensor(sensor)
	})
	if err != nil {
		http.Error(w, err.Error(), 400)
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

func userDevices(w http.ResponseWriter, r *http.Request) {
	session := getSession(w, r)
	db.View(func(tx msgpdb.Tx) error {
		u := tx.User(session.Values["user"].(string))
		if u == nil {
			removeSessionAndNotifyUser(w, r, session)
			return nil
		}

		templates.ExecuteTemplate(w, "user-devices", u)
		return nil
	})
}

func userDevicesAdd(w http.ResponseWriter, r *http.Request) {
	type context struct {
		Missing []string
	}

	session := getSession(w, r)
	if r.Method == "GET" {
		templates.ExecuteTemplate(w, "user-devices-add", context{})
		return
	}

	devId := r.FormValue("device")

	if devId == "" {
		templates.ExecuteTemplate(w, "user-devices-add", context{[]string{"device"}})
		return
	}

	db.Update(func(utx msgpdb.Tx) error {
		return devdb.Update(func(dtx regdev.Tx) error {
			user := utx.User(session.Values["user"].(string))
			if user == nil {
				http.Error(w, "not authorized", 401)
				return errors.New("")
			}
			dev := dtx.Device(devId)
			if dev == nil {
				http.Error(w, "no such device", 404)
				return errors.New("")
			}
			if err := dev.LinkTo(user.Id()); err != nil {
				http.Error(w, err.Error(), 400)
				return errors.New("")
			}
			_, err := user.AddDevice(dev.Id(), dev.Key())
			if err != nil {
				http.Error(w, err.Error(), 500)
				return errors.New("")
			}
			http.Redirect(w, r, "/user/devices", 303)
			return nil
		})
	})
}

func userDevicesRemove(w http.ResponseWriter, r *http.Request) {
	session := getSession(w, r)
	devId := mux.Vars(r)["device"]
	db.Update(func(utx msgpdb.Tx) error {
		return devdb.Update(func(dtx regdev.Tx) error {
			user := utx.User(session.Values["user"].(string))
			if user == nil {
				http.Error(w, "not authorized", 401)
				return errors.New("")
			}
			dev := dtx.Device(devId)
			if dev == nil {
				http.Error(w, "no such device", 404)
				return errors.New("")
			}
			if err := dev.Unlink(); err != nil {
				http.Error(w, err.Error(), 500)
				return errors.New("")
			}
			if err := user.RemoveDevice(devId); err != nil {
				http.Error(w, err.Error(), 500)
				return errors.New("")
			}
			http.Redirect(w, r, "/user/devices", 303)
			return nil
		})
	})
}

func main() {
	router := mux.NewRouter()
	server := regdev.DeviceServer{Db: devdb}

	router.HandleFunc("/", loggedInSwitch(wsTemplate("index_user"), staticTemplate("index_nouser"))).Methods("GET")
	router.HandleFunc("/user/login", staticTemplate("user-login")).Methods("GET")
	router.HandleFunc("/user/login", doLogin).Methods("POST")
	router.HandleFunc("/user/logout", doLogout).Methods("GET")
	router.HandleFunc("/user/register", staticTemplate("register")).Methods("GET")
	router.HandleFunc("/user/register", defaultHeaders(userRegister)).Methods("POST")
	router.HandleFunc("/user/devices", defaultHeaders(userDevices)).Methods("GET")
	router.HandleFunc("/user/devices/add", defaultHeaders(userDevicesAdd)).Methods("GET", "POST")
	router.HandleFunc("/user/devices/remove/{device}", userDevicesRemove).Methods("POST")

	if *args.motherlode {
		router.HandleFunc("/admin", defaultHeaders(adminHandler))
		router.HandleFunc("/admin/add-user", adminAddUser).Methods("POST")
		router.HandleFunc("/admin/user/{user}/add-device", adminUser_AddDev).Methods("POST")
		router.HandleFunc("/admin/device/{user}/{device}/add-sensor", adminDevice_AddSensor).Methods("POST")
		router.HandleFunc("/admin/device/{user}/{device}/remove-sensor", adminDevice_RemoveSensor).Methods("POST")
	}

	router.HandleFunc("/ws/user/{user}/{token}", wsHandlerUser)
	router.HandleFunc("/ws/device/{user}/{device}", wsHandlerDevice)
	router.PathPrefix("/regdev").Handler(&server)
	router.PathPrefix("/").Handler(http.FileServer(http.Dir(*args.assets)))

	http.Handle("/", router)

	log.Print("Listening on ", *args.listen)
	if *args.sslCert != "" {
		log.Printf("Using SSL cert and key %v, %v", *args.sslCert, *args.sslKey)

		if err := http.ListenAndServeTLS(*args.listen, *args.sslCert, *args.sslKey, nil); err != nil {
			log.Fatal("failed: ", err)
		}
	} else {
		if err := http.ListenAndServe(*args.listen, nil); err != nil {
			log.Fatal("failed: ", err)
		}
	}
}

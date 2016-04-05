package main

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	msgp "github.com/mysmartgrid/msg-prototype-2"
	msgpdb "github.com/mysmartgrid/msg-prototype-2/db"
	"github.com/mysmartgrid/msg-prototype-2/hub"
	"github.com/mysmartgrid/msg-prototype-2/regdev"
	"github.com/mysmartgrid/msg2api"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

type postgresConfig struct {
	User     string `toml:"user"`
	Password string `toml:"password"`
	Address  string `toml:"address"`
	Port     string `toml:"port"`
	Database string `toml:"database"`
}

type tlsConfig struct {
	Cert string `toml:"certificate"`
	Key  string `toml:"key"`
}

type benchmarkConfig struct {
	DoBenchmark bool          `toml:"do-benchmark"`
	UserCount   int           `toml:"users"`
	DeviceCount int           `toml:"devices"`
	SensorCount int           `toml:"sensors"`
	Duration    time.Duration `toml:"duration"`
}

type serverConfig struct {
	ListenAddr        string          `toml:"listen"`
	AssetsDir         string          `toml:"assets-dir"`
	TemplatesDir      string          `toml:"templates-dir"`
	DbDir             string          `toml:"db-dir"`
	Postgres          postgresConfig  `toml:"postgres"`
	TLS               tlsConfig       `toml:"tls"`
	DeviceProxyConfig string          `toml:"device-proxy-config"`
	EnableAdminOps    bool            `toml:"motherlode"`
	Benchmark         benchmarkConfig `toml:"benchmark"`
}

const (
	sessionCookieVersion = 1
)

var configFile = flag.String("config", "", "configuration file")

var config serverConfig

var templates *template.Template
var cookieStore = sessions.NewCookieStore([]byte("test-key"))
var proxyConf struct {
	PostURL    string
	CertPath   string
	DeviceKeys map[string]string
}
var oldAPIPostClient *http.Client
var db msgpdb.Db
var devdb regdev.Db
var h = hub.New()

var apiCtx msgp.WsAPIContext

func init() {
	flag.Parse()

	if *configFile == "" {
		log.Fatal("missing -config")
	}

	configData, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatalf("could not read config file: %v", err.Error())
	}
	if err := toml.Unmarshal(configData, &config); err != nil {
		log.Fatalf("could not load config file: %v", err.Error())
	}

	if config.DeviceProxyConfig != "" {
		configData, err := ioutil.ReadFile(config.DeviceProxyConfig)
		if err != nil {
			log.Fatalf("could not read device key map: %v", err.Error())
		}
		if err := json.Unmarshal(configData, &proxyConf); err != nil {
			log.Fatalf("could not load device key map: %v", err.Error())
		}
		oldAPIPostClient = &http.Client{
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
			oldAPIPostClient.Transport = &http.Transport{TLSClientConfig: tlsConfig}
		}
	}

	orDefault := func(value *string, def string) {
		if *value == "" {
			*value = def
		}
	}

	orDefault(&config.ListenAddr, "localhost:8080")
	orDefault(&config.AssetsDir, "./assets")
	orDefault(&config.TemplatesDir, "./templates")
	orDefault(&config.DbDir, ".")

	if config.Postgres.User == "" || config.Postgres.Address == "" || config.Postgres.Database == "" {
		log.Fatal("postgres config incomplete")
	}

	switch fi, err := os.Stat(config.AssetsDir); true {
	case err != nil:
		log.Fatalf("bad assets-dir: %v", err.Error())
		os.Exit(1)

	case !fi.IsDir():
		log.Fatal("assets-dir is not a directory")
		os.Exit(1)
	}

	if config.TLS.Key != "" && config.TLS.Cert == "" {
		log.Fatal("tls cert missing")
	}
	if config.TLS.Cert != "" && config.TLS.Key == "" {
		log.Fatal("tls key missing")
	}

	templates = template.New("")

	templates.Funcs(template.FuncMap{
		"activeIfAt": func(this, page string) template.HTMLAttr {
			if strings.SplitN(page, ":", 2)[0] == this {
				return `class="active"`
			}
			return ""
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

	_, err = templates.ParseGlob(path.Join(config.TemplatesDir, "*.html"))
	if err != nil {
		log.Fatal("error parsing templates: ", err)
	}

	db, err = msgpdb.OpenDb(config.Postgres.Address, config.Postgres.Port, config.Postgres.Database,
		config.Postgres.User, config.Postgres.Password)
	if err != nil {
		log.Fatal("error opening user db: ", err)
	}

	devdb, err = regdev.Open(config.DbDir + "/devices.db")
	if err != nil {
		log.Fatal("error opening device db: ", err)
	}

	apiCtx = msgp.WsAPIContext{Db: db, Hub: h}
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
		userID, found := session.Values["user"]
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
			user := tx.User(userID.(string))
			if user == nil {
				removeSessionAndNotifyUser(w, r, session)
				return nil
			}
			var url = scheme + r.Host + "/ws/user/" + user.ID() + "/" + session.Values["wsToken"].(string)
			return templates.ExecuteTemplate(w, name, ctx{Ws: url, User: user})
		})
	})
}

func staticTemplate(name string) func(http.ResponseWriter, *http.Request) {
	return defaultHeaders(func(w http.ResponseWriter, r *http.Request) {
		templates.ExecuteTemplate(w, name, nil)
	})
}

func doRedirect(target string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target, http.StatusSeeOther)
	}
}

func wsHandlerUser(w http.ResponseWriter, r *http.Request) {
	session := getSession(w, r)

	token, good := session.Values["wsToken"].(string)
	if !good || token != mux.Vars(r)["token"] {
		http.Error(w, "bad request", 400)
		return
	}

	x := msgp.WsUserAPI{
		Ctx:     &apiCtx,
		User:    mux.Vars(r)["user"],
		Writer:  w,
		Request: r,
	}
	defer x.Close()
	x.Run()
}

func wsHandlerDevice(w http.ResponseWriter, r *http.Request) {
	x := msgp.WsDevAPI{
		User:    mux.Vars(r)["user"],
		Device:  mux.Vars(r)["device"],
		Writer:  w,
		Request: r,
	}
	if oldAPIPostClient != nil {
		x.Key = []byte(proxyConf.DeviceKeys[x.Device])
		x.PostURL = proxyConf.PostURL
		x.PostClient = oldAPIPostClient
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

	user := r.PostFormValue("user")
	password := r.PostFormValue("password")

	var missing []string
	if user == "" {
		missing = append(missing, "user")
	}
	if password == "" {
		missing = append(missing, "password")
	}
	if missing != nil {
		type ctx struct {
			Missing []string
		}
		templates.ExecuteTemplate(w, "user-login", ctx{missing})
		return
	}

	db.View(func(tx msgpdb.Tx) error {
		user := tx.User(user)
		if user == nil || !user.HasPassword(password) {
			http.Error(w, "bad username/password", 400)
			return nil
		}

		session.Values["user"] = r.PostFormValue("user")
		session.Values["wsToken"] = fmt.Sprintf("%x", sha256.Sum256([]byte(time.Now().String())))
		session.Save(r, w)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return nil
	})
}

func doLogout(w http.ResponseWriter, r *http.Request) {
	session, _ := cookieStore.Get(r, "msgp-session")
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func userRegister(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("user")
	password := r.FormValue("password")

	var ctx struct {
		Missing []string
		Error   string
	}
	if name == "" {
		ctx.Missing = append(ctx.Missing, "user")
	}
	if password == "" {
		ctx.Missing = append(ctx.Missing, "password")
	}
	if ctx.Missing != nil {
		templates.ExecuteTemplate(w, "register", ctx)
		return
	}

	db.Update(func(tx msgpdb.Tx) error {
		_, err := tx.AddUser(name, password)
		if err != nil {
			ctx.Error = err.Error()
			templates.ExecuteTemplate(w, "register", ctx)
			return err
		}
		templates.ExecuteTemplate(w, "register_done", nil)
		return nil
	})
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	tstr := `
<div>Users:</div>
<ul>
{{range $name, $user := (.U.Users)}}
	<li>
		<div>{{$name}} (is admin: {{$user.IsAdmin}})</div>
		<ul>
		{{range $name, $dev := $user.Devices}}
			<li>
				<div>{{$dev.Id}}({{$dev.Name}}) key 0x{{$dev.Key | printf "%x"}}</div>
				<ul>
				{{range $name, $sens := $dev.Sensors}}
					<li>{{$sens.Id}} (name: {{$sens.Name}}) (port: {{$sens.Port}}) (unit: {{$sens.Unit}})</li>
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
	<li>
		{{$id}} 0x{{$link.Key | printf "%x"}} -> {{userOfLink $link}}
		<ul>
			<li>{{configOf $link}}</li>
		</ul>
		{{range $_, $hb := ($link.GetHeartbeats 0)}}
		<ul>
			<li>Heartbeat at {{$hb}}</li>
		</ul>
		{{end}}
	</li>
{{end}}
</ul>
<strong>done</strong>
`

	session := getSession(w, r)
	userID, found := session.Values["user"]
	if !found {
		removeSessionAndNotifyUser(w, r, session)
		return
	}

	db.View(func(utx msgpdb.Tx) error {
		user := utx.User(userID.(string))
		if user == nil || !user.IsAdmin() {
			http.Error(w, "unauthorized", 401)
			return nil
		}

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
				"configOf": func(link regdev.RegisteredDevice) string {
					data, err := json.Marshal(link.GetNetworkConfig())
					if err != nil {
						return err.Error()
					}
					return string(data)
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

func adminUserAdd(w http.ResponseWriter, r *http.Request) {
	user := mux.Vars(r)["user"]
	password := r.FormValue("password")

	err := db.Update(func(tx msgpdb.Tx) error {
		_, err := tx.AddUser(user, password)
		return err
	})
	if err != nil {
		http.Error(w, err.Error(), 400)
	}
}

func adminUserSet(w http.ResponseWriter, r *http.Request) {
	userID := mux.Vars(r)["user"]
	db.Update(func(tx msgpdb.Tx) error {
		user := tx.User(userID)
		if user == nil {
			http.Error(w, "not found", 404)
			return errors.New("")
		}

		switch r.FormValue("isAdmin") {
		case "":
		case "true":
			user.SetAdmin(true)
		case "false":
			user.SetAdmin(false)
		default:
			http.Error(w, "bad isAdmin", 400)
			return errors.New("")
		}

		return nil
	})
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

type apiError struct {
	Code int
	Msg  string
}

func apiBlock(fn func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			switch err := recover().(type) {
			case nil:
				return
			case *apiError:
				http.Error(w, err.Msg, err.Code)
			default:
				panic(err)
			}
		}()

		fn(w, r)
	}
}

func apiAbort(code int, msg string) {
	panic(&apiError{code, msg})
}

func apiAbortIf(code int, err error) {
	if err != nil {
		apiAbort(code, err.Error())
	}
}

func apiSessionUser(tx msgpdb.Tx, s *sessions.Session) msgpdb.User {
	user := tx.User(s.Values["user"].(string))
	if user == nil {
		apiAbort(401, "not authorized")
	}
	return user
}


func apiDevice(tx regdev.Tx, devID string) regdev.RegisteredDevice {
	dev := tx.Device(devID)
	if dev == nil {
		apiAbort(404, "no such device")
	}
	return dev
}

func apiUserDevice(user msgpdb.User, devID string) msgpdb.Device {
	dev := user.Device(devID)
	if dev == nil {
		apiAbort(404, "no such device")
	}
	return dev
}

func apiUserDevicesGet(w http.ResponseWriter, r *http.Request) {
	session := getSession(w, r)
	db.View(func(tx msgpdb.Tx) error {
		user := apiSessionUser(tx, session)

		devs := make(map[string]interface{})
		for id, dev := range user.Devices() {
			sensors := make(map[string]interface{})
			for id, sens := range dev.Sensors() {
				sensors[id] = map[string]interface{}{
					"id":   id,
					"name": sens.Name(),
					"port": sens.Port(),
					"unit": sens.Unit(),
				}
			}
			devs[id] = map[string]interface{}{
				"name":    dev.Name(),
				"sensors": sensors,
			}
		}

		data, err := json.Marshal(devs)
		apiAbortIf(500, err)
		w.Write(data)
		return nil
	})
}

func apiUserDevicesAdd(w http.ResponseWriter, r *http.Request) {
	session := getSession(w, r)
	devID := mux.Vars(r)["device"]

	db.Update(func(utx msgpdb.Tx) error {
		return devdb.Update(func(dtx regdev.Tx) error {
			user := apiSessionUser(utx, session)
			dev := apiDevice(dtx, devID)

			apiAbortIf(400, dev.LinkTo(user.ID()))

			_, err := user.AddDevice(dev.ID(), dev.Key(), false)
			apiAbortIf(500, err)

			data := map[string]interface{}{
				devID: map[string]interface{}{
					"name": user.Device(devID).Name(),
				},
			}
			raw, _ := json.Marshal(data)
			w.Write(raw)
			return nil
		})
	})
}

func apiUserDevicesRemove(w http.ResponseWriter, r *http.Request) {
	session := getSession(w, r)
	devID := mux.Vars(r)["device"]

	db.Update(func(utx msgpdb.Tx) error {
		return devdb.Update(func(dtx regdev.Tx) error {
			user := apiSessionUser(utx, session)
			dev := apiDevice(dtx, devID)

			apiAbortIf(500, dev.Unlink())
			apiAbortIf(500, user.RemoveDevice(devID))
			return nil
		})
	})
}

func apiUserDeviceConfigGet(w http.ResponseWriter, r *http.Request) {
	session := getSession(w, r)
	devID := mux.Vars(r)["device"]
	db.View(func(utx msgpdb.Tx) error {
		return devdb.View(func(dtx regdev.Tx) error {
			user := apiSessionUser(utx, session)
			rdev := apiDevice(dtx, devID)
			dev := apiUserDevice(user, devID)

			netconf := rdev.GetNetworkConfig()
			conf := map[string]interface{}{
				"lan":  netconf.LAN,
				"wifi": netconf.Wifi,
				"name": dev.Name(),
			}
			data, err := json.Marshal(conf)
			apiAbortIf(500, err)
			w.Write(data)
			return nil
		})
	})
}

func apiUserDeviceConfigSet(w http.ResponseWriter, r *http.Request) {
	session := getSession(w, r)
	devID := mux.Vars(r)["device"]
	db.Update(func(utx msgpdb.Tx) error {
		return devdb.Update(func(dtx regdev.Tx) error {
			user := apiSessionUser(utx, session)
			rdev := apiDevice(dtx, devID)
			dev := apiUserDevice(user, devID)

			var conf regdev.DeviceConfigNetwork
			var name struct {
				Name string `json:"name"`
			}

			data, err := ioutil.ReadAll(r.Body)
			apiAbortIf(500, err)

			apiAbortIf(400, json.Unmarshal(data, &conf))
			apiAbortIf(400, json.Unmarshal(data, &name))

			apiAbortIf(400, dev.SetName(name.Name))
			apiAbortIf(400, rdev.SetNetworkConfig(&conf))
			return nil
		})
	})
}

func apiUserDeviceSensorPropsGet(w http.ResponseWriter, r *http.Request) {
	session := getSession(w, r)
	devID := mux.Vars(r)["device"]
	sensID := mux.Vars(r)["sensor"]
	db.View(func(utx msgpdb.Tx) error {
		user := apiSessionUser(utx, session)
		dev := apiUserDevice(user, devID)

		sens := dev.Sensor(sensID)
		if sens == nil {
			apiAbort(404, "no such sensor")
		}

		conf := map[string]interface{}{
			"name":   sens.Name(),
			"unit":   sens.Unit(),
			"port":   sens.Port(),
			"factor": sens.Factor(),
		}

		data, err := json.Marshal(conf)
		apiAbortIf(500, err)
		w.Write(data)
		return nil
	})
}

func apiUserDeviceSensorPropsSet(w http.ResponseWriter, r *http.Request) {
	session := getSession(w, r)
	devID := mux.Vars(r)["device"]
	sensID := mux.Vars(r)["sensor"]
	db.Update(func(utx msgpdb.Tx) error {
		user := apiSessionUser(utx, session)
		dev := apiUserDevice(user, devID)

		sens := dev.Sensor(sensID)
		if sens == nil {
			apiAbort(404, "no such sensor")
		}

		data, err := ioutil.ReadAll(r.Body)
		apiAbortIf(500, err)

		var conf struct {
			Name string
		}

		apiAbortIf(500, json.Unmarshal(data, &conf))
		apiAbortIf(500, sens.SetName(conf.Name))

		apiCtx.Hub.Publish(user.ID(), msg2api.UserEventMetadataArgs{
			Devices: map[string]msg2api.DeviceMetadata{
				devID: {
					Sensors: map[string]msg2api.SensorMetadata{
						sensID: {
							Name: &conf.Name,
						},
					},
				},
			},
		})

		return nil
	})
}

func main() {
	if config.Benchmark.DoBenchmark {
		db.RunBenchmark(config.Benchmark.UserCount, config.Benchmark.DeviceCount, config.Benchmark.SensorCount, config.Benchmark.Duration*time.Minute)
	} else {
		router := mux.NewRouter()
		server := regdev.DeviceServer{Db: devdb}

		router.HandleFunc("/", loggedInSwitch(wsTemplate("index_user"), staticTemplate("index_nouser"))).Methods("GET")
		router.HandleFunc("/user/login", staticTemplate("user-login")).Methods("GET")
		router.HandleFunc("/user/login", doLogin).Methods("POST")
		router.HandleFunc("/user/logout", doLogout).Methods("GET")
		router.HandleFunc("/user/register", staticTemplate("register")).Methods("GET")
		router.HandleFunc("/user/register", defaultHeaders(userRegister)).Methods("POST")
		router.HandleFunc("/user/devices", loggedInSwitch(staticTemplate("user-devices"), doRedirect("/"))).Methods("GET")
		router.HandleFunc("/api/user/v1/devices", apiBlock(apiUserDevicesGet)).Methods("GET")
		router.HandleFunc("/api/user/v1/device/{device}", apiBlock(apiUserDevicesAdd)).Methods("POST")
		router.HandleFunc("/api/user/v1/device/{device}", apiBlock(apiUserDevicesRemove)).Methods("DELETE")
		router.HandleFunc("/api/user/v1/device/{device}/config", apiBlock(apiUserDeviceConfigGet)).Methods("GET")
		router.HandleFunc("/api/user/v1/device/{device}/config", apiBlock(apiUserDeviceConfigSet)).Methods("POST")
		router.HandleFunc("/api/user/v1/sensor/{device}/{sensor}/props", apiBlock(apiUserDeviceSensorPropsGet)).Methods("GET")
		router.HandleFunc("/api/user/v1/sensor/{device}/{sensor}/props", apiBlock(apiUserDeviceSensorPropsSet)).Methods("POST")

		router.HandleFunc("/admin", defaultHeaders(adminHandler))

		if config.EnableAdminOps {
			router.HandleFunc("/admin/user/{user}", adminUserAdd).Methods("PUT")
			router.HandleFunc("/admin/user/{user}/props", adminUserSet).Methods("POST")
		}

		router.HandleFunc("/ws/user/{user}/{token}", wsHandlerUser)
		router.HandleFunc("/ws/device/{user}/{device}", wsHandlerDevice)
		server.RegisterRoutes(router.PathPrefix("/api/regdev").Subrouter())
		router.PathPrefix("/").Handler(http.FileServer(http.Dir(config.AssetsDir)))

		http.Handle("/", router)

		log.Print("Listening on ", config.ListenAddr)
		if config.TLS.Cert != "" {
			log.Printf("Using SSL cert and key %v, %v", config.TLS.Cert, config.TLS.Key)

			if err := http.ListenAndServeTLS(config.ListenAddr, config.TLS.Cert, config.TLS.Key, nil); err != nil {
				log.Fatalf("failed: %v", err.Error())
			}
		} else {
			if err := http.ListenAndServe(config.ListenAddr, nil); err != nil {
				log.Fatalf("failed: %v", err.Error())
			}
		}
	}
}

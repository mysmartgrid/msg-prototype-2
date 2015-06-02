package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"github.com/mysmartgrid/msg-prototype-2/regdev"
	"io/ioutil"
	"log"
	"net/http"
)

var (
	listenDevAddr = flag.String("listen-dev", ":18009", "listener address for devices")
	listenCmdAddr = flag.String("listen-cmd", ":18010", "listener address for management commands")

	dbPath     = flag.String("db", "", "database filename")
	serverCert = flag.String("cert", "", "tls certificate file")
	serverKey  = flag.String("key", "", "tls key file")
	clientCA   = flag.String("client-ca", "", "ca for connecting clients")

	db regdev.Db

	tlsConfig tls.Config
)

func init() {
	flag.Parse()

	bailIfMissing := func(x *string, name string) {
		if *x == "" {
			log.Fatalf("%v missing", name)
		}
	}

	bailIfMissing(listenDevAddr, "-listen-dev")
	bailIfMissing(listenCmdAddr, "-listen-cmd")
	bailIfMissing(dbPath, "-db")
	bailIfMissing(serverCert, "-cert")
	bailIfMissing(serverKey, "-key")
	bailIfMissing(clientCA, "-client-ca")

	var err error

	db, err = regdev.Open(*dbPath)
	if err != nil {
		log.Fatalf("error opening db: %v", err.Error())
	}

	clientCAPEM, err := ioutil.ReadFile(*clientCA)
	if err != nil {
		log.Fatalf("error loading client CA: %v", err.Error())
	}

	clientPool := x509.NewCertPool()
	clientPool.AppendCertsFromPEM(clientCAPEM)

	tlsConfig = tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  clientPool,
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
		PreferServerCipherSuites: true,
	}
}

func deviceListener(err chan<- error) {
	// TODO: run regdev.DeviceServer
}

func commandListener(err chan<- error) {
	server := http.Server{
		Addr:      *listenCmdAddr,
		TLSConfig: &tlsConfig,
	}
	// TODO: insert a regdev.CommandServer
	err <- server.ListenAndServeTLS(*serverCert, *serverKey)
}

func main() {
	dev, cmd := make(chan error), make(chan error)
	go deviceListener(dev)
	go commandListener(cmd)
	select {
	case err := <-dev:
		log.Fatalf("error in device listener: %v", err.Error())

	case err := <-cmd:
		log.Fatalf("error in command listener: %v", err.Error())
	}
}

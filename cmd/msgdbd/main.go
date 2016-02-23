package main

import (
	"database/sql"
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	_ "github.com/lib/pq"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type postgresConfig struct {
	User     string `toml:"user"`
	Password string `toml:"password"`
	Address  string `toml:"address"`
	Port     string `toml:"port"`
	Database string `toml:"database"`
}

type daemonConfig struct {
	AggregationInterval time.Duration  `toml:"aggregationinterval"`
	CleanupInterval     time.Duration  `toml:"cleanupinterval"`
	DbCOnfig            postgresConfig `toml:"postgres"`
}

var configFile = flag.String("config", "", "configuration file")
var verbose = flag.Bool("v", false, "verbose output")
var config daemonConfig
var db *sql.DB

type jobStatus struct {
	Err      error
	Duration time.Duration
}

type job struct {
	Name     string
	RunJob   func(chan jobStatus)
	Interval time.Duration
}

func (j *job) Run() {
	ticker := time.NewTicker(j.Interval)
	status := make(chan jobStatus)
	defer func() {
		ticker.Stop()
		close(status)
	}()

	handleStatus := func(s jobStatus) {
		if s.Err != nil {
			log.Panicf("Error during job '%s': %s", j.Name, s.Err)
		}
		if *verbose {
			log.Printf("Job '%s' took %s", j.Name, s.Duration)
		}
		if s.Duration > config.AggregationInterval {
			log.Printf("WARNING: Job '%s' took longer than interval (%s)!", j.Name, s.Duration)
		}
	}

	for {
		if *verbose {
			log.Printf("Running job '%s'...", j.Name)
		}
		go j.RunJob(status)

		select {
		case <-ticker.C:
			s := <-status
			handleStatus(s)
		case s := <-status:
			handleStatus(s)
			<-ticker.C
		}
	}
}

func newDbFunc(query string) func(chan jobStatus) {
	return func(state chan jobStatus) {
		start := time.Now()

		tx, err := db.Begin()
		if err != nil {
			state <- jobStatus{err, time.Since(start)}
			return
		}

		defer func() {
			if err != nil {
				tx.Rollback()
			}
		}()

		_, err = tx.Exec(query)
		if err != nil {
			_ = tx.Rollback()
			state <- jobStatus{err, time.Since(start)}
			return
		}

		state <- jobStatus{tx.Commit(), time.Since(start)}
		return
	}
}

func openDb(sqlAddr, sqlPort, sqlDb, sqlUser, sqlPass string) (*sql.DB, error) {
	cfg := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=disable",
		sqlUser,
		sqlPass,
		sqlDb,
		sqlAddr,
		sqlPort,
	)

	postgres, err := sql.Open("postgres", cfg)
	if err != nil {
		return nil, err
	}

	return postgres, nil
}

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

	if config.DbCOnfig.User == "" || config.DbCOnfig.Address == "" || config.DbCOnfig.Database == "" {
		log.Fatal("postgres config incomplete")
	}

	config.AggregationInterval = config.AggregationInterval * time.Minute
	config.CleanupInterval = config.CleanupInterval * time.Minute

	db, err = openDb(config.DbCOnfig.Address, config.DbCOnfig.Port, config.DbCOnfig.Database,
		config.DbCOnfig.User, config.DbCOnfig.Password)
	if err != nil {
		log.Fatal("error opening user db: ", err)
	}
}

func main() {
	defer db.Close()

	if config.AggregationInterval != 0 {
		aggrJob := job{Name: "Aggregation", RunJob: newDbFunc(`SELECT do_aggregate()`), Interval: config.AggregationInterval}
		go aggrJob.Run()
	} else {
		if *verbose {
			log.Println("Aggregation interval not set or 0, not starting aggregation job.")
		}
	}
	if config.CleanupInterval != 0 {
		aggrJob := job{Name: "Cleanup", RunJob: newDbFunc(`SELECT do_remove_old_values()`), Interval: config.CleanupInterval}
		go aggrJob.Run()
	} else {
		if *verbose {
			log.Println("Cleanup interval not set or 0, not starting cleanup job.")
		}
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Println("Exiting...")
}

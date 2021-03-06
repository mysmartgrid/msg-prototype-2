package db

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/mysmartgrid/msg2api"
	"log"
	"time"
)

const (
	bufferSize = 100000
)

var (
	// ErrIDExists is returned after an attempt to insert a new object into the DB using an id which already exists in the DB.
	ErrIDExists = errors.New("id exists")
)

type db struct {
	sqldb sqlHandler

	bufferedValues     map[uint64][]msg2api.Measurement
	bufferedValueCount uint32

	bufferInput chan bufferValue
	bufferAdd   chan uint64
	bufferKill  chan uint64
}

type bufferValue struct {
	key   uint64
	value msg2api.Measurement
}

func (db *db) flushBuffer() {
	if db.bufferedValueCount == 0 {
		return
	}

	err := db.sqldb.saveValuesAndClear(db.bufferedValues)
	if err != nil {
		panic(err.Error())
	}

	db.bufferedValueCount = 0
}

func (db *db) manageBuffer() {
	ticker := time.NewTicker(1 * time.Second)
	defer func() {
		ticker.Stop()
		db.flushBuffer()
	}()

	for {
		select {

		// Handle new measurement inputs to the buffer
		case bval, ok := <-db.bufferInput:
			if !ok {
				return
			}
			slice, found := db.bufferedValues[bval.key]
			if !found {
				log.Printf("adding value to bad key %v", bval.key)
				continue
			}
			db.bufferedValues[bval.key] = append(slice, bval.value)
			db.bufferedValueCount++

			// Flush full buffer to database
			if db.bufferedValueCount >= bufferSize {
				db.flushBuffer()
			}

		// Remove a sensor from buffer management, dropping any buffered values
		case key := <-db.bufferKill:
			delete(db.bufferedValues, key)

		// Add a new sensor to the buffer management
		case key := <-db.bufferAdd:
			db.bufferedValues[key] = make([]msg2api.Measurement, 0, 4)

		// Periodically flush buffer
		case <-ticker.C:
			db.flushBuffer()
		}
	}
}

// OpenDb opens a connection to the postgres database with the given parameters,
// starts a process to manage its value buffer and adds all sensors in the database to the buffer manager.
// Returns a Db struct on success or an error otherwise
func OpenDb(sqlAddr, sqlPort, sqlDb, sqlUser, sqlPass string) (Db, error) {
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

	result := &db{
		sqldb:          sqlHandler{postgres},
		bufferedValues: make(map[uint64][]msg2api.Measurement),
		bufferInput:    make(chan bufferValue),
		bufferKill:     make(chan uint64),
		bufferAdd:      make(chan uint64),
	}

	go result.manageBuffer()

	rows, err := result.sqldb.db.Query(`SELECT sensor_seq FROM sensors`)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var seq uint64
		err = rows.Scan(&seq)
		if err != nil {
			return nil, err
		}
		result.bufferAdd <- seq
	}
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (db *db) Close() {
	close(db.bufferInput)
	db.sqldb.db.Close()
}

func (db *db) View(fn func(Tx) error) error {
	t, err := db.sqldb.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			t.Rollback()
		}
	}()

	err = fn(&tx{db, t})

	if err != nil {
		_ = t.Rollback()
		return err
	}
	if err := t.Rollback(); err != nil {
		return err
	}
	return nil
}

func (db *db) Update(fn func(Tx) error) error {
	t, err := db.sqldb.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			t.Rollback()
		}
	}()

	err = fn(&tx{db, t})

	if err != nil {
		_ = t.Rollback()
		return err
	}
	return t.Commit()
}

func (db *db) AddReading(sensor Sensor, time time.Time, value float64) error {
	db.bufferInput <- bufferValue{sensor.DbID(), msg2api.Measurement{time, value}}
	return nil
}

package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
)

const (
	bufferSize = 100000
)

var (
	InvalidId = errors.New("id invalid")
	IdExists  = errors.New("id exists")

	noSensor            = errors.New("sensor not found")
	deviceNotRegistered = errors.New("device not registered")
)

type db struct {
	sqldb sqlHandler

	bufferedValues     map[uint64][]Value
	bufferedValueCount uint32

	bufferInput chan bufferValue
	bufferAdd   chan uint64
	bufferKill  chan uint64
}

type bufferValue struct {
	key   uint64
	value Value
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

			if db.bufferedValueCount >= bufferSize {
				db.flushBuffer()
			}

		case key := <-db.bufferKill:
			delete(db.bufferedValues, key)

		case key := <-db.bufferAdd:
			db.bufferedValues[key] = make([]Value, 0, 4)

		case <-ticker.C:
			db.flushBuffer()
		}
	}
}

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
		bufferedValues: make(map[uint64][]Value),
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
	db.bufferInput <- bufferValue{sensor.DbId(), Value{time, value}}
	return nil
}

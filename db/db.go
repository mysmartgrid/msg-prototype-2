package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
)

const (
	bufferSize = 10000
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

	realtimeSensors      map[string]map[uint64]*realtimeEntry
	realtimeHandler      func(values map[string]map[string]map[string]map[string][]Value)
	realtimeSensorsInput chan map[string][]Sensor
}

type bufferValue struct {
	key   uint64
	value Value
}

type realtimeEntry struct {
	sensor                  Sensor
	lastRequest, lastUpdate time.Time
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

func (d *db) manageBuffer() {
	ticker := time.NewTicker(1 * time.Second)
	defer func() {
		ticker.Stop()
		d.flushBuffer()
	}()

	for {
		select {
		case bval, ok := <-d.bufferInput:
			if !ok {
				return
			}
			slice, found := d.bufferedValues[bval.key]
			if !found {
				log.Printf("adding value to bad key %v", bval.key)
				continue
			}
			d.bufferedValues[bval.key] = append(slice, bval.value)
			d.bufferedValueCount++

			if d.bufferedValueCount >= bufferSize {
				d.flushBuffer()
			}

		case key := <-d.bufferKill:
			delete(d.bufferedValues, key)

		case key := <-d.bufferAdd:
			d.bufferedValues[key] = make([]Value, 0, 4)

		case <-ticker.C:
			d.flushBuffer()
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

		realtimeSensors:      make(map[string]map[uint64]*realtimeEntry),
		realtimeSensorsInput: make(chan map[string][]Sensor),
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

func (d *db) Close() {
	close(d.bufferInput)
	d.sqldb.db.Close()
}

func (d *db) View(fn func(Tx) error) error {
	t, err := d.sqldb.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			t.Rollback()
		}
	}()

	err = fn(&tx{d, t})

	if err != nil {
		_ = t.Rollback()
		return err
	}
	if err := t.Rollback(); err != nil {
		return err
	}
	return nil
}

func (d *db) Update(fn func(Tx) error) error {
	t, err := d.sqldb.db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			t.Rollback()
		}
	}()

	err = fn(&tx{d, t})

	if err != nil {
		_ = t.Rollback()
		return err
	}
	return t.Commit()
}

func (d *db) AddReading(sensor Sensor, time time.Time, value float64) error {
	d.bufferInput <- bufferValue{sensor.DbId(), Value{time, value}}
	return nil
}

func (d *db) SetRealtimeHandler(handler func(values map[string]map[string]map[string]map[string][]Value)) {
	d.realtimeHandler = handler
}

func (d *db) manageRealtimeSensors() {
	lifetimeTicker := time.NewTicker(30 * time.Second)
	updateTicker := time.NewTicker(1 * time.Minute)
	defer func() {
		lifetimeTicker.Stop()
		updateTicker.Stop()
	}()

	for {
		select {
		case sensors, ok := <-d.realtimeSensorsInput:
			if !ok {
				return
			}
			for resolution, senses := range sensors {
				if _, ok := d.realtimeSensors[resolution]; !ok {
					d.realtimeSensors[resolution] = make(map[uint64]*realtimeEntry)
				}
				for _, sens := range senses {
					if _, ok := d.realtimeSensors[resolution][sens.DbId()]; !ok {
						d.realtimeSensors[resolution][sens.DbId()] = &realtimeEntry{sens, time.Now(), time.Now()}
					} else {
						d.realtimeSensors[resolution][sens.DbId()].lastRequest = time.Now()
					}
				}
			}

		case <-lifetimeTicker.C:
			for _, sensorids := range d.realtimeSensors {
				for sensorid, entry := range sensorids {
					if entry.lastRequest.Add(time.Second * 40).Before(time.Now()) {
						delete(sensorids, sensorid)
					}
				}
			}

		case <-updateTicker.C:
			d.doRealtimeUpdates()
		}
	}
}

func (d *db) doRealtimeUpdates() {
	result := make(map[string]map[string]map[string]map[string][]Value)
	for resolution, sensorids := range d.realtimeSensors {
		for sensorid, entry := range sensorids {
			values, err := d.sqldb.loadValuesSingle(entry.lastUpdate, time.Now(), resolution, sensorid)
			if err == nil {
				entry.lastUpdate = time.Now()
				if len(values) > 0 {
					user := entry.sensor.Device().User().Id()
					device := entry.sensor.Device().Id()
					if _, ok := result[user]; !ok {
						result[user] = make(map[string]map[string]map[string][]Value)
					}
					if _, ok := result[user][device]; !ok {
						result[user][device] = make(map[string]map[string][]Value)
					}
					if _, ok := result[user][device][resolution]; !ok {
						result[user][device][resolution] = make(map[string][]Value)
					}
					result[user][device][resolution][entry.sensor.Id()] = values
				}
			}
		}
	}
	d.realtimeHandler(result)
}

func (d *db) RequestRealtimeUpdates(user, device, resolution string, sensors []string) error {
	senses := make([]Sensor, len(sensors))
	err := d.View(func(tx Tx) error {
		usr := tx.User(user)
		if usr == nil {
			return deviceNotRegistered
		}
		dev := usr.Device(device)
		if dev == nil {
			return deviceNotRegistered
		}
		for idx, sensor := range sensors {
			sens := dev.Sensor(sensor)
			if sens == nil {
				return noSensor
			}
			senses[idx] = sens
		}
		return nil
	})

	if err != nil {
		return err
	}

	result := make(map[string][]Sensor)
	result[resolution] = senses
	d.realtimeSensorsInput <- result

	return err
}

func (d *db) Run() {
	go d.manageRealtimeSensors()
}

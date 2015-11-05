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
)

type db struct {
	sqldb sqlHandler

	bufferedValues     map[uint64][]Value
	bufferedValueCount uint32

	bufferInput chan bufferValue
	bufferAdd   chan uint64
	bufferKill  chan uint64

	realtimeSensors map[Device]map[string]map[Sensor]realtimeEntry
	realtimeHandler func(values map[Device]map[string]map[Sensor][]Value)
}

type bufferValue struct {
	key   uint64
	value Value
}

type realtimeEntry struct {
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

		realtimeSensors: make(map[Device]map[string]map[Sensor]realtimeEntry),
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

func (d *db) SetRealtimeHandler(handler func(values map[Device]map[string]map[Sensor][]Value)) {
	d.realtimeHandler = handler
}

func (d *db) RequestRealtimeUpdates(sensors map[Device]map[string][]Sensor) {
	for device, resolutions := range sensors {
		for resolution, sens := range resolutions {
			for _, sensor := range sens {
				entry, ok := d.realtimeSensors[device][resolution][sensor]
				if !ok {
					d.realtimeSensors[device][resolution][sensor] = realtimeEntry{time.Now(), time.Now()}
				} else {
					entry.lastRequest = time.Now()
				}
			}
		}
	}
}

func (d *db) doRealtimeUpdates(resolution string) {
	var interval time.Duration

	switch resolution {
	case "second":
		interval = time.Second
	case "minute":
		interval = time.Minute
	case "hour":
		interval = time.Hour
	case "day", "week", "month", "year":
		interval = time.Hour * 24
	default:
		return
	}

	for {
		result := make(map[Device]map[string]map[Sensor][]Value)

		for device, resolutions := range d.realtimeSensors {
			for sensor, entry := range resolutions[resolution] {
				values, err := d.sqldb.loadValuesSingle(entry.lastUpdate, time.Now(), resolution, sensor.DbId())
				if err == nil {
					result[device][resolution][sensor] = values
					entry.lastUpdate = time.Now()
				}
			}
		}
		d.realtimeHandler(result)
		time.Sleep(interval)
	}
}

func (d *db) Run() {
	go func() {
		for _, resolutions := range d.realtimeSensors {
			for _, sensors := range resolutions {
				for sensor, entry := range sensors {
					if !entry.lastRequest.Add(time.Second * 30).After(time.Now()) {
						delete(sensors, sensor)
					}
				}
			}
		}
	}()

	resolutions := [...]string{"second", "minute", "hour", "day", "week", "month", "year"}
	for _, res := range resolutions {
		go d.doRealtimeUpdates(res)
	}

}

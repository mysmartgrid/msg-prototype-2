package oldapidb


import (
	"errors"
	"time"
//	"fmt"
	"log"
)

var (
	// ErrIDExists is returned after an attempt to insert a new object into the DB using an id which already exists in the DB.
	ErrIDExists = errors.New("id exists")
	// ErrAlreadyLinked is returned when trying to link a user to a device already associated with a user.
//	ErrAlreadyLinked = errors.New("already linked")

	dbRegisteredSensors = []byte("registeredSensors")
)

type dbdata struct {
	timestamp int64
	value     int
}

type db struct {
	values map[string](dbdata)
}


func Open() (Db, error) {
	m := make(map[string](dbdata))
	result := &db{
		values: m,
	}

	return result, nil
}

func (d *db) View(fn func(Tx) error) error {
	log.Printf("    Get db.view():")
	
	return fn(&tx{d})
	//return nil
}

func (d *db) Update(fn func(Tx) error) error {
	log.Printf("    Get db.update():")
	return fn(&tx{d})
        return nil
}

func (db *db) AddLastValue(sensorId string, time time.Time, value float64) error {
	//log.Printf("    db.AddLastValue(): %s", sensorId)
	//fmt.Println("DB add value-pair")
	data := dbdata{
		timestamp: time.Unix(),
		value: int(value)}
	db.values[sensorId] = data
        return nil
}

func (db *db) GetLastValue(sensorId string) (int64, int, error) {
        data, ok := db.values[sensorId]
        if ok {
	        return data.timestamp, data.value, nil

        //} else {
        //        fmt.Println("key not found")
        }
	
        return 0, 0, ErrIDExists
}

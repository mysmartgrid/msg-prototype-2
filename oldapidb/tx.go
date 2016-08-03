package oldapidb

import (
	"time"
	"encoding/binary"
	"math"
)

type tx struct {
	db *db
}

func Float64frombytes(bytes []byte) float64 {
    bits := binary.LittleEndian.Uint64(bytes)
    float := math.Float64frombits(bits)
    return float
}

func Float64bytes(float float64) []byte {
    bits := math.Float64bits(float)
    bytes := make([]byte, 8)
    binary.LittleEndian.PutUint64(bytes, bits)
    return bytes
}

func (tx *tx) Sensor(sensorID string) RegisteredSensor {
	return nil
}

func (tx *tx) AddLastValue(sensorID string, Time time.Time, value float64) error {
	//log.Printf("AddLast....")
        //log.Printf("   Search sensor: %s", sensorID)
	err := tx.db.AddLastValue(sensorID, Time, value)
	if err != nil {
		return err
	}
	//log.Printf("   Wrote value: %f", value)
	
	return nil
}
func (tx *tx) GetLastValue(sensorID string) (time.Time, float64, error) {
	//lvalue := 0.0
	var Time time.Time
        //log.Printf("   Search sensor: %s", sensorID)

	ts, val, err := tx.db.GetLastValue(sensorID)
	if err != nil {
		return time.Unix(0,0),0.0,ErrIDExists
	}
	Time = time.Unix(ts, 0)

	//log.Printf("   Return last value pair (%d, %f)",Time.Unix(), lvalue)
	return Time, float64(val), nil
}

func (tx *tx) Sensors() map[string]RegisteredSensor {
	result := make(map[string]RegisteredSensor)
	return result
}

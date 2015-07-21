package db

import (
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"time"
)

func seriesID(key bufferKey) string {
	return fmt.Sprintf(`%d.%d.%d`, key.user, key.device, key.sensor)
}

type sqlHandler struct {
	db *sql.DB
}

func (h *sqlHandler) saveValuesAndClear(valueMap map[bufferKey][]Value) error {
	tx, err := h.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(pq.CopyIn("measure_raw", "SensorID", "Timestamp", "Value"))
	if err != nil {
		return err
	}

	for key, values := range valueMap {
		//TODO Not here
		_, err := h.db.Exec(
			`INSERT INTO "sensors" SELECT $1
				WHERE NOT EXISTS (
					SELECT * FROM "sensors" WHERE "SensorID" = $2
				)`,
			seriesID(key), seriesID(key))
		if err != nil {
			return err
		}

		for _, value := range values {
			_, err := stmt.Exec(seriesID(key), value.Time, value.Value)
			if err != nil {
				return err
			}
		}
	}

	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	err = stmt.Close()
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	for id, _ := range valueMap {
		valueMap[id] = make([]Value, 0, 1)
	}
	return nil
}

func (h *sqlHandler) loadValuesIn(since, until time.Time, keys []bufferKey) (map[bufferKey][]Value, error) {
	result := make(map[bufferKey][]Value)

	for _, key := range keys {
		rows, err := h.db.Query(`SELECT "Timestamp", "Value" FROM "measure_raw" WHERE "SensorID" = $1 AND "Timestamp" BETWEEN $2 AND $3`,
			seriesID(key), since, until)
		if err != nil {
			return nil, err
		}

		var values []Value
		for rows.Next() {
			var timestamp time.Time
			var value float64

			err = rows.Scan(&timestamp, &value)
			if err != nil {
				return nil, err
			}

			values = append(values, Value{timestamp, value})
		}

		rows.Close()
		err = rows.Err()
		if err != nil {
			return nil, err
		}

		result[key] = values
	}

	return result, nil
}

func (h *sqlHandler) loadValues(since time.Time, keys []bufferKey) (map[bufferKey][]Value, error) {
	return h.loadValuesIn(since, time.Now(), keys)
}

func (h *sqlHandler) removeSeriesFor(user, device, sensor uint64) error {
	sensorid := fmt.Sprintf(`%d.%d.%d`, user, device, sensor)
	_, err := h.db.Exec(`DELETE FROM "sensors" WHERE "SensorID" = $1`, sensorid)

	return err
}

package db

import (
	"database/sql"
	"github.com/lib/pq"
	"time"
)

type sqlHandler struct {
	db *sql.DB
}

func (h *sqlHandler) saveValuesAndClear(valueMap map[uint64][]Value) error {
	tx, err := h.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(pq.CopyIn("measure_raw", "sensor", "timestamp", "value"))
	if err != nil {
		return err
	}

	for seq, values := range valueMap {
		for _, value := range values {
			_, err := stmt.Exec(seq, value.Time, value.Value)
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

func (h *sqlHandler) loadValuesIn(since, until time.Time, sensor_seqs []uint64) (map[uint64][]Value, error) {
	result := make(map[uint64][]Value)

	for _, seq := range sensor_seqs {
		rows, err := h.db.Query(`SELECT "timestamp", "value" FROM "measure_raw" WHERE "sensor" = $1 AND "timestamp" BETWEEN $2 AND $3`,
			seq, since, until)
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

		result[seq] = values
	}

	return result, nil
}

func (h *sqlHandler) loadValues(since time.Time, sensor_seqs []uint64) (map[uint64][]Value, error) {
	return h.loadValuesIn(since, time.Now(), sensor_seqs)
}

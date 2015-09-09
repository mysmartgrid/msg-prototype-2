package db

import (
	"database/sql"
	"errors"
	"fmt"
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

func (h *sqlHandler) loadValues(since, until time.Time, res TimeRes, sensor_seqs []uint64) (map[uint64][]Value, error) {
	result := make(map[uint64][]Value)
	var tableName string

	switch res {
	// case TimeResSecond:
	// 	tableName = "measure_abg_1s"
	case TimeResMinute:
		tableName = "measure_avg_1m"
	case TimeResHour:
		tableName = "measure_avg_1h"
	case TimeResDay:
		tableName = "measure_avg_1d"
	case TimeResWeek:
		tableName = "measure_avg_1w"
	case TimeResMonth:
		tableName = "measure_avg_1mo"
	case TimeResYear:
		tableName = "measure_avg_1y"
	default:
		return result, errors.New("Time resolution not supported.")
	}

	valueQuery := fmt.Sprintf(`SELECT "timestamp", "sum", "count" FROM "%v" WHERE "sensor" = $1 AND "timestamp" BETWEEN $2 AND $3`, tableName)

	for _, seq := range sensor_seqs {
		rows, err := h.db.Query(valueQuery, seq, since, until)
		if err != nil {
			return nil, err
		}

		var values []Value
		for rows.Next() {
			var timestamp time.Time
			var sum float64
			var count int64

			err = rows.Scan(&timestamp, &sum, &count)
			if err != nil {
				return nil, err
			}

			values = append(values, Value{timestamp, sum / float64(count)})
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

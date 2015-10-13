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

var timeResTable map[TimeRes]string = map[TimeRes]string{
	TimeResSecond: "measure_aggregated_seconds",
	TimeResMinute: "measure_aggregated_minutes",
	TimeResHour:   "measure_aggregated_hours",
	TimeResDay:    "measure_aggregated_days",
	TimeResWeek:   "measure_aggregated_weeks",
	TimeResMonth:  "measure_aggregated_months",
	TimeResYear:   "measure_aggregated_years",
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

func (h *sqlHandler) loadValuesSingle(since, until time.Time, res TimeRes, sensor_seq uint64) ([]Value, error) {
	valueQuery := fmt.Sprintf(`SELECT "timestamp", "sum", "count" FROM "%v" WHERE "sensor" = $1 AND "timestamp" BETWEEN $2 AND $3`, timeResTable[res])
	rows, err := h.db.Query(valueQuery, sensor_seq, since, until)
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

	return values, nil
}

func (h *sqlHandler) loadValues(since, until time.Time, res TimeRes, sensor_seqs []uint64) (map[uint64][]Value, error) {
	result := make(map[uint64][]Value)

	_, ok := timeResTable[res]
	if !ok {
		return result, errors.New("Time resolution not supported.")
	}

	var err error
	for _, seq := range sensor_seqs {
		result[seq], err = h.loadValuesSingle(since, until, res, seq)
		if err != nil {
			return result, err
		}
	}

	return result, nil
}

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

type timeRes int

const (
	timeResSecond = iota
	timeResMinute
	timeResHour
	timeResDay
	timeResWeek
	timeResMonth
	timeResYear
)

var timeResMap map[string]timeRes = map[string]timeRes{
	"second": timeResSecond,
	"minute": timeResMinute,
	"hour":   timeResHour,
	"day":    timeResDay,
	"week":   timeResWeek,
	"month":  timeResMonth,
	"year":   timeResYear,
}

var timeResTable map[timeRes]string = map[timeRes]string{
	timeResSecond: "measure_aggregated_seconds",
	timeResMinute: "measure_aggregated_minutes",
	timeResHour:   "measure_aggregated_hours",
	timeResDay:    "measure_aggregated_days",
	timeResWeek:   "measure_aggregated_weeks",
	timeResMonth:  "measure_aggregated_months",
	timeResYear:   "measure_aggregated_years",
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

func (h *sqlHandler) loadValuesSingle(since, until time.Time, resolution string, sensor_seq uint64) ([]Value, error) {
	res, ok := timeResMap[resolution]
	if !ok {
		return nil, errors.New("Time resolution not supported.")
	}

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

func (h *sqlHandler) loadValues(since, until time.Time, resolution string, sensor_seqs []uint64) (map[uint64][]Value, error) {
	result := make(map[uint64][]Value)

	var err error
	for _, seq := range sensor_seqs {
		result[seq], err = h.loadValuesSingle(since, until, resolution, seq)
		if err != nil {
			return result, err
		}
	}

	return result, nil
}

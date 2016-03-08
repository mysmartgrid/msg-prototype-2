package db

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"github.com/mysmartgrid/msg2api"
	"strconv"
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

// saveValuesAndClear write a set of measurements from different sensors to the database and empty the valueMap
func (h *sqlHandler) saveValuesAndClear(valueMap map[uint64][]msg2api.Measurement) error {
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

	for id := range valueMap {
		valueMap[id] = make([]msg2api.Measurement, 0, 1)
	}
	return nil
}

// loadValues loads measurements for a set of sensors in a single timespan and for a single resolution
func (h *sqlHandler) loadValues(since, until time.Time, resolution string, sensorSeqs []uint64) (map[uint64][]msg2api.Measurement, error) {
	if len(sensorSeqs) < 1 {
		return make(map[uint64][]msg2api.Measurement), nil
	}

	var valueQuery string
	var sensorSeqsList bytes.Buffer
	for idx, seq := range sensorSeqs {
		if idx != 0 {
			sensorSeqsList.WriteString(", ")
		}
		sensorSeqsList.WriteString(strconv.FormatUint(seq, 10))
	}

	if resolution == "raw" {
		valueQuery = fmt.Sprintf(`SELECT "sensor", "timestamp", "value" FROM "measure_raw" WHERE "sensor" IN (%v) AND "timestamp" BETWEEN $1 AND $2`, sensorSeqsList.String())
	} else {
		res, ok := timeResMap[resolution]
		if !ok {
			return nil, errors.New("Time resolution not supported.")
		}
		valueQuery = fmt.Sprintf(`SELECT "sensor", "timestamp", "sum", "count" FROM "%v" WHERE "sensor" IN (%v) AND "timestamp" BETWEEN $1 AND $2`, timeResTable[res], sensorSeqsList.String())
	}

	rows, err := h.db.Query(valueQuery, since, until)
	if err != nil {
		return nil, err
	}

	result := make(map[uint64][]msg2api.Measurement)
	if resolution == "raw" {
		for rows.Next() {
			var sensorid uint64
			var timestamp time.Time
			var value float64

			err = rows.Scan(&sensorid, &timestamp, &value)
			if err != nil {
				return nil, err
			}
			result[sensorid] = append(result[sensorid], msg2api.Measurement{timestamp, value})
		}
	} else {
		for rows.Next() {
			var sensorid uint64
			var timestamp time.Time
			var sum float64
			var count int64

			err = rows.Scan(&sensorid, &timestamp, &sum, &count)
			if err != nil {
				return nil, err
			}
			result[sensorid] = append(result[sensorid], msg2api.Measurement{timestamp, sum / float64(count)})
		}
	}

	rows.Close()
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return result, nil
}

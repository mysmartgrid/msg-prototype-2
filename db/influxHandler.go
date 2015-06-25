package db

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/influxdb/influxdb/client"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var badSeriesName = errors.New("bad series name")

const seriesNameTemplate = `user.%d.device.%d.sensor.%d.unit.x%x.sensor_data`
const listSeriesTemplate = `list series /^user\.%d\.device\.%d\.sensor\.%d\./`

func seriesName(key bufferKey) string {
	return fmt.Sprintf(seriesNameTemplate, key.user, key.device, key.sensor, key.unit)
}

func parseSeriesName(name string) (user, device, sensor uint64, unit string, err error) {
	parts := strings.Split(name, ".")
	if len(parts) != 9 || parts[0] != "user" || parts[2] != "device" || parts[4] != "sensor" ||
		parts[6] != "unit" || len(parts[7]) < 3 || parts[7][0] != 'x' || parts[8] != "sensor_data" {
		err = badSeriesName
		return
	}
	if user, err = strconv.ParseUint(parts[1], 10, 64); err != nil {
		err = badSeriesName
		return
	}
	if device, err = strconv.ParseUint(parts[3], 10, 64); err != nil {
		err = badSeriesName
		return
	}
	if sensor, err = strconv.ParseUint(parts[5], 10, 64); err != nil {
		err = badSeriesName
		return
	}
	dec, err := hex.DecodeString(parts[7][1:])
	if err != nil {
		err = badSeriesName
		return
	}
	unit = string(dec)
	return
}

type influxHandler struct {
	client *client.Client
}

func influxTime(t time.Time) int64 {
	return t.Unix()*1000 + int64(t.Nanosecond()/1e6)
}

func goTime(t float64) time.Time {
	return time.Unix(int64(t/1000), int64(t)%1000*1e6)
}

func (h *influxHandler) saveValuesAndClear(valueMap map[bufferKey][]Value) error {
	series := make([]*client.Series, 0, len(valueMap))

	for key, values := range valueMap {
		item := &client.Series{
			Name:    seriesName(key),
			Columns: []string{"time", "value"},
			Points:  make([][]interface{}, len(values)),
		}
		for i, value := range values {
			item.Points[i] = []interface{}{influxTime(value.Time), value.Value}
		}
		series = append(series, item)
	}

	if err := h.client.WriteSeriesWithTimePrecision(series, client.Millisecond); err != nil {
		return err
	}

	for id, _ := range valueMap {
		valueMap[id] = make([]Value, 0, 1)
	}
	return nil
}

func (h *influxHandler) loadValues(since time.Time, keys []bufferKey) (map[bufferKey][]Value, error) {
	series := make([]string, 0, len(keys))
	for _, key := range keys {
		series = append(series, regexp.QuoteMeta(seriesName(key)))
	}
	query := fmt.Sprintf("select time, value from /^(%v)$/ where time > %vms", strings.Join(series, "|"), influxTime(since))

	data, err := h.client.Query(query, client.Millisecond)
	if err != nil {
		return nil, err
	}

	result := make(map[bufferKey][]Value, len(data))
	for _, series := range data {
		if series.Columns[0] != "time" || series.Columns[2] != "value" {
			panic(fmt.Sprintf("wrong field order in %v", series))
		}

		uid, did, sid, unit, err := parseSeriesName(series.Name)
		if err != nil {
			return nil, err
		}

		key := bufferKey{uid, did, sid, unit}
		values := make([]Value, 0, len(series.Points))
		for _, point := range series.Points {
			values = append(values, Value{goTime(point[0].(float64)), point[2].(float64)})
		}
		result[key] = values
	}

	return result, nil
}

func (h *influxHandler) removeSeriesFor(user, device, sensor uint64) error {
	listQ := fmt.Sprintf(listSeriesTemplate, user, device, sensor)
	list, err := h.client.Query(listQ)
	if err != nil {
		return err
	}
	for _, series := range list {
		if len(series.Points) == 0 {
			continue
		}
		if series.Columns[0] != "time" || series.Columns[1] != "name" {
			panic(fmt.Sprintf("wrong field order in %v", series))
		}
		for _, point := range series.Points {
			_, err := h.client.Query(fmt.Sprintf(`drop series "%v"`, regexp.QuoteMeta(point[1].(string))))
			if err != nil {
				panic(err)
			}
		}
	}
	return err
}

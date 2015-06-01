package db

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"github.com/influxdb/influxdb/client"
)

var badSeriesName = errors.New("bad series name")

const seriesNameTemplate = "data-u%d-d%d-s%d"

func seriesName(user, device, sensor uint64) string {
	return fmt.Sprintf(seriesNameTemplate, user, device, sensor)
}

func parseSeriesName(name string) (user, device, sensor uint64, err error) {
	scanned, err := fmt.Sscanf(name, seriesNameTemplate, &user, &device, &sensor)
	if scanned != 3 {
		err = badSeriesName
	}
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
			Name: seriesName(key.user, key.device, key.sensor),
			Columns: []string{"time", "value"},
			Points: make([][]interface{}, len(values)),
		}
		for i, value := range values {
			item.Points[i] = []interface{}{influxTime(value.Time), value.Value}
		}
		series = append(series, item)
	}

	return h.client.WriteSeriesWithTimePrecision(series, client.Millisecond)
}

func (h *influxHandler) loadValues(since time.Time, keys []bufferKey) (map[bufferKey][]Value, error) {
	series := make([]string, 0, len(keys))
	for _, key := range keys {
		series = append(series, seriesName(key.user, key.device, key.sensor))
	}
	query := fmt.Sprintf("select time, value from /%v/ where time > %vms", strings.Join(series, "|"), influxTime(since))

	data, err := h.client.Query(query, client.Millisecond)
	if err != nil {
		return nil, err
	}

	result := make(map[bufferKey][]Value, len(data))
	for _, series := range data {
		if series.Columns[0] != "time" || series.Columns[2] != "value" {
			panic(fmt.Sprintf("wrong field order in %v", series.Points))
		}

		uid, did, sid, err := parseSeriesName(series.Name)
		if err != nil {
			return nil, err
		}

		key := bufferKey{uid, did, sid}
		values := make([]Value, 0, len(series.Points))
		for _, point := range series.Points {
			values = append(values, Value{goTime(point[0].(float64)), point[2].(float64)})
		}
		result[key] = values
	}

	return result, nil
}

func (h *influxHandler) removeSeriesFor(user, device, sensor uint64) error {
	query := fmt.Sprintf("drop series \"%s\"", seriesName(user, device, sensor))
	_, err := h.client.Query(query)
	return err
}

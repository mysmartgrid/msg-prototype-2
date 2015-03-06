package db

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type influxHandler struct {
	influxAddr, influxDb, influxUser, influxPass string
}

func influxTime(t time.Time) int64 {
	return t.Unix()*1000 + int64(t.Nanosecond()/1e6)
}

func goTime(t float64) time.Time {
	return time.Unix(int64(t/1000), int64(t)%1000*1e6)
}

func (h *influxHandler) dbUrl(args map[string]string) string {
	result := fmt.Sprintf(
		"%v/db/%v/series?time_precision=ms&u=%v&p=%v",
		h.influxAddr,
		url.QueryEscape(h.influxDb),
		url.QueryEscape(h.influxUser),
		url.QueryEscape(h.influxPass))

	for key, value := range args {
		result += fmt.Sprintf("&%v=%v", url.QueryEscape(key), url.QueryEscape(value))
	}

	return result
}

func (h *influxHandler) saveValuesAndClear(valueMap map[bufferKey][]Value) error {
	var buf bytes.Buffer

	buf.WriteRune('[')
	for key, values := range valueMap {
		if buf.Len() > 1 {
			buf.WriteRune(',')
		}
		fmt.Fprintf(&buf, `{"name":"%v-%v-%v",`, key.user, key.device, key.sensor)
		buf.WriteString(`"columns":["time","value"],`)
		buf.WriteString(`"points":[`)
		for i, value := range values {
			if i > 0 {
				buf.WriteRune(',')
			}
			fmt.Fprintf(&buf, `[%v,%v]`, influxTime(value.Time), value.Value)
		}
		buf.WriteString("]}")
		valueMap[key] = values[0:0]
	}
	buf.WriteRune(']')

	client := http.Client{Timeout: 1 * time.Second}
	resp, err := client.Post(h.dbUrl(nil), "application/json; charset=utf-8", &buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		data, _ := ioutil.ReadAll(resp.Body)
		return errors.New(resp.Status + " " + string(data))
	}
	return nil
}

func (h *influxHandler) loadValues(since time.Time, keys []bufferKey) (map[bufferKey][]Value, error) {
	type inputSeries struct {
		Name   string       `json:"name"`
		Points [][3]float64 `json:"points"`
	}

	var queryResult []inputSeries

	series := make([]string, 0, len(keys))
	for _, key := range keys {
		series = append(series, fmt.Sprintf("%v-%v-%v", key.user, key.device, key.sensor))
	}
	query := fmt.Sprintf("select * from /%v/ where time > %vms", strings.Join(series, "|"), influxTime(since))

	client := http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(h.dbUrl(map[string]string{"q": query}))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		data, _ := ioutil.ReadAll(resp.Body)
		return nil, errors.New(resp.Status + " " + string(data))
	}

	err = json.NewDecoder(resp.Body).Decode(&queryResult)
	if err != nil {
		return nil, err
	}

	result := make(map[bufferKey][]Value, len(keys))
	for _, series := range queryResult {
		parts := strings.Split(series.Name, "-")
		key := bufferKey{parts[0], parts[1], parts[2]}
		values := make([]Value, 0, len(series.Points))
		for _, point := range series.Points {
			values = append(values, Value{goTime(point[0]), point[2]})
		}
		result[key] = values
	}

	return result, nil
}

func (h *influxHandler) removeSeriesFor(user, device, sensor string) error {
	query := fmt.Sprintf("drop series \"%v-%v-%v\"", user, device, sensor)

	client := http.Client{Timeout: 1 * time.Second}
	resp, err := client.Get(h.dbUrl(map[string]string{"q": query}))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		data, _ := ioutil.ReadAll(resp.Body)
		return errors.New(resp.Status + " " + string(data))
	}

	return nil
}

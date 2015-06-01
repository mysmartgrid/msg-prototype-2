package msg2api

import (
	"encoding/json"
	"time"
)

type Measurement struct {
	Time  time.Time
	Value float64
}

func (p *Measurement) UnmarshalJSON(data []byte) error {
	var arr [2]float64

	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}

	ms := int64(arr[0])
	p.Time = time.Unix(ms/1000, (ms%1000)*1e6)
	p.Value = arr[1]
	return nil
}

func (p *Measurement) MarshalJSON() ([]byte, error) {
	return json.Marshal([2]float64{float64(jsTime(p.Time)), p.Value})
}

func jsTime(time time.Time) int64 {
	return 1000*time.Unix() + int64(time.Nanosecond()/1e6)
}

package db

type device struct {
	user *user
	id   string
}

func (d *device) AddSensor(id, unit string, port int32) (Sensor, error) {
	var seq uint64
	err := d.user.tx.QueryRow(`INSERT INTO sensors(sensor_id, device_id, user_id, name, port, unit) VALUES($1, $2, $3, $4, $5, $6) RETURNING sensor_seq`,
		id, d.id, d.user.id, id, port, unit).Scan(&seq)
	if err != nil {
		return nil, err
	}

	result := &sensor{d, id, seq}

	d.user.tx.db.bufferAdd <- seq

	return result, nil
}

func (d *device) Sensor(id string) Sensor {
	var seq uint64
	err := d.user.tx.QueryRow(`SELECT sensor_seq FROM sensors WHERE user_id = $1 AND device_id = $2 AND sensor_id = $3`, d.user.id, d.id, id).Scan(&seq)
	if err != nil {
		return nil
	}

	result := &sensor{d, id, seq}

	return result
}

func (d *device) Sensors() map[string]Sensor {
	rows, err := d.user.tx.Query(`SELECT sensor_id, sensor_seq FROM sensors WHERE user_id = $1 AND device_id = $2`, d.user.id, d.id)
	if err != nil {
		return nil
	}

	result := make(map[string]Sensor)
	defer rows.Close()
	for rows.Next() {
		var id string
		var seq uint64
		err = rows.Scan(&id, &seq)
		if err != nil {
			return nil
		}

		result[id] = &sensor{d, id, seq}
	}
	err = rows.Err()
	if err != nil {
		return nil
	}

	return result
}

func (d *device) RemoveSensor(id string) error {
	_, err := d.user.tx.Exec(`DELETE FROM sensors WHERE user_id = $1 AND device_id = $2 AND sensor_id = $3`, d.user.id, d.id, id)
	return err
}

func (d *device) Key() []byte {
	var key []byte
	err := d.user.tx.QueryRow(`SELECT key FROM devices WHERE user_id = $1 AND device_id = $2`, d.user.id, d.id).Scan(&key)
	if err != nil {
		return nil
	}
	return key
}

func (d *device) Id() string {
	return d.id
}

func (d *device) Name() string {
	var name string
	err := d.user.tx.QueryRow(`SELECT name FROM devices WHERE user_id = $1 AND device_id = $2`, d.user.id, d.id).Scan(&name)
	if err != nil {
		return ""
	}
	return name
}

func (d *device) SetName(name string) error {
	_, err := d.user.tx.Exec(`UPDATE devices SET name = $1 WHERE user_id = $2 AND device_id = $3`, name, d.user.id, d.id)
	return err
}

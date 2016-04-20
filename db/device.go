package db

type device struct {
	tx        *tx
	user      *user
	id        string
	isVirtual bool
}

func (d *device) AddSensor(id, unit string, port int32, factor float64) (Sensor, error) {
	var seq uint64
	err := d.tx.QueryRow(`INSERT INTO sensors(sensor_id, device_id, user_id, name, port, unit, factor, is_virtual) VALUES($1, $2, $3, $4, $5, $6, $7, $8) RETURNING sensor_seq`,
		id, d.id, d.User().ID(), id, port, unit, factor, false).Scan(&seq)
	if err != nil {
		return nil, err
	}

	result := &sensor{d.tx, d, id, seq, factor, false}

	d.tx.db.bufferAdd <- seq

	return result, nil
}

func (d *device) Sensor(id string) Sensor {
	var seq uint64
	var factor float64
	var isVirtual bool
	err := d.tx.QueryRow(`SELECT sensor_seq, factor, is_virtual FROM sensors WHERE user_id = $1 AND device_id = $2 AND sensor_id = $3`, d.User().ID(), d.id, id).Scan(&seq, &factor, &isVirtual)
	if err != nil {
		return nil
	}

	result := &sensor{d.tx, d, id, seq, factor, isVirtual}

	return result
}

func (d *device) Sensors() map[string]Sensor {
	rows, err := d.tx.Query(`SELECT sensor_id, sensor_seq, factor, is_virtual FROM sensors WHERE user_id = $1 AND device_id = $2`, d.User().ID(), d.id)
	if err != nil {
		return nil
	}

	result := make(map[string]Sensor)
	defer rows.Close()
	for rows.Next() {
		var id string
		var seq uint64
		var factor float64
		var isVirtual bool
		err = rows.Scan(&id, &seq, &factor, &isVirtual)
		if err != nil {
			return nil
		}

		result[id] = &sensor{d.tx, d, id, seq, factor, isVirtual}
	}
	err = rows.Err()
	if err != nil {
		return nil
	}

	return result
}

func (d *device) VirtualSensors() map[string]Sensor {
	rows, err := d.tx.Query(`SELECT sensor_id, sensor_seq, factor FROM sensors WHERE user_id = $1 AND device_id = $2 AND is_virtual = $3`, d.User().ID(), d.id, true)
	if err != nil {
		return nil
	}

	result := make(map[string]Sensor)
	defer rows.Close()
	for rows.Next() {
		var id string
		var seq uint64
		var factor float64
		err = rows.Scan(&id, &seq)
		if err != nil {
			return nil
		}

		result[id] = &sensor{d.tx, d, id, seq, factor, true}
	}
	err = rows.Err()
	if err != nil {
		return nil
	}

	return result
}

func (d *device) RemoveSensor(id string) error {
	_, err := d.tx.Exec(`DELETE FROM sensors WHERE user_id = $1 AND device_id = $2 AND sensor_id = $3`, d.User().ID(), d.id, id)
	return err
}

func (d *device) ID() string {
	return d.id
}

func (d *device) User() User {
	if d.user != nil {
		return d.user
	}

	var userID string
	err := d.tx.QueryRow(`SELECT user_id FROM devices WHERE device_id = $1`, d.id).Scan(&userID)
	if err != nil {
		return nil
	}

	d.user = &user{d.tx, userID}
	return d.user
}

func (d *device) Key() []byte {
	var key []byte
	err := d.tx.QueryRow(`SELECT key FROM devices WHERE user_id = $1 AND device_id = $2`, d.User().ID(), d.id).Scan(&key)
	if err != nil {
		return nil
	}
	return key
}

func (d *device) Name() string {
	var name string
	err := d.tx.QueryRow(`SELECT name FROM devices WHERE user_id = $1 AND device_id = $2`, d.User().ID(), d.id).Scan(&name)
	if err != nil {
		return ""
	}
	return name
}

func (d *device) SetName(name string) error {
	_, err := d.tx.Exec(`UPDATE devices SET name = $1 WHERE user_id = $2 AND device_id = $3`, name, d.User().ID(), d.id)
	return err
}

func (d *device) IsVirtual() bool {
	return d.isVirtual
}

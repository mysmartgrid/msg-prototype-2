package db

type group struct {
	tx *tx
	id string
}

func (g *group) AddUser(id string) error {
	_, err := g.tx.Exec(`INSERT INTO user_groups(user_id, group_id, is_admin) VALUES ($1,$2,$3)`, id, g.id, false)
	return err
}

func (g *group) RemoveUser(id string) error {
	_, err := g.tx.Exec(`DELETE FROM user_groups WHERE user_id = $1 AND group_id = $2`, id, g.id)
	return err
}

func (g *group) GetUsers() map[string]User {
	rows, err := g.tx.Query(`SELECT user_id FROM user_groups WHERE group_id = $1`, g.id)
	if err != nil {
		return nil
	}

	result := make(map[string]User)
	defer rows.Close()
	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			return nil
		}

		result[id] = &user{g.tx, id}
	}
	err = rows.Err()
	if err != nil {
		return nil
	}

	return result
}

func (g *group) SetAdmin(id string) error {
	_, err := g.tx.Exec(`UPDATE user_groups SET is_admin = true WHERE user_id = $1 AND group_id = $2`, id, g.id)
	return err
}

func (g *group) UnsetAdmin(id string) error {
	_, err := g.tx.Exec(`UPDATE user_groups SET is_admin = false WHERE user_id = $1 AND group_id = $2`, id, g.id)
	return err
}

func (g *group) GetAdmins() map[string]User {
	rows, err := g.tx.Query(`SELECT user_id FROM user_groups WHERE group_id = $1 AND is_admin = true`, g.id)
	if err != nil {
		return nil
	}

	result := make(map[string]User)
	defer rows.Close()
	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			return nil
		}

		result[id] = &user{g.tx, id}
	}
	err = rows.Err()
	if err != nil {
		return nil
	}

	return result
}

func (g *group) AddSensor(deviceID string, sensorID string) error {
	_, err := g.tx.Exec(`INSERT INTO sensor_groups(sensor_seq, group_id) `+
		`SELECT sensor_seq, $1 FROM sensors WHERE device_id = $2 AND sensor_id = $3`, g.id, deviceID, sensorID)
	return err
}

func (g *group) RemoveSensor(deviceID string, sensorID string) error {
	_, err := g.tx.Exec(`DELETE FROM sensor_groups WHERE group_id = $1 AND sensor_seq IN `+
		`(SELECT sensor_seq FROM sensors WHERE device_id = $2 AND sensor_id = $3)`, g.id, deviceID, sensorID)
	return err
}

func (g *group) GetSensors() []Sensor {
	rows, err := g.tx.Query(`SELECT sensors.sensor_seq, sensor_id, factor, is_virtual `+
		`FROM sensor_groups INNER JOIN sensors ON sensor_groups.sensor_seq = sensors.sensor_seq WHERE group_id = $1`, g.id)
	if err != nil {
		return nil
	}

	var result []Sensor

	defer rows.Close()
	for rows.Next() {
		var seq uint64
		var id string
		var factor float64
		var isVirtual bool

		err = rows.Scan(&seq, &id, &factor, &isVirtual)
		if err != nil {
			return nil
		}
		sensor := &sensor{g.tx, nil, id, seq, factor, isVirtual}
		result = append(result, sensor)
	}
	err = rows.Err()
	if err != nil {
		return nil
	}

	return result
}

func (g *group) ID() string {
	return g.id
}

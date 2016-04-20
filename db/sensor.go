package db

type sensor struct {
	tx        *tx
	device    *device
	id        string
	seq       uint64
	factor    float64
	isVirtual bool
}

func (s *sensor) ID() string {
	return s.id
}

func (s *sensor) DbID() uint64 {
	return s.seq
}

func (s *sensor) Name() string {
	var name string
	err := s.tx.QueryRow(`SELECT name FROM sensors WHERE sensor_seq = $1`, s.seq).Scan(&name)
	if err != nil {
		return ""
	}
	return name
}

func (s *sensor) Device() Device {
	if s.device != nil {
		return s.device
	}

	var deviceID string
	var isVirtual bool

	err := s.tx.QueryRow(`SELECT devices.device_id, devices.is_virtual FROM sensors INNER JOIN devices ON devices.device_id = sensors.device_id WHERE sensor_seq = $1`, s.DbID()).Scan(&deviceID, &isVirtual)
	if err != nil {
		return nil
	}

	s.device = &device{s.tx, nil, deviceID, isVirtual}

	return s.device
}

func (s *sensor) SetName(name string) error {
	_, err := s.tx.Exec(`UPDATE sensors SET name = $1 WHERE sensor_seq = $2`, name, s.seq)
	return err
}

func (s *sensor) Groups() map[string]Group {
	rows, err := s.tx.Query(`SELECT group_id FROM sensor_groups WHERE sensor_seq = $1`, s.seq)
	if err != nil {
		return nil
	}

	result := make(map[string]Group)
	defer rows.Close()
	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			return nil
		}

		result[id] = &group{s.tx, id}

	}
	err = rows.Err()
	if err != nil {
		return nil
	}

	return result
}

func (s *sensor) Port() int32 {
	var port int32
	err := s.tx.QueryRow(`SELECT port FROM sensors WHERE sensor_seq = $1`, s.seq).Scan(&port)
	if err != nil {
		return -1
	}
	return port
}

func (s *sensor) Unit() string {
	var unit string
	err := s.tx.QueryRow(`SELECT unit FROM sensors WHERE sensor_seq = $1`, s.seq).Scan(&unit)
	if err != nil {
		return ""
	}
	return unit
}

func (s *sensor) Factor() float64 {
	return s.factor
}

func (s *sensor) IsVirtual() bool {
	return s.isVirtual
}

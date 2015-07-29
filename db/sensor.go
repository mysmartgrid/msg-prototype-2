package db

type sensor struct {
	device *device
	id     string
	seq    uint64
}

func (s *sensor) Id() string {
	return s.id
}

func (s *sensor) DbId() uint64 {
	return s.seq
}

func (s *sensor) Name() string {
	var name string
	err := s.device.user.tx.QueryRow(`SELECT name FROM sensors WHERE sensor_seq = $1`, s.seq).Scan(&name)
	if err != nil {
		return ""
	}
	return name
}

func (s *sensor) SetName(name string) error {
	_, err := s.device.user.tx.Exec(`UPDATE sensors SET name = $1 WHERE sensor_seq = $2`, name, s.seq)
	return err
}

func (s *sensor) Port() int32 {
	var port int32
	err := s.device.user.tx.QueryRow(`SELECT port FROM sensors WHERE sensor_seq = $1`, s.seq).Scan(&port)
	if err != nil {
		return -1
	}
	return port
}

func (s *sensor) Unit() string {
	var unit string
	err := s.device.user.tx.QueryRow(`SELECT unit FROM sensors WHERE sensor_seq = $1`, s.seq).Scan(&unit)
	if err != nil {
		return ""
	}
	return unit
}

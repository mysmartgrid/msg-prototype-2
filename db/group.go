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

func (g *group) AddSensor(dbid uint64) error {
	_, err := g.tx.Exec(`INSERT INTO sensor_groups(sensor_seq, group_id) VALUES ($1,$2)`, dbid, g.id)
	return err
}

func (g *group) RemoveSensor(dbid uint64) error {
	_, err := g.tx.Exec(`DELETE FROM sensor_groups WHERE sensor_seq = $1 AND group_id = $2`, dbid, g.id)
	return err
}

func (g *group) GetSensors() []uint64 {
	rows, err := g.tx.Query(`SELECT sensor_seq FROM sensor_groups WHERE group_id = $1`, g.id)
	if err != nil {
		return nil
	}

	var result []uint64
	defer rows.Close()
	for rows.Next() {
		var id uint64
		err = rows.Scan(&id)
		if err != nil {
			return nil
		}

		result = append(result, id)
	}
	err = rows.Err()
	if err != nil {
		return nil
	}

	return result
}

func (g *group) Id() string {
	return g.id
}

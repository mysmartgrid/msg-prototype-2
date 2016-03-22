package db

import (
	"database/sql"
)

type tx struct {
	db *db
	*sql.Tx
}

func (tx *tx) AddUser(id, password string) (User, error) {
	if tx.User(id) != nil {
		return nil, ErrIDExists
	}

	_, err := tx.Exec(`INSERT INTO users(user_id, is_admin) VALUES($1, false)`, id)
	if err != nil {
		return nil, err
	}

	result := &user{tx, id}

	if err := result.init(password); err != nil {
		return nil, err
	}
	return result, nil
}

func (tx *tx) User(id string) User {
	var userID string
	err := tx.QueryRow(`SELECT user_id FROM users WHERE user_id = $1`, id).Scan(&userID)
	if err != nil {
		return nil
	}

	result := &user{tx, id}

	return result
}

func (tx *tx) RemoveUser(id string) error {
	_, err := tx.Exec(`DELETE FROM users WHERE user_id = $1`, id)
	return err
}

func (tx *tx) Users() map[string]User {
	rows, err := tx.Query(`SELECT user_id FROM users`)
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

		result[id] = &user{tx, id}
	}
	err = rows.Err()
	if err != nil {
		return nil
	}

	return result
}

func (tx *tx) AddGroup(id string) (Group, error) {
	if tx.Group(id) != nil {
		return nil, ErrIDExists
	}

	_, err := tx.Exec(`INSERT INTO groups VALUES($1)`, id)
	if err != nil {
		return nil, err
	}

	result := &group{tx, id}

	return result, nil
}

func (tx *tx) RemoveGroup(id string) error {
	_, err := tx.Exec(`DELETE FROM groups WHERE group_id = $1`, id)
	return err
}

func (tx *tx) Group(id string) Group {
	var groupID string
	err := tx.QueryRow(`SELECT group_id FROM groups WHERE group_id = $1`, id).Scan(&groupID)
	if err != nil {
		return nil
	}

	result := &group{tx, id}

	return result
}

func (tx *tx) Groups() map[string]Group {
	rows, err := tx.Query(`SELECT group_id FROM group`)
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

		result[id] = &group{tx, id}
	}
	err = rows.Err()
	if err != nil {
		return nil
	}

	return result
}

func (tx *tx) Sensor(id string) Sensor {
	var deviceId string
	var userId   string
	err := tx.QueryRow(`SELECT device_id,user_id FROM sensors WHERE sensor_id = $1`, id).Scan(&deviceId, &userId)
	if err != nil {
		return nil
	}
	user := tx.User(userId)	
	if( user != nil ) {
		device := user.Device(deviceId)
		return device.Sensor(id)
	}
	return nil
}

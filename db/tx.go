package db

import (
	"database/sql"
	"time"
)

type tx struct {
	db *db
	*sql.Tx
}

func (tx *tx) AddUser(id, password string) (User, error) {
	if tx.User(id) != nil {
		return nil, IdExists
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
	var user_id string
	err := tx.QueryRow(`SELECT user_id FROM users WHERE user_id = $1`, id).Scan(&user_id)
	if err != nil {
		return nil
	}

	result := &user{tx, id}

	return result
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

func (tx *tx) loadReadings(since, until time.Time, user User, res TimeRes, sensors map[Device][]Sensor) (map[Device]map[Sensor][]Value, error) {
	keys := make([]uint64, 0)
	for _, sensors := range sensors {
		for _, sensor := range sensors {
			keys = append(keys, sensor.DbId())
		}
	}

	queryResult, err := tx.db.sqldb.loadValues(since, until, res, keys)
	if err != nil {
		return nil, err
	}

	result := make(map[Device]map[Sensor][]Value)
	for device, sensors := range sensors {
		result[device] = make(map[Sensor][]Value)
		for _, sensor := range sensors {
			result[device][sensor] = queryResult[sensor.DbId()]
		}
	}

	return result, nil
}

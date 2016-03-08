package db

import (
	"github.com/mysmartgrid/msg2api"
	"golang.org/x/crypto/bcrypt"
	"time"
)

type user struct {
	tx *tx
	id string
}

func (u *user) init(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 0)
	if err != nil {
		return err
	}

	_, err = u.tx.Exec(`UPDATE users SET pw_hash = $1 WHERE user_id = $2`, hash, u.id)
	if err != nil {
		return err
	}

	return err
}

func (u *user) HasPassword(pw string) bool {
	var pwHash []byte
	err := u.tx.QueryRow(`SELECT pw_hash FROM users WHERE user_id = $1`, u.id).Scan(&pwHash)
	if err != nil {
		return false
	}

	err = bcrypt.CompareHashAndPassword(pwHash, []byte(pw))
	return err == nil
}

func (u *user) AddDevice(id string, key []byte, isVirtual bool) (Device, error) {
	_, err := u.tx.Exec(`INSERT INTO devices(device_id, name, key, user_id, is_virtual) VALUES($1, $2, $3, $4, $5)`,
		id, id, key, u.id, isVirtual)
	if err != nil {
		return nil, err
	}

	result := &device{u, id, isVirtual}
	return result, nil
}

func (u *user) RemoveDevice(id string) error {
	_, err := u.tx.Exec(`DELETE FROM devices WHERE user_id = $1 and device_id = $2`, u.id, id)
	return err
}

func (u *user) Device(id string) Device {
	var deviceId string
	var isVirtual bool
	err := u.tx.QueryRow(`SELECT device_id, is_virtual FROM devices WHERE user_id = $1 and device_id = $2`, u.id, id).Scan(&deviceId, &isVirtual)
	if err != nil {
		return nil
	}

	result := &device{u, id, isVirtual}

	return result
}

func (u *user) Devices() map[string]Device {
	rows, err := u.tx.Query(`SELECT device_id, is_virtual FROM devices WHERE user_id = $1`, u.id)
	if err != nil {
		return nil
	}

	result := make(map[string]Device)
	defer rows.Close()
	for rows.Next() {
		var id string
		var isVirtual bool
		err = rows.Scan(&id, &isVirtual)
		if err != nil {
			return nil
		}

		result[id] = &device{u, id, isVirtual}
	}
	err = rows.Err()
	if err != nil {
		return nil
	}

	return result
}

func (u *user) VirtualDevices() map[string]Device {
	rows, err := u.tx.Query(`SELECT device_id FROM devices WHERE user_id = $1 AND is_virtual = $2`, u.id, true)
	if err != nil {
		return nil
	}

	result := make(map[string]Device)
	defer rows.Close()
	for rows.Next() {
		var id string
		err = rows.Scan(&id)
		if err != nil {
			return nil
		}

		result[id] = &device{u, id, true}
	}
	err = rows.Err()
	if err != nil {
		return nil
	}

	return result
}

func (u *user) Groups() map[string]Group {
	rows, err := u.tx.Query(`SELECT group_id FROM user_groups WHERE user_id = $1`, u.id)
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

		result[id] = &group{u.tx, id}

	}
	err = rows.Err()
	if err != nil {
		return nil
	}

	return result
}

func (u *user) IsGroupAdmin(groupId string) bool {
	var isAdmin bool
	err := u.tx.QueryRow(`SELECT is_admin FROM user_groups WHERE user_id = $1 AND group_id = $2`, u.id, groupId).Scan(&isAdmin)
	if err == nil {
		return isAdmin
	}
	return false
}

func (u *user) IsAdmin() bool {
	var isAdmin bool
	err := u.tx.QueryRow(`SELECT is_admin FROM users WHERE user_id = $1`, u.id).Scan(&isAdmin)
	if err == nil {
		return isAdmin
	}
	return false
}

func (u *user) SetAdmin(b bool) error {
	_, err := u.tx.Exec(`UPDATE users SET is_admin = $1 WHERE user_id = $2`, b, u.id)
	return err
}

func (u *user) Id() string {
	return u.id
}

func (u *user) LoadReadings(since, until time.Time, resolution string, sensors map[string][]string) (map[string]map[string][]msg2api.Measurement, error) {
	var keys []uint64
	sensorsByKey := make(map[uint64]Sensor)

	for devID, sensorIDs := range sensors {
		dev := u.Device(devID)
		if dev != nil {
			for _, sensorID := range sensorIDs {
				sensor := dev.Sensor(sensorID)
				if sensor != nil {
					keys = append(keys, sensor.DbId())
					sensorsByKey[sensor.DbId()] = sensor
				}
			}
		}
	}

	readings, err := u.tx.db.sqldb.loadValues(since, until, resolution, keys)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string][]msg2api.Measurement)
	for dbid, values := range readings {
		devID := sensorsByKey[dbid].Device().Id()
		if _, ok := result[devID]; !ok {
			result[devID] = make(map[string][]msg2api.Measurement)
		}
		result[devID][sensorsByKey[dbid].Id()] = values
	}

	return result, nil
}

package db

import (
	"fmt"
	"log"
	"os"
	"time"
)

func measureTime(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}

func (d *db) clearDb() {
	defer measureTime(time.Now(), "Clearing database")
	_, err := d.sqldb.db.Exec(`TRUNCATE users CASCADE`)

	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func (d *db) deviceEmulator(sensors []Sensor, interval time.Duration, done, hold chan bool) {
	ok := true
	for ok {
		select {
		case _, ok = <-hold:
		}
	}

	for {
		select {
		case _, ok := <-done:
			if !ok {
				return
			}
		default:

			for _, sensor := range sensors {
				d.AddReading(sensor, time.Now(), 0.39393)
			}
		}
		time.Sleep(interval)
	}
}

func (d *db) benchAddUsers(count int) []User {
	defer measureTime(time.Now(), fmt.Sprintf("Adding %d users", count))

	names := make([]string, count)
	passwords := make([]string, count)
	var result []User

	for i := 0; i < count; i++ {
		names[i] = fmt.Sprintf("Bench User %d", i)
		passwords[i] = fmt.Sprintf("benchmark%d", i)
	}

	for i := 0; i < count; i++ {
		err := d.Update(func(tx Tx) error {
			user, err := tx.AddUser(names[i], passwords[i])
			if err != nil {
				return err
			}

			result = append(result, user)
			return nil
		})

		if err != nil {
			log.Print(err)
			os.Exit(1)
		}
	}
	return result
}

func (d *db) benchAddDevices(count int) map[User][]Device {
	defer measureTime(time.Now(), fmt.Sprintf("Adding %d devices per user", count))

	names := make([]string, count)
	result := make(map[User][]Device)

	for i := 0; i < count; i++ {
		names[i] = fmt.Sprintf("benchdev%d", i)
	}

	err := d.Update(func(tx Tx) error {
		users := tx.Users()
		for _, user := range users {
			for i := 0; i < count; i++ {
				device, err := user.AddDevice(names[i], nil)
				if err != nil {
					return err
				}
				result[user] = append(result[user], device)
			}
		}
		return nil
	})

	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	return result
}

func (d *db) benchAddSensors(count int) map[User]map[Device][]Sensor {
	defer measureTime(time.Now(), fmt.Sprintf("Adding %d sensors per device", count))

	names := make([]string, count)
	result := make(map[User]map[Device][]Sensor)

	for i := 0; i < count; i++ {
		names[i] = fmt.Sprintf("benchsens%d", i)
	}

	err := d.Update(func(tx Tx) error {
		users := tx.Users()
		for _, user := range users {
			result[user] = make(map[Device][]Sensor)
			devices := user.Devices()
			for _, device := range devices {
				for i := 0; i < count; i++ {
					sensor, err := device.AddSensor(names[i], "JW", 1)
					if err != nil {
						return err
					}
					result[user][device] = append(result[user][device], sensor)
				}
			}
		}
		return nil
	})

	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	return result
}

func (d *db) PeriodicRate(interval time.Duration, done, hold chan bool) {
	ok := true
	for ok {
		select {
		case _, ok = <-hold:
		}
	}

	start := time.Now()

	for {
		select {
		case _, ok := <-done:
			if !ok {
				return
			}
		default:
			elapsed := time.Since(start)
			var count int64
			d.sqldb.db.QueryRow(`SELECT COUNT(sensor) FROM measure_raw`).Scan(&count)
			log.Printf("Current Rate: %.2f v/s", float64(count)/elapsed.Seconds())
		}
		time.Sleep(interval)
	}
}

func (d *db) benchAddReadings(sensors map[User]map[Device][]Sensor, duration time.Duration, interval time.Duration) float64 {
	defer measureTime(time.Now(), fmt.Sprintf("Adding readings"))

	log.Printf("Running for %v", duration)

	done := make(chan bool)
	hold := make(chan bool)

	for _, device := range sensors {
		for _, devsensors := range device {
			go d.deviceEmulator(devsensors, interval, done, hold)
		}
	}

	go d.PeriodicRate(time.Second, done, hold)

	close(hold)

	time.Sleep(duration)

	close(done)
	// Wait for buffer to flush
	time.Sleep(time.Second * 2)

	var count int64
	d.sqldb.db.QueryRow(`SELECT COUNT(*) FROM measure_raw`).Scan(&count)

	return float64(count) / duration.Seconds()
}

func (d *db) RunBenchmark(usr_cnt, dev_cnt, sns_cnt int, duration time.Duration) {
	defer measureTime(time.Now(), "Benchmark")
	d.clearDb()
	d.benchAddUsers(usr_cnt)
	d.benchAddDevices(dev_cnt)
	sensors := d.benchAddSensors(sns_cnt)
	rate := d.benchAddReadings(sensors, duration, time.Second*1)

	log.Printf("==== Result ====")
	log.Printf("Simulated %d users having %d devices having %d sensors", usr_cnt, dev_cnt, sns_cnt)
	log.Printf("Total of %d sensors", usr_cnt*dev_cnt*sns_cnt)
	log.Printf("Wrote %.2f values per second", rate)
	log.Printf("%.2f v/s per device", rate/float64(usr_cnt*dev_cnt))
	log.Printf("%.2f v/s per sensor", rate/float64(usr_cnt*dev_cnt*sns_cnt))

}

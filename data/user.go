package data

import (
	log "github.com/sirupsen/logrus"
)

type UserRecord struct {
	UniqueID string
}

// LookupUser returns true if the user is already in the database or else false
func LookupUser(uid string) bool {
	var count int
	stmt, err := Db.Prepare("select count(unique_id) from users where unique_id = ?")
	if err != nil {
		log.Fatal(err)
	}
	err = stmt.QueryRow(uid).Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	if count != 1 {
		return false
	}
	return true
}

// InsertUser creates a new user record
func InsertUser(uid string) error {
	tx, err := Db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("insert into users(unique_id) values(?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(uid)
	if err != nil {
		return err
	}
	tx.Commit()
	return nil
}

// GetUsers returns a slice of strings with each unique_id
func GetUsers() []string {
	names := []string{}
	rows, err := Db.Query("select unique_id from users")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			log.Fatal(err)
		}
		names = append(names, name)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	return names
}

// DeleteUser deletes a user record
func DeleteUser(uid string) error {
	tx, err := Db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("delete from users where unique_id = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	result, err := stmt.Exec(uid)
	if err != nil {
		return err
	}
	tx.Commit()
	_, err = result.RowsAffected()
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"user": uid,
	}).Info("User deleted")
	return nil
}

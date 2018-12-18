package data

import (
	log "github.com/sirupsen/logrus"
)

type TxEntry struct {
	UniqueID string
	TxType   string
}

// LookupTxEntry returns true if the user is already in the database or else false
func LookupTxEntry(txEntry *TxEntry) bool {
	var count int
	stmt, err := Db.Prepare("select count(*) from txlog where unique_id = ? and txtype = ?")
	if err != nil {
		log.Fatal(err)
	}
	err = stmt.QueryRow(txEntry.UniqueID, txEntry.TxType).Scan(&count)
	if err != nil {
		log.Fatal(err)
	}
	if count != 1 {
		return false
	}
	return true
}

// InsertTxEntry creates a new TxEntry
func InsertTxEntry(txEntry *TxEntry) error {
	tx, err := Db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("insert into txlog(unique_id, txtype) values(?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(txEntry.UniqueID, txEntry.TxType)
	if err != nil {
		return err
	}
	tx.Commit()
	return nil
}

// DeleteTxEntry deletes a TxEntry
func DeleteTxEntry(txEntry *TxEntry) error {
	tx, err := Db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare("delete from txlog where unique_id = ? and txtype = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(txEntry.UniqueID, txEntry.TxType)
	if err != nil {
		return err
	}
	tx.Commit()
	return nil
}

// GetTxEntries pulls at most 10 entries from the table
func GetTxEntries() ([]TxEntry, error) {
	entries := []TxEntry{}
	rows, err := Db.Query("select unique_id, txtype from txlog limit 10")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var entry TxEntry
		err = rows.Scan(&entry.UniqueID, &entry.TxType)
		if err != nil {
			log.Fatal(err)
		}
		entries = append(entries, entry)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}
	return entries, nil
}

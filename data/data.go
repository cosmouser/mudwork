package data

import (
	"database/sql"
	"github.com/cosmouser/mudwork/config"
	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
)

var Db *sql.DB

func init() {
	var err error
	if *config.FlagProd {
		Db, err = sql.Open("sqlite3", config.C.DbPath)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		file, err := ioutil.TempFile(".", "testcache")
		if err != nil {
			log.Fatal(err)
		}
		os.Remove(file.Name())
		Db, err = sql.Open("sqlite3", file.Name())
		if err != nil {
			log.Fatal(err)
		}
		log.WithFields(log.Fields{
			"file": file.Name(),
		}).Info("Initializing test cache")
	}
	sqlStmt := `
	create table if not exists users
	(unique_id varchar(30) not null primary key);
	create table if not exists txlog
	(unique_id varchar(30) not null, txtype varchar(30) not null);
	`
	_, err = Db.Exec(sqlStmt)
	if err != nil {
		log.Fatal(err)
	}
}
func GetDBSize() float64 {
	var numPages, pageSize float64
	row := Db.QueryRow("pragma page_count")
	err := row.Scan(&numPages)
	if err != nil {
		log.Fatal(err)
	}
	row = Db.QueryRow("pragma page_size")
	err = row.Scan(&pageSize)
	if err != nil {
		log.Fatal(err)
	}
	return numPages * pageSize
}

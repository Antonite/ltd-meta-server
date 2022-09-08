package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

const holdsQuery = "create table if not exists %s(id int not null auto_increment,position_hash varchar(4096) not null,position varchar(4096) not null,total_value int not null,version_added varchar(16) not null,primary key(id));"
const sendsQuery = "create table if not exists %s(id int not null auto_increment,holds_id int not null,sends varchar(1024) not null,total_mythium int not null,adjusted_value int not null, held int not null,leaked int not null,primary key(id),foreign key(holds_id) references %s(id));"
const allTables = "show tables like '%_holds';"

const user = "antonite"
const database = "ltd"

func New() (*sql.DB, error) {
	pwd := os.Getenv("DB_PW")
	connstring := fmt.Sprintf("%s:%s@/%s", user, pwd, database)
	db, err := sql.Open("mysql", connstring)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func CreateTable(db *sql.DB, name string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()
	holdsName := name + "_holds"
	query := fmt.Sprintf(holdsQuery, holdsName)
	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}

	defer stmt.Close()
	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	sendsName := name + "_sends"
	query = fmt.Sprintf(sendsQuery, sendsName, holdsName)
	sendsStmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}

	defer sendsStmt.Close()

	_, err = sendsStmt.Exec()
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return err
}

func GetTables(db *sql.DB) (map[string]bool, error) {
	tables := make(map[string]bool)

	rows, err := db.Query(allTables)
	defer rows.Close()
	if err != nil {
		return tables, err
	}

	for rows.Next() {
		var atable string
		err = rows.Scan(&atable)
		tables[atable] = true
	}

	return tables, rows.Err()
}

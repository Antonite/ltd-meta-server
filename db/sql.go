package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

const holdsQuery = "create table if not exists %s(id int not null auto_increment,position_hash varchar(2048) not null,position varchar(2048) not null,total_value int not null,version_added varchar(16) not null,won int not null,lost int not null,workers int not null, player varchar(64) not null, primary key(id),index version_index (version_added));"
const sendsQuery = "create table if not exists %s(id int not null auto_increment,holds_id int not null,sends varchar(1024) not null,held int not null,leaked int not null,leaked_amount int not null,primary key(id),foreign key(holds_id) references %s(id));"
const allTables = "show tables like '%_holds';"

const user = "antonite"
const database = "ltd"

func New() (*sql.DB, error) {
	pwd := os.Getenv("DB_PW")
	host := os.Getenv("DB_HOST")
	connstring := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s", user, pwd, host, database)
	db, err := sql.Open("mysql", connstring)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func Prep(db *sql.DB, table string) error {
	q := fmt.Sprintf("OPTIMIZE TABLE %s;", table)
	_, err := db.Exec(q)
	return err
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
	if err != nil {
		return tables, err
	}
	defer rows.Close()

	for rows.Next() {
		var atable string
		err = rows.Scan(&atable)
		tables[atable] = true
	}

	return tables, rows.Err()
}

func CloseRows(rows *sql.Rows) {
	if rows != nil {
		rows.Close()
	}
}

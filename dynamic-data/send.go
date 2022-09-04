package dynamicdata

import (
	"database/sql"
	"fmt"
)

const findSendQuery = `SELECT holds_id, sends, total_mythium, adjusted_value, held, leaked FROM %s where holds_id = %v and sends = '%s'`
const getSendsQuery = `SELECT holds_id, sends, total_mythium, adjusted_value, held, leaked FROM %s where holds_id = %v`
const insertSendsQuery = `INSERT INTO %s(holds_id, sends, total_mythium, adjusted_value, held, leaked) VALUES(?,?,?,?,?,?)`
const updateSendsQuery = `UPDATE %s SET held = ?, leaked = ? where holds_id = %v and sends = '%s'`
const getTopSendsQuery = `SELECT holds_id, sends, total_mythium, adjusted_value, held, leaked FROM %s`

type Send struct {
	HoldsID       int
	Sends         string
	TotalMythium  int
	AdjustedValue int
	Held          int
	Leaked        int
}

func FindSends(db *sql.DB, tb string, id int, sends string) (*Send, error) {
	q := fmt.Sprintf(findSendQuery, tb, id, sends)
	rows, err := db.Query(q)
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var s Send
		err = rows.Scan(&s.HoldsID, &s.Sends, &s.TotalMythium, &s.AdjustedValue, &s.Held, &s.Leaked)
		if err != nil {
			return &s, err
		}
	}

	return nil, nil
}

func (s *Send) InsertSend(db *sql.DB, tb string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	q := fmt.Sprintf(insertSendsQuery, tb)
	defer tx.Rollback()
	stmt, err := tx.Prepare(q)
	if err != nil {
		return err
	}

	defer stmt.Close()
	_, err = stmt.Exec(s.HoldsID, s.Sends, s.TotalMythium, s.AdjustedValue, s.Held, s.Leaked)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return err
}

func (s *Send) UpdateSend(db *sql.DB, tb string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	q := fmt.Sprintf(updateSendsQuery, tb, s.HoldsID, s.Sends)
	defer tx.Rollback()
	stmt, err := tx.Prepare(q)
	if err != nil {
		return err
	}

	defer stmt.Close()
	_, err = stmt.Exec(s.Held, s.Leaked)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return err
}

func getTopSends(db *sql.DB, tb string) ([]*Send, error) {
	q := fmt.Sprintf(getTopSendsQuery, tb)
	rows, err := db.Query(q)
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	sends := []*Send{}
	for rows.Next() {
		var s Send
		err = rows.Scan(&s.HoldsID, &s.Sends, &s.TotalMythium, &s.AdjustedValue, &s.Held, &s.Leaked)
		if err != nil {
			return nil, err
		}
		sends = append(sends, &s)
	}

	return sends, nil
}

package dynamicdata

import (
	"database/sql"
	"fmt"
)

const findSendQuery = `SELECT id, holds_id, sends, total_mythium, adjusted_value, held, leaked, leaked_amount FROM %s where holds_id = %v and sends = '%s'`
const insertSendsQuery = `INSERT INTO %s(holds_id, sends, total_mythium, adjusted_value, held, leaked, leaked_amount) VALUES(?,?,?,?,?,?,?)`
const updateSendsQuery = `UPDATE %s SET held = ?, leaked = ?, leaked_amount = ? where id = ?`
const getTopSendsQuery = `SELECT holds_id, sends, total_mythium, adjusted_value, held, leaked, leaked_amount FROM %s`

type Send struct {
	ID            int
	HoldsID       int
	Sends         string
	TotalMythium  int
	AdjustedValue int
	Held          int
	Leaked        int
	LeakedAmount  int

	// not saved
	LeakedRatio int
}

func FindSend(db *sql.DB, tb string, id int, sends string) (*Send, error) {
	q := fmt.Sprintf(findSendQuery, tb, id, sends)
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var s Send
	for rows.Next() {
		err = rows.Scan(&s.ID, &s.HoldsID, &s.Sends, &s.TotalMythium, &s.AdjustedValue, &s.Held, &s.Leaked, &s.LeakedAmount)
		return &s, err
	}

	return nil, nil
}

func (s *Send) InsertSend(db *sql.DB, tb string) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	q := fmt.Sprintf(insertSendsQuery, tb)
	stmt, err := tx.Prepare(q)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	resp, err := stmt.Exec(s.HoldsID, s.Sends, s.TotalMythium, s.AdjustedValue, s.Held, s.Leaked, s.LeakedAmount)
	if err != nil {
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	id, err := resp.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), nil
}

func (s *Send) UpdateSend(db *sql.DB, tb string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q := fmt.Sprintf(updateSendsQuery, tb)
	stmt, err := tx.Prepare(q)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(s.Held, s.Leaked, s.LeakedAmount, s.ID)
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
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sends := []*Send{}
	for rows.Next() {
		var s Send
		err = rows.Scan(&s.HoldsID, &s.Sends, &s.TotalMythium, &s.AdjustedValue, &s.Held, &s.Leaked, &s.LeakedAmount)
		if err != nil {
			return nil, err
		}
		sends = append(sends, &s)
	}

	return sends, nil
}

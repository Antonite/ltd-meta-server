package dynamicdata

import (
	"database/sql"
	"fmt"
)

const getHoldsQuery = `SELECT id, position_hash, position, total_value, version_added FROM %s where position_hash = '%s'`
const getHoldsByIDQuery = `SELECT id, position_hash, position, total_value, version_added FROM %s where id = '%v'`
const saveHoldQuery = `INSERT INTO %s(position_hash, position, total_value, version_added) VALUES(?,?,?,?)`

type Hold struct {
	ID           int
	PositionHash string
	Position     string
	TotalValue   int
	VersionAdded string
}

func FindHold(db *sql.DB, tb string, hash string) (*Hold, error) {
	q := fmt.Sprintf(getHoldsQuery, tb, hash)
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var h Hold
	for rows.Next() {
		err = rows.Scan(&h.ID, &h.PositionHash, &h.Position, &h.TotalValue, &h.VersionAdded)
		return &h, err
	}

	return nil, nil
}

func FindHoldByID(db *sql.DB, tb string, id int) (*Hold, error) {
	q := fmt.Sprintf(getHoldsByIDQuery, tb, id)
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var h Hold
	for rows.Next() {
		err = rows.Scan(&h.ID, &h.PositionHash, &h.Position, &h.TotalValue, &h.VersionAdded)
		return &h, err
	}

	return nil, nil
}

func (h *Hold) SaveHold(db *sql.DB, tb string) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	q := fmt.Sprintf(saveHoldQuery, tb)
	defer tx.Rollback()
	stmt, err := tx.Prepare(q)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	resp, err := stmt.Exec(h.PositionHash, h.Position, h.TotalValue, h.VersionAdded)
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

	return int(id), err
}

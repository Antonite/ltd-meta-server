package dynamicdata

import (
	"database/sql"
	"fmt"
)

const getHoldsQuery = `SELECT id, position_hash, position, total_value, won, lost, workers, version_added FROM %s where position_hash = '%s' and version_added = '%s'`
const getHoldsByIDQuery = `SELECT id, position_hash, position, total_value, won, lost, workers, version_added FROM %s where id = '%v'`
const saveHoldQuery = `INSERT INTO %s(position_hash, position, total_value, won, lost, workers, version_added) VALUES(?,?,?,?,?,?,?)`
const updateHoldQuery = `UPDATE %s SET won = ?, lost = ?, workers = ? where id = ?`

type Hold struct {
	ID           int
	PositionHash string
	Position     string
	TotalValue   int
	Won          int
	Lost         int
	Workers      int
	VersionAdded string

	// not saved
	BiggestUnit string
}

func FindHold(db *sql.DB, tb string, hash string, version string) (*Hold, error) {
	q := fmt.Sprintf(getHoldsQuery, tb, hash, version)
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var h Hold
	for rows.Next() {
		err = rows.Scan(&h.ID, &h.PositionHash, &h.Position, &h.TotalValue, &h.Won, &h.Lost, &h.Workers, &h.VersionAdded)
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
		err = rows.Scan(&h.ID, &h.PositionHash, &h.Position, &h.TotalValue, &h.Won, &h.Lost, &h.Workers, &h.VersionAdded)
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

	resp, err := stmt.Exec(h.PositionHash, h.Position, h.TotalValue, h.Won, h.Lost, h.Workers, h.VersionAdded)
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

func (h *Hold) UpdateHold(db *sql.DB, tb string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	q := fmt.Sprintf(updateHoldQuery, tb)
	defer tx.Rollback()
	stmt, err := tx.Prepare(q)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(h.Won, h.Lost, h.Workers, h.ID)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return err
}

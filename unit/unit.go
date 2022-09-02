package unit

import "database/sql"

type Unit struct {
	ID         string
	Name       string
	IconPath   string
	TotalValue int
	Version    string
}

func GetAll(db *sql.DB) (map[string]*Unit, error) {
	units := make(map[string]*Unit)

	rows, err := db.Query(`SELECT unit_id, name, total_value, icon_path, version FROM unit`)
	defer rows.Close()
	if err != nil {
		return units, err
	}

	for rows.Next() {
		var aunit Unit
		err = rows.Scan(&aunit.ID, &aunit.Name, &aunit.TotalValue, &aunit.IconPath, &aunit.Version)
		units[aunit.ID] = &aunit
	}

	return units, rows.Err()
}

func (unit *Unit) Save(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()
	stmt, err := tx.Prepare("INSERT INTO unit(unit_id, name, icon_path, total_value, version) VALUES(?,?,?,?,?)")
	if err != nil {
		return err
	}

	defer stmt.Close()
	_, err = stmt.Exec(unit.ID, unit.Name, unit.IconPath, unit.TotalValue, unit.Version)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return err
}

package unit

import "database/sql"

type Unit struct {
	ID         int
	UnitID     string
	Name       string
	IconPath   string
	TotalValue int
	Usable     bool
	Version    string
}

func GetAll(db *sql.DB) (map[string]*Unit, error) {
	units := make(map[string]*Unit)

	rows, err := db.Query(`SELECT id, unit_id, name, total_value, usable, icon_path, version FROM unit`)
	if err != nil {
		return units, err
	}
	defer rows.Close()

	for rows.Next() {
		var aunit Unit
		err = rows.Scan(&aunit.ID, &aunit.UnitID, &aunit.Name, &aunit.TotalValue, &aunit.Usable, &aunit.IconPath, &aunit.Version)
		units[aunit.UnitID] = &aunit
	}

	return units, rows.Err()
}

func (unit *Unit) Save(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO unit(unit_id, name, icon_path, total_value, usable, version) VALUES(?,?,?,?,?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(unit.UnitID, unit.Name, unit.IconPath, unit.TotalValue, unit.Usable, unit.Version)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return err
}

package mercenary

import "database/sql"

type Mercenary struct {
	ID          string
	Name        string
	IconPath    string
	MythiumCost int
	Version     string
}

func GetAll(db *sql.DB) (map[string]*Mercenary, error) {
	units := make(map[string]*Mercenary)

	rows, err := db.Query(`SELECT unit_id, name, mythium_cost, icon_path, version FROM mercenary`)
	defer rows.Close()
	if err != nil {
		return units, err
	}

	for rows.Next() {
		var aunit Mercenary
		err = rows.Scan(&aunit.ID, &aunit.Name, &aunit.MythiumCost, &aunit.IconPath, &aunit.Version)
		units[aunit.Name] = &aunit
	}

	return units, rows.Err()
}

func (unit *Mercenary) Save(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()
	stmt, err := tx.Prepare("INSERT INTO mercenary(unit_id, name, icon_path, mythium_cost, version) VALUES(?,?,?,?,?)")
	if err != nil {
		return err
	}

	defer stmt.Close()
	_, err = stmt.Exec(unit.ID, unit.Name, unit.IconPath, unit.MythiumCost, unit.Version)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return err
}

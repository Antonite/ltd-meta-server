package unit

import (
	"database/sql"
	"strconv"
)

type UnitUpgrade struct {
	UnitID    int
	UpgradeID int
}

func GetUpgrades(db *sql.DB) (map[string][]string, error) {
	upgrades := make(map[string][]string)
	rows, err := db.Query(`SELECT unit_id, upgrade_id FROM unit_upgrade`)
	if err != nil {
		return upgrades, err
	}
	defer rows.Close()
	for rows.Next() {
		var up UnitUpgrade
		err = rows.Scan(&up.UnitID, &up.UpgradeID)
		upgrades[strconv.Itoa(up.UnitID)] = append(upgrades[strconv.Itoa(up.UnitID)], strconv.Itoa(up.UpgradeID))
	}

	return upgrades, rows.Err()
}

func (up *UnitUpgrade) Save(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO unit_upgrade(unit_id, upgrade_id) VALUES(?,?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(up.UnitID, up.UpgradeID)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return err
}

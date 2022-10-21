package dynamicdata

import (
	"database/sql"
	"sort"
)

const selectVersions = "select distinct version_added from nightmare_wave_1_holds"

func GetVersions(db *sql.DB) ([]string, error) {
	rows, err := db.Query(selectVersions)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	versions := []string{}
	for rows.Next() {
		var v string
		err = rows.Scan(&v)
		versions = append(versions, v)
	}

	sort.Strings(versions)

	return versions, nil
}

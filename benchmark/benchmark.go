package benchmark

import (
	"database/sql"
	"sync"
)

type Benchmark struct {
	Wave   int
	UnitId string
	Value  int
	Mu     sync.Mutex
}

func GetAll(db *sql.DB) (map[int]map[string]*Benchmark, error) {
	benchmarks := make(map[int]map[string]*Benchmark)

	rows, err := db.Query(`SELECT wave, unit_id, value FROM benchmark`)
	defer rows.Close()
	if err != nil {
		return benchmarks, err
	}

	for rows.Next() {
		var abenchmark Benchmark
		err = rows.Scan(&abenchmark.Wave, &abenchmark.UnitId, &abenchmark.Value)
		val, ok := benchmarks[abenchmark.Wave]
		if !ok {
			val = make(map[string]*Benchmark)
		}
		val[abenchmark.UnitId] = &abenchmark
		benchmarks[abenchmark.Wave] = val
	}

	return benchmarks, rows.Err()
}

func (bm *Benchmark) Save(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()
	stmt, err := tx.Prepare("INSERT INTO benchmark(wave, unit_id, value) VALUES(?,?,?)")
	if err != nil {
		return err
	}

	defer stmt.Close()
	_, err = stmt.Exec(bm.Wave, bm.UnitId, bm.Value)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return err
}

package benchmark

import "database/sql"

type Benchmark struct {
	Wave   int
	UnitId int
	Value  int
}

func GetAll(db *sql.DB) (map[int]map[int]Benchmark, error) {
	benchmarks := make(map[int]map[int]Benchmark)

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
			val = make(map[int]Benchmark)
		}
		val[abenchmark.UnitId] = abenchmark
		benchmarks[abenchmark.Wave] = val
	}

	return benchmarks, rows.Err()
}

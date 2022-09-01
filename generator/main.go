package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/antonite/ltd-meta-server/benchmark"
	"github.com/antonite/ltd-meta-server/ltdapi"
	"github.com/antonite/ltd-meta-server/mercenary"
	"github.com/antonite/ltd-meta-server/server"
	"github.com/antonite/ltd-meta-server/unit"
	"github.com/antonite/ltd-meta-server/util"
)

const vers = "9.07.2"
const waves = 10

func main() {
	srv, err := server.New()
	if err != nil {
		panic("failed to create server")
	}

	fmt.Println("starting unit generation")
	if err := generateUnits(srv); err != nil {
		fmt.Println(err)
	}
	fmt.Println("finished unit generation")

	fmt.Println("starting table generation")
	if err := generateTables(srv); err != nil {
		fmt.Println(err)
	}
	fmt.Println("finished table generation")

	fmt.Println("starting initial benchmark")
	if err := initialBenchMark(srv); err != nil {
		fmt.Println(err)
	}
	fmt.Println("finished initial benchmark")
}

func generateTables(srv *server.Server) error {
	savedUnits, err := srv.GetUnits()
	if err != nil {
		return err
	}

	for k, v := range savedUnits {
		n := strings.TrimSuffix(k, "unit_id")
		for i := 1; i <= waves; i++ {
			if i == 1 && v.TotalValue >= 273 {
				continue
			}
			tableName := fmt.Sprintf("%swave_%v", n, i)
			if err := srv.CreateTable(tableName); err != nil {
				return err
			}
		}
	}

	return nil
}

func generateUnits(srv *server.Server) error {
	savedUnits, err := srv.GetUnits()
	if err != nil {
		return err
	}
	savedMercs, err := srv.GetMercs()
	if err != nil {
		return err
	}

	units := make(chan ltdapi.Unit)
	errChan := make(chan error, 1)
	go srv.Api.RequestUnits(srv.Version, units, errChan)
	for u := range units {
		if u.CategoryClass != "Standard" && !util.IsSpecialUnit(u.UnitId) {
			continue
		}
		switch u.UnitClass {
		case "Mercenary":
			if _, ok := savedMercs[u.UnitId]; !ok {
				if u.MythiumCost == "" {
					return errors.New(fmt.Sprintf("got a merc with empty myth cost: %s", u.UnitId))
				}
				cost, err := strconv.Atoi(u.MythiumCost)
				if err != nil {
					return err
				}
				newMerc := mercenary.Mercenary{
					ID:          u.UnitId,
					Name:        u.Name,
					IconPath:    u.IconPath,
					MythiumCost: cost,
					Version:     u.Version,
				}
				if err := srv.SaveMerc(newMerc); err != nil {
					return err
				}
			}
		case "Fighter":
			if _, ok := savedUnits[u.UnitId]; !ok {
				if u.TotalValue == "" {
					return errors.New(fmt.Sprintf("got a unit with empty total value: %s", u.UnitId))
				}
				val, err := strconv.Atoi(u.TotalValue)
				if err != nil {
					return errors.New(fmt.Sprintf("failed to convert unit total value: %s", u.UnitId))
				}
				newUnit := unit.Unit{
					ID:         u.UnitId,
					Name:       u.Name,
					IconPath:   u.IconPath,
					TotalValue: val,
					Version:    u.Version,
				}
				if err := srv.SaveUnit(newUnit); err != nil {
					return err
				}
			}
		}
	}
	for err := range errChan {
		return err
	}

	return nil
}

func initialBenchMark(srv *server.Server) error {
	allUnits, err := srv.GetUnits()
	if err != nil {
		return err
	}

	benchmarks := make(map[string]benchmark.Benchmark)
	for u := range allUnits {
		for i := 1; i <= waves; i++ {
			bn := util.GenerateUnitTableName(u, i)
			b := benchmark.Benchmark{
				Wave:   i,
				UnitId: u,
				Value:  0,
			}
			benchmarks[bn] = b
		}
	}

	// get games
	date := "2022-08-31%2018:00:00.000Z"
	games := make(chan ltdapi.Game)
	errChan := make(chan error, 1)
	go srv.Api.RequestGames(date, games, errChan)
	for g := range games {
		if (g.QueueType != "Normal" && g.QueueType != "Classic") || g.EndingWave <= 1 {
			continue
		}
		for _, player := range g.PlayersData {
			if player.Cross {
				continue
			}

			for i := 0; i < util.Min(len(player.BuildPerWave), 9); i++ {
				// nothing leaked
				if len(player.LeaksPerWave[i]) == 0 {
					// find most expensive unit
					expU, totalCost, err := unit.MostExpensive(player.BuildPerWave[i], allUnits)
					if err != nil {
						fmt.Println(player.LeaksPerWave[i])
						fmt.Println(player.BuildPerWave[i])
						fmt.Println(i)
						fmt.Println(g)
						return err
					}
					// get table name
					bn := util.GenerateUnitTableName(expU, i)
					// assign new benchmark
					if benchmarks[bn].Value == 0 || benchmarks[bn].Value > totalCost {
						newBm := benchmarks[bn]
						newBm.Value = totalCost
						benchmarks[bn] = newBm
					}
				}
			}
		}
	}
	for err := range errChan {
		return err
	}

	// save benchmarks
	for _, b := range benchmarks {
		if err := srv.SaveBenchmark(b); err != nil {
			return err
		}
	}

	return nil
}

package main

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/antonite/ltd-meta-server/benchmark"
	dynamicdata "github.com/antonite/ltd-meta-server/dynamic-data"
	"github.com/antonite/ltd-meta-server/ltdapi"
	"github.com/antonite/ltd-meta-server/mercenary"
	"github.com/antonite/ltd-meta-server/server"
	"github.com/antonite/ltd-meta-server/unit"
	"github.com/antonite/ltd-meta-server/util"
)

const vers = "9.07.2"
const waves = 10
const holdMultipler = 1.2
const workers = 20

type analysis struct {
	biggestUnitID  string
	biggestUnitPos string
	TotalValue     int
	adjustedValue  int
	TotalMythium   int
	sendHash       string
	positionHash   string
	position       string
}

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

	fmt.Println("starting historical generation")
	if err := generateHistoricalData(srv); err != nil {
		fmt.Println(err)
	}
	fmt.Println("finished historical generation")
}

func generateTables(srv *server.Server) error {
	savedUnits, err := srv.GetUnits()
	if err != nil {
		return err
	}

	for k, v := range savedUnits {
		n := strings.TrimSuffix(k, "unit_id")
		for i := 1; i <= waves; i++ {
			if i == 1 && v.TotalValue >= util.CashoutGold {
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
		// skip hybrid units
		if strings.HasPrefix(u.UnitId, "hybrid") || strings.HasPrefix(u.UnitId, "test_") {
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
				if err := srv.SaveMerc(&newMerc); err != nil {
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
				if err := srv.SaveUnit(&newUnit); err != nil {
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

func generateHistoricalData(srv *server.Server) error {
	allUnits, err := srv.GetUnits()
	if err != nil {
		return err
	}

	allMercs, err := srv.GetMercs()
	if err != nil {
		return err
	}

	benchmarks, err := srv.Getbenchmarks()
	if err != nil {
		return err
	}

	date := "2022-09-02%2000:00:00.000Z"
	games := make(chan ltdapi.Game)
	errChan := make(chan error, 1)
	wg := &sync.WaitGroup{}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go srv.Api.RequestGames(date, games, errChan, wg, w, workers)
	}
	go func(wg *sync.WaitGroup, games chan ltdapi.Game, errChan chan error) {
		wg.Wait()
		close(games)
		close(errChan)
	}(wg, games, errChan)
	for g := range games {
		if (g.QueueType != "Normal" && g.QueueType != "Classic") || g.EndingWave <= 1 {
			continue
		}
		for _, player := range g.PlayersData {
			if player.Cross {
				continue
			}

			for i := 0; i < util.Min(g.EndingWave-2, waves); i++ {
				// something was sent and built
				if len(player.MercenariesReceivedPerWave[i]) > 0 && len(player.BuildPerWave[i]) > 0 {
					// find most expensive unit
					anls, err := analyzeBoard(player, allUnits, allMercs, i)
					if err != nil {
						return err
					}

					// get benchmark
					bn, ok := benchmarks[i+1][anls.biggestUnitID]
					if !ok {
						fmt.Printf("wave: %v unit: %v\n", i+1, anls.biggestUnitID)
					}
					// check if hold is good enough
					if float64(anls.adjustedValue) < float64(bn.Value)*holdMultipler {
						bn.Mu.Lock()
						leaked := len(player.LeaksPerWave[i]) > 0
						if !leaked {
							// check hydra case
							if anls.biggestUnitID == util.Eggsack && len(player.BuildPerWave) > i+1 {
								fullHydra := util.IsFullHydra(player.BuildPerWave[i+1], anls.biggestUnitPos)
								// don't consider broken eggs as a hold
								if !fullHydra {
									leaked = true
								}
							}
						}
						if !leaked {
							if bn.Value == 0 || bn.Value > anls.adjustedValue {
								bn.Value = anls.adjustedValue
							}
						}
						bn.Mu.Unlock()
						tn := util.GenerateUnitTableName(anls.biggestUnitID, i+1)
						htn := tn + "_holds"
						id := 0
						h, err := srv.FindHold(htn, anls.positionHash)
						if err != nil {
							return err
						}
						if h == nil && leaked {
							continue
						}
						if h == nil {
							h = &dynamicdata.Hold{
								PositionHash: anls.positionHash,
								Position:     anls.position,
								TotalValue:   anls.TotalValue,
								VersionAdded: srv.Version,
							}
							id, err = srv.SaveHold(htn, h)
							if err != nil {
								return err
							}
						} else {
							id = h.ID
						}
						stn := tn + "_sends"
						s, err := srv.FindSends(stn, id, anls.sendHash)
						if err != nil {
							return err
						}
						if s == nil {
							newS := &dynamicdata.Send{
								HoldsID:       id,
								Sends:         anls.sendHash,
								TotalMythium:  anls.TotalMythium,
								AdjustedValue: anls.adjustedValue,
							}
							if leaked {
								newS.Leaked = 1
							} else {
								newS.Held = 1
							}
							err := srv.InsertSend(stn, newS)
							if err != nil {
								return err
							}
						} else {
							if leaked {
								s.Leaked = 1
							} else {
								s.Held = 1
							}
						}
					}
				}
			}
		}
	}
	for err := range errChan {
		return err
	}

	// save benchmarks
	for _, w := range benchmarks {
		for _, v := range w {
			if err := srv.SaveBenchmark(v); err != nil {
				return err
			}
		}
	}

	return nil
}

func analyzeBoard(player ltdapi.PlayersData, allUnits map[string]*unit.Unit, allMercs map[string]*mercenary.Mercenary, index int) (analysis, error) {
	anls := analysis{}
	expCost := 0
	// rehash board
	sort.Strings(player.BuildPerWave[index])
	diff, err := strconv.ParseFloat(strings.Split(strings.Split(player.BuildPerWave[index][0], ":")[1], "|")[1], 64)
	if err != nil {
		return anls, err
	}
	for _, u := range player.BuildPerWave[index] {
		id := strings.Split(u, ":")[0]
		existing, ok := allUnits[id]
		if !ok {
			return anls, errors.New(fmt.Sprintf("couldn't find unit in unit map: %s", id))
		}
		if expCost < existing.TotalValue {
			expCost = existing.TotalValue
			anls.biggestUnitID = existing.ID
			anls.biggestUnitPos = u
		}
		anls.TotalValue += existing.TotalValue

		hash, err := adjustUnit(u, diff)
		if err != nil {
			return anls, err
		}
		anls.positionHash += hash + ","
	}
	anls.positionHash = strings.TrimSuffix(anls.positionHash, ",")
	if anls.biggestUnitID == "" {
		return anls, errors.New("failed to compute most expensive unit")
	}

	// analyse sends
	for _, m := range player.MercenariesReceivedPerWave[index] {
		val, ok := allMercs[m]
		if !ok {
			return anls, errors.New(fmt.Sprintf("failed to find merc: %s", m))
		}
		anls.TotalMythium += val.MythiumCost
	}
	anls.adjustedValue = anls.TotalValue - int(math.Ceil(1.25*float64(anls.TotalMythium)))
	sort.Strings(player.MercenariesReceivedPerWave[index])
	anls.sendHash = strings.Join(player.MercenariesReceivedPerWave[index], ",")

	anls.position = strings.Join(player.BuildPerWave[index], ",")

	return anls, nil
}

func adjustUnit(u string, diff float64) (string, error) {
	build := strings.Split(u, ":")
	pos := strings.Split(build[1], "|")
	y, err := strconv.ParseFloat(pos[1], 64)
	if err != nil {
		return "", err
	}
	adjusted := int(y - diff)
	return fmt.Sprintf("%s:%s|%v:%s", build[0], pos[0], adjusted, build[2]), nil
}

func initialBenchMark(srv *server.Server) error {
	allUnits, err := srv.GetUnits()
	if err != nil {
		return err
	}

	allMercs, err := srv.GetMercs()
	if err != nil {
		return err
	}

	benchmarks := make(map[string]*benchmark.Benchmark)
	for u := range allUnits {
		for i := 1; i <= waves; i++ {
			if i == 1 && allUnits[u].TotalValue > util.CashoutGold {
				continue
			}
			bn := util.GenerateUnitTableName(u, i)
			b := benchmark.Benchmark{
				Wave:   i,
				UnitId: u,
				Value:  0,
				Mu:     &sync.Mutex{},
			}
			benchmarks[bn] = &b
		}
	}

	// get games
	date := "2022-09-3%2000:00:00.000Z"
	games := make(chan ltdapi.Game)
	errChan := make(chan error, 1)
	wg := &sync.WaitGroup{}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go srv.Api.RequestGames(date, games, errChan, wg, w, workers)
	}
	go func(wg *sync.WaitGroup, games chan ltdapi.Game, errChan chan error) {
		wg.Wait()
		close(games)
		close(errChan)
	}(wg, games, errChan)
	for g := range games {
		if (g.QueueType != "Normal" && g.QueueType != "Classic") || g.EndingWave <= 1 {
			continue
		}
		for _, player := range g.PlayersData {
			if player.Cross {
				continue
			}

			for i := 0; i < util.Min(g.EndingWave-2, waves); i++ {
				// nothing leaked
				if len(player.LeaksPerWave[i]) == 0 {
					// find most expensive unit
					anls, err := analyzeBoard(player, allUnits, allMercs, i)
					if err != nil {
						return err
					}
					// check hydra case
					if anls.biggestUnitID == util.Eggsack && len(player.BuildPerWave) > i+1 {
						fullHydra := util.IsFullHydra(player.BuildPerWave[i+1], anls.biggestUnitPos)
						// don't consider broken eggs as a hold
						if !fullHydra {
							continue
						}
					}

					// get table name
					bn := util.GenerateUnitTableName(anls.biggestUnitID, i+1)
					// assign new benchmark
					benchmarks[bn].Mu.Lock()
					if benchmarks[bn].Value == 0 || benchmarks[bn].Value > anls.TotalValue {
						newBm := benchmarks[bn]
						newBm.Value = anls.TotalValue
						benchmarks[bn] = newBm
					}
					benchmarks[bn].Mu.Unlock()
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

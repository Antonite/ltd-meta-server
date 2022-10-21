package main

import (
	"errors"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	dynamicdata "github.com/antonite/ltd-meta-server/dynamic-data"
	"github.com/antonite/ltd-meta-server/ltdapi"
	"github.com/antonite/ltd-meta-server/mercenary"
	"github.com/antonite/ltd-meta-server/server"
	"github.com/antonite/ltd-meta-server/unit"
	"github.com/antonite/ltd-meta-server/util"
)

const workers = 20
const minElo = 2600

type analysis struct {
	biggestUnitID  string
	biggestUnitPos string
	TotalValue     int
	TotalMythium   int
	sendHash       string
	positionHash   string
	position       string
}

func main() {
	start := time.Now()

	srv, err := server.New()
	if err != nil {
		panic("failed to create server")
	}

	fmt.Println(time.Now().Format("Mon Jan _2 15:04:05 2006") + ": starting unit generation")
	if err := generateUnits(srv); err != nil {
		fmt.Printf("failed to generate units: %v\n", err)
	}
	fmt.Println(time.Now().Format("Mon Jan _2 15:04:05 2006") + ": finished unit generation")

	fmt.Println(time.Now().Format("Mon Jan _2 15:04:05 2006") + ": starting table generation")
	if err := generateTables(srv); err != nil {
		fmt.Printf("failed to generate tables: %v\n", err)
	}
	fmt.Println(time.Now().Format("Mon Jan _2 15:04:05 2006") + ": finished table generation")

	fmt.Println(time.Now().Format("Mon Jan _2 15:04:05 2006") + ": starting historical generation")
	daysAgo, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic("failed to parse input param")
	}
	if err := generateHistoricalData(srv, daysAgo); err != nil {
		fmt.Printf("failed to generate historical: %v\n", err)
	}
	fmt.Println(time.Now().Format("Mon Jan _2 15:04:05 2006") + ": finished historical generation")

	totalTime := time.Now().Sub(start)
	hours := math.Floor(totalTime.Hours())
	minutes := math.Floor(totalTime.Minutes()) - hours*60
	seconds := math.Floor(totalTime.Seconds()) - hours*60*60 - minutes*60
	fmt.Printf("total processing time: %.0fh %.0fm %.0fs\n", hours, minutes, seconds)
}

func generateTables(srv *server.Server) error {
	savedUnits, err := srv.GetUnits()
	if err != nil {
		return err
	}

	for k, v := range savedUnits {
		if !v.Usable {
			continue
		}
		n := strings.TrimSuffix(k, "unit_id")
		for i := 1; i <= util.Waves; i++ {
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
	upgrades := make(map[string][]string)
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
			if _, ok := savedMercs[u.Name]; !ok {
				if u.MythiumCost == "" {
					return errors.New(fmt.Sprintf("got a merc with empty myth cost: %s", u.UnitId))
				}
				if u.IncomeBonus == "" {
					return errors.New(fmt.Sprintf("got a merc with empty income bonus: %s", u.UnitId))
				}
				cost, err := strconv.Atoi(u.MythiumCost)
				if err != nil {
					return err
				}
				inc, err := strconv.Atoi(u.IncomeBonus)
				if err != nil {
					return err
				}
				newMerc := mercenary.Mercenary{
					ID:          u.UnitId,
					Name:        u.Name,
					IconPath:    u.IconPath,
					MythiumCost: cost,
					IncomeBonus: inc,
					Version:     u.Version,
				}
				if err := srv.SaveMerc(&newMerc); err != nil {
					return err
				}
			}
		case "Fighter":
			for _, upgu := range u.UpgradesFrom {
				for _, upg := range strings.Split(upgu, " ") {
					if upg == "units" || upg == "" || upg == " " {
						continue
					}
					upgrades[upg] = append(upgrades[upg], u.UnitId)
				}
			}

			if _, ok := savedUnits[u.UnitId]; !ok {
				if u.TotalValue == "" {
					return errors.New(fmt.Sprintf("got a unit with empty total value: %s", u.UnitId))
				}
				val, err := strconv.Atoi(u.TotalValue)
				if err != nil {
					return errors.New(fmt.Sprintf("failed to convert unit total value: %s", u.UnitId))
				}
				if u.UnitId == "hell_raiser_buffed_unit_id" {
					u.Name = "Hell Raiser (Tantrum)"
				}
				newUnit := unit.Unit{
					UnitID:     u.UnitId,
					Name:       u.Name,
					IconPath:   u.IconPath,
					TotalValue: val,
					Usable:     true,
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

	// update upgrades
	allUnits, err := srv.GetUnits()
	if err != nil {
		return err
	}
	// custom upgrades
	upgrades["eggsack_unit_id"] = append(upgrades["eggsack_unit_id"], "hydra_unit_id")
	upgrades["hell_raiser_unit_id"] = append(upgrades["hell_raiser_unit_id"], "hell_raiser_buffed_unit_id")
	upgrades["pack_rat_unit_id"] = append(upgrades["pack_rat_unit_id"], "pack_rat_nest_unit_id")

	for k, v := range upgrades {
		for _, upg := range v {
			up := unit.UnitUpgrade{
				UnitID:    allUnits[k].ID,
				UpgradeID: allUnits[upg].ID,
			}
			if err = srv.SaveUpgrade(&up); err != nil {
				return err
			}
		}
	}

	return nil
}

func generateHistoricalData(srv *server.Server, daysAgo int) error {
	allUnits, err := srv.GetUnits()
	if err != nil {
		return err
	}

	allMercs, err := srv.GetMercs()
	if err != nil {
		return err
	}

	if err := srv.RefreshTables(); err != nil {
		return err
	}

	bounties := make(map[string]int)
	bounties["Crab"] = 6
	bounties["Wale"] = 7
	bounties["Hopper"] = 5
	bounties["Flying Chicken"] = 8
	bounties["Scorpion"] = 9
	bounties["Scorpion King"] = 36
	bounties["Rocko"] = 19
	bounties["Sludge"] = 10
	bounties["Blob"] = 2
	bounties["Kobra"] = 11

	today := time.Now().UTC().Add(time.Hour * -24 * time.Duration(daysAgo-1))
	yesterday := today.Add(time.Hour * -24)
	dateStart := fmt.Sprintf("%d-%02d-%02d%%2000:00:00.000Z", yesterday.Year(), yesterday.Month(), yesterday.Day())
	dateEnd := fmt.Sprintf("%d-%02d-%02d%%2000:00:00.000Z", today.Year(), today.Month(), today.Day())
	games := make(chan ltdapi.Game, 500)
	errChan := make(chan error, 1)
	wg := &sync.WaitGroup{}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go srv.Api.RequestGames(dateStart, dateEnd, games, errChan, wg, w, workers)
	}
	go func(wg *sync.WaitGroup, games chan ltdapi.Game, errChan chan error) {
		wg.Wait()
		close(games)
		close(errChan)
	}(wg, games, errChan)
	gamesProcessed := 0
	timeMarker := time.Now()

	// temp maps
	holds := make(map[int]map[string]*dynamicdata.Hold)
	sends := make(map[int]map[string]map[string]*dynamicdata.Send)
	for i := 0; i < util.Waves; i++ {
		holds[i] = make(map[string]*dynamicdata.Hold)
		sends[i] = make(map[string]map[string]*dynamicdata.Send)
	}

	// version regex
	reg, err := regexp.Compile("v[0-9]+.[0-9]+(.[0-9])*")
	if err != nil {
		return err
	}

	for g := range games {
		gamesProcessed++
		if gamesProcessed%1000 == 0 {
			fmt.Printf(time.Now().Format("Mon Jan _2 15:04:05 2006")+" processed %v games, rate: %v games per second \n", gamesProcessed, math.Ceil(float64(1000)/math.Abs(timeMarker.Sub(time.Now()).Seconds())*100)/100)
			timeMarker = time.Now()
		}
		if g.QueueType != "Normal" || g.EndingWave <= 1 {
			continue
		}
		if !reg.MatchString(g.Version) {
			continue
		}
		for _, player := range g.PlayersData {
			for i := 0; i < util.Min(g.EndingWave-2, util.Waves); i++ {
				if len(player.BuildPerWave[i]) == 0 {
					continue
				}

				// find most expensive unit
				anls, err := analyzeBoard(player, allUnits, allMercs, i)
				if err != nil {
					fmt.Printf("failed to analyze board: %v\n", err)
					continue
				}

				// check if we care about this unit
				tn := util.GenerateUnitTableName(anls.biggestUnitID, i+1)
				htn := tn + "_holds"
				if !srv.Tables[htn] {
					continue
				}

				h, ok := holds[i][anls.positionHash]
				// skip original low elo builds
				if !ok && player.OverallElo < minElo {
					continue
				}
				if !ok {
					h = &dynamicdata.Hold{
						PositionHash: anls.positionHash,
						Position:     anls.position,
						TotalValue:   anls.TotalValue,
						VersionAdded: util.NormalizeVersion(g.Version),
						BiggestUnit:  anls.biggestUnitID,
					}
					holds[i][anls.positionHash] = h
				}
				won := player.GameResult == "won"
				if won {
					h.Won++
				} else {
					h.Lost++
				}

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

				sMap, ok := sends[i][anls.positionHash]
				if !ok {
					sMap = make(map[string]*dynamicdata.Send)
					sends[i][anls.positionHash] = sMap
				}
				s, ok := sMap[anls.sendHash]
				if !ok {
					s = &dynamicdata.Send{
						Sends:        anls.sendHash,
						TotalMythium: anls.TotalMythium,
					}
					sMap[anls.sendHash] = s
				}
				if leaked {
					s.Leaked++
				} else {
					s.Held++
				}

				for _, leak := range player.LeaksPerWave[i] {
					m, ok := allMercs[leak]
					if ok {
						s.LeakedAmount += m.IncomeBonus
					} else {
						m, ok := bounties[leak]
						if ok {
							s.LeakedAmount += m
						}
					}
				}
			}
		}
	}
	for err := range errChan {
		fmt.Printf("error in error channel: %v\n", err)
		return err
	}

	holdsProcessed := 0
	l := 0
	for i := 0; i < util.Waves; i++ {
		l += len(holds[i])
	}

	for i := 0; i < util.Waves; i++ {
		for _, h := range holds[i] {
			holdsProcessed++
			if holdsProcessed%100 == 0 {
				rate := math.Ceil(float64(100)/math.Abs(timeMarker.Sub(time.Now()).Seconds())*100) / 100
				fmt.Printf(time.Now().Format("Mon Jan _2 15:04:05 2006")+" processed %v holds, rate: %v holds per second, ETA: %v seconds \n", holdsProcessed, rate, math.Ceil(float64(l-holdsProcessed)/rate))
				timeMarker = time.Now()
			}

			allS, ok := sends[i][h.PositionHash]
			if !ok {
				continue
			}

			// update hold
			tn := util.GenerateUnitTableName(h.BiggestUnit, i+1)
			htn := tn + "_holds"
			dbHold, err := srv.FindHold(htn, h.PositionHash, h.VersionAdded)
			if err != nil {
				fmt.Printf("failed to find hold: %s, tn: %s, err: %v\n", h.PositionHash, htn, err)
				// todo: add retries
				continue
			}
			if dbHold == nil {
				id, err := srv.SaveHold(htn, h)
				if err != nil || id == 0 {
					fmt.Printf("failed to save hold: %s, tn: %s, err: %v\n", h.PositionHash, htn, err)
					// todo: add retries
					continue
				}
				h.ID = id
				dbHold = h
			} else {
				dbHold.Won += h.Won
				dbHold.Lost += h.Lost
				if err := srv.UpdateHold(htn, dbHold); err != nil {
					fmt.Printf("failed to update hold: %s, tn: %s, err: %v\n", h.PositionHash, htn, err)
					// todo: add retries
					continue
				}
			}

			// update sends
			for _, s := range allS {
				s.HoldsID = dbHold.ID
				stn := tn + "_sends"
				dbSend, err := srv.FindSend(stn, s.HoldsID, s.Sends)
				if err != nil {
					fmt.Printf("failed to find send: %s, tn: %s, err: %v\n", s.Sends, stn, err)
					// todo: add retries
					continue
				}
				if dbSend == nil {
					_, err := srv.InsertSend(stn, s)
					if err != nil {
						fmt.Printf("failed to insert send: %s, tn: %s, err: %v\n", s.Sends, stn, err)
						// todo: add retries
						continue
					}
				} else {
					dbSend.Held += s.Held
					dbSend.Leaked += s.Leaked
					dbSend.LeakedAmount += s.LeakedAmount

					err := srv.UpdateSend(stn, dbSend)
					if err != nil {
						fmt.Printf("failed to update send: %s, tn: %s, err: %v\n", s.Sends, stn, err)
						// todo: add retries
						continue
					}
				}
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
	// find the biggest unit
	for _, u := range player.BuildPerWave[index] {
		id := strings.Split(u, ":")[0]
		existing, ok := allUnits[id]
		if !ok {
			return anls, errors.New(fmt.Sprintf("couldn't find unit in unit map: %s", id))
		}
		if expCost < existing.TotalValue {
			expCost = existing.TotalValue
			anls.biggestUnitID = existing.UnitID
			anls.biggestUnitPos = u
		}
	}
	// hash based on the biggest unit
	diff, err := strconv.ParseFloat(strings.Split(strings.Split(anls.biggestUnitPos, ":")[1], "|")[1], 64)
	if err != nil {
		return anls, err
	}
	for _, u := range player.BuildPerWave[index] {
		hash, org, err := adjustUnit(u, diff, allUnits)
		if err != nil {
			return anls, err
		}
		anls.positionHash += hash + ","
		anls.position += org + ","
	}

	anls.TotalValue += player.ValuePerWave[index]
	anls.positionHash = strings.TrimSuffix(anls.positionHash, ",")
	anls.position = strings.TrimSuffix(anls.position, ",")
	if anls.biggestUnitID == "" {
		return anls, errors.New("failed to compute most expensive unit")
	}

	sort.Strings(player.MercenariesReceivedPerWave[index])
	anls.sendHash = strings.Join(player.MercenariesReceivedPerWave[index], ",")

	return anls, nil
}

func adjustUnit(u string, diff float64, allUnits map[string]*unit.Unit) (string, string, error) {
	build := strings.Split(u, ":")
	pos := strings.Split(build[1], "|")
	y, err := strconv.ParseFloat(pos[1], 64)
	if err != nil {
		return "", "", err
	}
	adjusted := math.Round((y-diff)*10) / 10
	adjStr := fmt.Sprintf("%v:%s|%v:%s", allUnits[build[0]].ID, pos[0], adjusted, build[2])
	orgStr := fmt.Sprintf("%v:%s:%s", allUnits[build[0]].ID, build[1], build[2])
	return adjStr, orgStr, nil
}

package main

import (
	"errors"
	"fmt"
	"math"
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

const waves = 8
const workers = 1
const usable_value = 60

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
	if err := generateHistoricalData(srv); err != nil {
		fmt.Printf("failed to generate historical: %v\n", err)
	}
	fmt.Println(time.Now().Format("Mon Jan _2 15:04:05 2006") + ": finished historical generation")

	totalTime := start.Sub(time.Now())
	hours := math.Floor(totalTime.Hours())
	minutes := math.Floor(totalTime.Minutes()) - hours*60
	fmt.Printf("total processing time: %vh %vm\n", hours, minutes)
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
			if _, ok := savedUnits[u.UnitId]; !ok {
				if u.TotalValue == "" {
					return errors.New(fmt.Sprintf("got a unit with empty total value: %s", u.UnitId))
				}
				val, err := strconv.Atoi(u.TotalValue)
				if err != nil {
					return errors.New(fmt.Sprintf("failed to convert unit total value: %s", u.UnitId))
				}
				usable := val >= usable_value
				if u.UnitId == "hell_raiser_buffed_unit_id" {
					u.Name = "Hell Raiser (Tantrum)"
				}
				newUnit := unit.Unit{
					UnitID:     u.UnitId,
					Name:       u.Name,
					IconPath:   u.IconPath,
					TotalValue: val,
					Usable:     usable,
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

	if err := srv.RefreshTables(); err != nil {
		return err
	}

	today := time.Now().UTC()
	yesterday := today.Add(time.Hour * -24)
	dateStart := fmt.Sprintf("%d-%02d-%02d%%2000:00:00.000Z", yesterday.Year(), yesterday.Month(), yesterday.Day())
	dateEnd := fmt.Sprintf("%d-%02d-%02d%%2000:00:00.000Z", today.Year(), today.Month(), today.Day())
	games := make(chan ltdapi.Game, 500)
	errChan := make(chan error, 1)
	wg := &sync.WaitGroup{}
	limiter := time.Tick(60 * time.Millisecond)
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
	for g := range games {
		gamesProcessed++
		if gamesProcessed%1000 == 0 {
			fmt.Printf(time.Now().Format("Mon Jan _2 15:04:05 2006")+" processed %v games, rate: %v games per second \n", gamesProcessed, math.Ceil(float64(1000)/math.Abs(timeMarker.Sub(time.Now()).Seconds())*100)/100)
			timeMarker = time.Now()
		}
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
						fmt.Printf("failed to analyze board: %v\n", err)
						continue
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

					tn := util.GenerateUnitTableName(anls.biggestUnitID, i+1)
					htn := tn + "_holds"
					if !srv.Tables[htn] {
						continue
					}

					id := 0
					// throttle
					<-limiter
					h, err := srv.FindHold(htn, anls.positionHash)
					if err != nil {
						fmt.Printf("failed to find hold: %v\n", err)
						continue
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
						// throttle
						<-limiter
						id, err = srv.SaveHold(htn, h)
						if err != nil {
							fmt.Printf("failed to save hold: %v\n", err)
							continue
						}
					} else {
						id = h.ID
					}
					stn := tn + "_sends"
					// throttle
					<-limiter
					s, err := srv.FindSends(stn, id, anls.sendHash)
					if err != nil {
						fmt.Printf("failed to find send: %v\n", err)
						continue
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
						// throttle
						<-limiter
						err := srv.InsertSend(stn, newS)
						if err != nil {
							fmt.Printf("failed to insert send: %v\n", err)
							continue
						}
					} else {
						if leaked {
							s.Leaked += 1
						} else {
							s.Held += 1
						}
						// throttle
						<-limiter
						err := srv.UpdateSend(stn, s)
						if err != nil {
							fmt.Printf("failed to update send: %v\n", err)
							continue
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
			anls.biggestUnitID = existing.UnitID
			anls.biggestUnitPos = u
		}

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

	// analyse sends
	ajMyth := 0.0
	for _, m := range player.MercenariesReceivedPerWave[index] {
		val, ok := allMercs[m]
		if !ok {
			return anls, errors.New(fmt.Sprintf("failed to find merc: %s", m))
		}
		anls.TotalMythium += val.MythiumCost
		if val.IncomeBonus != 0 {
			ajMyth += float64(val.MythiumCost) * (float64(val.MythiumCost) / float64(val.IncomeBonus) * float64(3) / float64(10))
		} else {
			ajMyth += float64(val.MythiumCost)
		}

	}
	anls.adjustedValue = anls.TotalValue - int(math.Ceil(1.25*ajMyth))
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

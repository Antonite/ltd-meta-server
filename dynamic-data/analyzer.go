package dynamicdata

import (
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/antonite/ltd-meta-server/mercenary"
	"github.com/antonite/ltd-meta-server/util"
	"github.com/pkg/errors"
)

type Stats struct {
	ID           int
	Score        int
	Sends        []*Send
	Position     string
	Hash         string
	TotalValue   int
	Winrate      int
	VersionAdded string
	Workers      float64
}

type analysis struct {
	sends      []*Send
	bestScore  float64
	totalGames int
	hold       *Hold
}

func GetTopHolds(db *sql.DB, primary string, secondary string, allMercs map[string]*mercenary.Mercenary, wave int, version string, max int, dedupe bool, leakScaler float64) ([]*Stats, error) {
	bounties := make(map[int]float64)
	bounties[1] = 72
	bounties[2] = 84
	bounties[3] = 90
	bounties[4] = 96
	bounties[5] = 108
	bounties[6] = 114
	bounties[7] = 120
	bounties[8] = 132

	stats := []*Stats{}
	tn := util.GenerateUnitTableName(primary, wave)
	tns := tn + "_sends"
	tnh := tn + "_holds"
	sends, err := getTopSends(db, tns)
	if err != nil {
		return stats, err
	}

	analyses := make(map[int]*analysis)
	for _, s := range sends {
		if _, ok := analyses[s.HoldsID]; !ok {
			h, err := FindHoldByID(db, tnh, s.HoldsID)
			if h == nil || err != nil {
				return nil, errors.Wrapf(err, "couldn't get hold id: %v", s.HoldsID)
			}
			if h.VersionAdded != version || (secondary != "Any" && !containsUnit(h.Position, secondary)) {
				continue
			}

			analyses[s.HoldsID] = &analysis{hold: h, bestScore: -300}
		}

		analyses[s.HoldsID].sends = append(analyses[s.HoldsID].sends, s)
		leakRate := 0.0
		if s.Leaked > 0 {
			leakRate = (float64(s.LeakedAmount) / float64(s.Leaked) / bounties[wave])
		}

		// analyse sends
		ajMyth := 0.0
		sp := strings.Split(s.Sends, ",")
		if sp[0] != "" {
			for _, m := range strings.Split(s.Sends, ",") {
				val, ok := allMercs[m]
				if !ok {
					return stats, errors.New(fmt.Sprintf("failed to find merc: %s", m))
				}
				s.TotalMythium += val.MythiumCost
				if val.IncomeBonus != 0 {
					ajMyth += float64(val.MythiumCost) * (float64(val.MythiumCost) / float64(val.IncomeBonus) * float64(3) / float64(10))
				} else {
					ajMyth += float64(val.MythiumCost)
				}
			}
		}

		goldLost := (1.0 - ((float64(s.Held) + float64(s.Leaked)*(1-leakRate)) / float64(s.Held+s.Leaked))) * bounties[wave] * leakScaler
		holdScore := (ajMyth * 1.25) - goldLost
		if holdScore > analyses[s.HoldsID].bestScore {
			analyses[s.HoldsID].bestScore = holdScore
		}

		s.LeakedRatio = int(math.Floor(leakRate * 100))
		analyses[s.HoldsID].totalGames += s.Held + s.Leaked
	}

	for k, v := range analyses {
		sortedSends := v.sends
		sort.Slice(sortedSends, func(i, j int) bool {
			return sortedSends[i].TotalMythium < sortedSends[j].TotalMythium
		})
		// egg case
		if primary == "eggsack_unit_id" {
			anyHeld := false
			for _, ss := range sortedSends {
				if ss.Held > 0 {
					anyHeld = true
					break
				}
			}
			if !anyHeld {
				continue
			}
		}
		stat := Stats{
			Score:   v.hold.TotalValue - int(math.Floor(v.bestScore)),
			Sends:   sortedSends,
			ID:      k,
			Winrate: int(math.Floor((float64(v.hold.Won) / float64(v.totalGames)) * 100)),
			Workers: math.Floor((float64(v.hold.Workers)/float64(v.totalGames))*10) / 10,
		}
		stats = append(stats, &stat)
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Score < stats[j].Score
	})

	dupes := make(map[string]*Stats)
	count := 0
	for _, s := range stats {
		h := analyses[s.ID].hold
		key := collapseUnits(h.Position)
		if dedupe {
			if v, ok := dupes[key]; ok {
				if v.Winrate > s.Winrate || v.Score < s.Score {
					continue
				}
			}
			dupes[key] = s
		} else {
			dupes[h.PositionHash] = s
		}
		count++

		s.Position = h.Position
		s.TotalValue = h.TotalValue
		s.VersionAdded = h.VersionAdded
		s.Hash = h.PositionHash
		if count >= max {
			break
		}
	}

	output := []*Stats{}

	for _, v := range dupes {
		output = append(output, v)
	}

	sort.Slice(output, func(i, j int) bool {
		return output[i].Score < output[j].Score
	})

	return output, nil
}

func collapseUnits(units string) string {
	used := []string{}
	l := strings.Split(units, ",")
	for _, u := range l {
		used = append(used, strings.Split(u, ":")[0])
	}
	sort.Strings(used)
	return strings.Join(used, ",")
}

func containsUnit(position string, unit string) bool {
	l := strings.Split(position, ",")
	for _, p := range l {
		u := strings.Split(p, ":")[0]
		if u == unit {
			return true
		}
	}

	return false
}

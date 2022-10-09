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
	TotalValue   int
	Winrate      int
	VersionAdded string
}

type analysis struct {
	sends      []*Send
	bestScore  float64
	totalGames int
	hold       *Hold
}

func GetTopHolds(db *sql.DB, primary string, secondary string, allMercs map[string]*mercenary.Mercenary, wave int, version string) ([]*Stats, error) {
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

			analyses[s.HoldsID] = &analysis{hold: h}
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

		holdScore := ajMyth * ((float64(s.Held) + float64(s.Leaked)*(1-leakRate)) / float64(s.Held+s.Leaked))
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
			Score: v.hold.TotalValue - int(math.Floor(v.bestScore*1.25)),
			Sends: sortedSends,
			ID:    k,
		}
		stats = append(stats, &stat)
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Score < stats[j].Score
	})

	dupes := make(map[string]*Stats)
	count := 0
	for _, s := range stats {
		h, err := FindHoldByID(db, tnh, s.ID)
		if h == nil || err != nil {
			return nil, errors.Wrapf(err, "couldn't get hold id: %v", s.ID)
		}
		key := collapseUnits(h.Position)
		if v, ok := dupes[key]; ok {
			if v.Winrate > s.Winrate || v.Score < s.Score {
				continue
			}
		} else {
			count++
		}
		dupes[key] = s
		s.Position = h.Position
		s.TotalValue = h.TotalValue
		s.VersionAdded = h.VersionAdded
		if count >= 20 {
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

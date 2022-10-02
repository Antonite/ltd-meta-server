package dynamicdata

import (
	"database/sql"
	"math"
	"sort"
	"strings"

	"github.com/antonite/ltd-meta-server/util"
	"github.com/pkg/errors"
)

type Stats struct {
	ID           int
	Score        int
	Sends        []*Send
	Position     string
	TotalValue   int
	VersionAdded string
}

func GetTopHolds(db *sql.DB, id string, wave int) ([]Stats, error) {
	bounties := make(map[int]float64)
	bounties[1] = 72
	bounties[2] = 84
	bounties[3] = 90
	bounties[4] = 96
	bounties[5] = 108

	stats := []Stats{}
	tn := util.GenerateUnitTableName(id, wave)
	tns := tn + "_sends"
	tnh := tn + "_holds"
	sends, err := getTopSends(db, tns)
	if err != nil {
		return stats, err
	}

	mappedSends := make(map[int][]*Send)
	mappedScores := make(map[int]float64)
	smallestScores := make(map[int]float64)
	totalgames := make(map[int]int)
	for _, s := range sends {
		if _, ok := mappedSends[s.HoldsID]; !ok {
			mappedSends[s.HoldsID] = []*Send{}
		}

		mappedSends[s.HoldsID] = append(mappedSends[s.HoldsID], s)
		score := float64(s.AdjustedValue) * (1 + ((1 - (float64(s.Held) / float64(s.Held+s.Leaked))) * 4 * (float64(s.LeakedAmount) / bounties[wave])))
		totalgames[s.HoldsID] += s.Held + s.Leaked
		mappedScores[s.HoldsID] += score
		if (smallestScores[s.HoldsID] > score || smallestScores[s.HoldsID] == 0) && s.Held != 0 {
			smallestScores[s.HoldsID] = score
		}
	}

	for k, v := range mappedScores {
		if smallestScores[k] == 0 {
			continue
		}
		// focus on the best hold
		v += smallestScores[k] * 5
		l := len(mappedSends[k])

		// average
		score := v / float64(l+5)
		// punish for low varience in sends
		if l == 1 {
			score = score * 1.8
		} else if l == 2 {
			score = score * 1.5
		} else if l == 3 {
			score = score * 1.3
		} else if l == 4 {
			score = score * 1.2
		} else if l == 5 {
			score = score * 1.1
		}
		// punish for low number of games
		if totalgames[k] == 1 {
			score = score * 3
		} else if totalgames[k] < 10 {
			score = score * 2.5
		} else if totalgames[k] < 100 {
			score = score * (1 + (1 - float64(totalgames[k])/100))
		}
		sortedSends := mappedSends[k]
		sort.Slice(sortedSends, func(i, j int) bool {
			return sortedSends[i].TotalMythium < sortedSends[j].TotalMythium
		})
		stat := Stats{
			Score: int(math.Ceil(score)),
			Sends: sortedSends,
			ID:    k,
		}
		stats = append(stats, stat)
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Score < stats[j].Score
	})

	top200 := []Stats{}
	// get winrate for top 200
	max := 200
	if len(stats) < 200 {
		max = len(stats)
	}
	for i := 0; i < max; i++ {
		winrate := 0.5
		if totalgames[stats[i].ID] >= 10 {
			h, err := FindHoldByID(db, tnh, stats[i].ID)
			if h == nil || err != nil {
				return nil, errors.Wrapf(err, "couldn't get hold id: %v", stats[i].ID)
			}
			winrate = float64(h.Won) / (float64(h.Won) + float64(h.Lost))
		}

		stats[i].Score = int(math.Ceil((1 + 0.5*(1-winrate)) * float64(stats[i].Score)))
		top200 = append(top200, stats[i])
	}

	sort.Slice(top200, func(i, j int) bool {
		return top200[i].Score < top200[j].Score
	})

	dupes := make(map[string]bool)
	count := 0
	output := []Stats{}
	for _, s := range top200 {
		h, err := FindHoldByID(db, tnh, s.ID)
		if h == nil || err != nil {
			return nil, errors.Wrapf(err, "couldn't get hold id: %v", s.ID)
		}
		key := collapseUnits(h.Position)
		if _, ok := dupes[key]; ok {
			continue
		}
		dupes[key] = true
		count++
		s.Position = h.Position
		s.TotalValue = h.TotalValue
		s.VersionAdded = h.VersionAdded
		output = append(output, s)
		if count >= 20 {
			break
		}
	}

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

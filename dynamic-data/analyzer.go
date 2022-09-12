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
	totalgames := 0
	for _, s := range sends {
		if _, ok := mappedSends[s.HoldsID]; !ok {
			mappedSends[s.HoldsID] = []*Send{}
		}

		mappedSends[s.HoldsID] = append(mappedSends[s.HoldsID], s)
		score := float64(s.AdjustedValue)
		if s.Held != 0 {
			score = score * (1 + (1 - (float64(s.Held) / float64(s.Held+s.Leaked))))
		}
		totalgames += s.Held + s.Leaked
		mappedScores[s.HoldsID] += score
	}

	for k, v := range mappedScores {
		l := len(mappedSends[k])
		// average
		score := v / float64(l)
		// punish for low varience in sends
		if l == 1 {
			score = score * 1.2
		} else if l == 2 {
			score = score * 1.1
		} else if l == 3 {
			score = score * 1.05
		}
		// punish for low number of games
		if totalgames < 3 {
			score = score * 1.2
		} else if totalgames < 10 {
			score = score * 1.1
		} else if totalgames < 50 {
			score = score * 1.05
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
	dupes := make(map[string]bool)
	count := 0
	output := []Stats{}
	for _, s := range stats {
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

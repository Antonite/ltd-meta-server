package guide

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	dynamicdata "github.com/antonite/ltd-meta-server/dynamic-data"
)

const Waves = 7

type Guide struct {
	MainUnitID      int
	SecondaryUnitID int
	Score           int
	Winrate         int
	Workers         float64
	Waves           []WaveGuide
	Mastermind      string
}

type WaveGuide struct {
	Position     string
	PositionHash string
	Value        int
	Score        int
	Winrate      int
	Sends        []*dynamicdata.Send
	Workers      float64
}

func GenerateGuides(uid int, smap map[int]map[int][]*dynamicdata.Stats, upgrades map[string][]string, specials []string) []Guide {
	wGuides := make(map[int]WaveGuide, Waves)
	return guideHelper(uid, 1, wGuides, smap, upgrades, specials)
}

func guideHelper(uid int, wave int, guides map[int]WaveGuide, smap map[int]map[int][]*dynamicdata.Stats, upgrades map[string][]string, specials []string) []Guide {
	if wave > Waves {
		score := 0
		winrate := 0
		workers := 0.0
		waves := []WaveGuide{}
		for i := 1; i <= Waves; i++ {
			scaler := 1.0
			if i == 1 {
				scaler = 4
			} else if i == 2 {
				scaler = 3
			} else if i == 3 {
				scaler = 2
			} else if i == 4 {
				scaler = 1.5
			} else if i == 5 {
				scaler = 1.25
			}
			score += int(math.Floor(float64(guides[i].Score) * scaler))
			winrate += guides[i].Winrate
			workers += guides[i].Workers
			waves = append(waves, guides[i])
		}
		winrate = int(math.Floor(float64(winrate) / float64(len(guides))))
		workers = math.Floor((workers/float64(len(guides)))*10) / 10
		guide := Guide{
			Score:   score,
			Winrate: winrate,
			Waves:   waves,
			Workers: workers,
		}
		return []Guide{guide}
	}
	gout := []Guide{}
	if wave == 1 {
		for _, s := range smap[uid][wave] {
			wg := WaveGuide{
				Position:     s.Position,
				PositionHash: s.Hash,
				Value:        s.TotalValue,
				Score:        s.Score,
				Winrate:      s.Winrate,
				Sends:        s.Sends,
				Workers:      s.Workers,
			}
			guides[wave] = wg
			gout = append(gout, guideHelper(uid, wave+1, guides, smap, upgrades, specials)...)
		}
	} else {
		for _, v := range smap {
			for _, s := range v[wave] {
				if matchBuildHash(guides[wave-1].PositionHash, s.Hash, upgrades, specials) {
					wg := WaveGuide{
						Position:     s.Position,
						PositionHash: s.Hash,
						Value:        s.TotalValue,
						Score:        s.Score,
						Winrate:      s.Winrate,
						Sends:        s.Sends,
						Workers:      s.Workers,
					}
					guides[wave] = wg
					gout = append(gout, guideHelper(uid, wave+1, guides, smap, upgrades, specials)...)
				}
			}
		}
	}

	return gout
}

func matchBuildHash(firstHash string, secondHash string, upgrades map[string][]string, specials []string) bool {
	firstUnits := strings.Split(firstHash, ",")
	secondUnits := strings.Split(secondHash, ",")
	viableDiffs := make(map[float64]bool)
	init := true
	for _, fUnit := range firstUnits {
		fparts := strings.Split(fUnit, ":")
		contains := false
		for _, sUnit := range secondUnits {
			sparts := strings.Split(sUnit, ":")
			unitMatch := false
			// are the units the same?
			if fparts[0] == sparts[0] {
				unitMatch = true
			}
			// how about upgrades of first unit
			for _, upg := range upgrades[fparts[0]] {
				if upg == fparts[0] {
					unitMatch = true
				}
			}
			// maybe upgrades are reversed (can happen in special 'upgrade' cases like pack rat)
			for _, upg := range upgrades[sparts[0]] {
				if upg == sparts[0] {
					unitMatch = true
				}
			}
			if !unitMatch {
				continue
			}
			// special cat and sakura case
			for _, sp := range specials {
				if fparts[0] == sp {
					unitMatch = false
				}
			}
			// rematch unit by stacks
			if !unitMatch && fparts[2] != sparts[2] {
				continue
			}

			fpos := strings.Split(fparts[1], "|")
			spos := strings.Split(sparts[1], "|")
			// is the x value positions the same
			if fpos[0] != spos[0] {
				continue
			}

			// calculate y diff
			fy, ferr := strconv.ParseFloat(fpos[1], 64)
			sy, serr := strconv.ParseFloat(spos[1], 64)
			if ferr != nil || serr != nil {
				fmt.Printf("error converting position to int, %v, %v", ferr, serr)
			}
			diff := fy - sy
			if init {
				viableDiffs[diff] = true
				contains = true
			} else {
				if !viableDiffs[diff] {
					continue
				}
				contains = true
			}
		}
		init = false
		if !contains {
			return false
		}
	}
	return true
}

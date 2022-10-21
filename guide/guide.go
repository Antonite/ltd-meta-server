package guide

import (
	"math"
	"strings"

	dynamicdata "github.com/antonite/ltd-meta-server/dynamic-data"
)

const Waves = 7

type Guide struct {
	MainUnitID      int
	SecondaryUnitID int
	Score           int
	Winrate         int
	Waves           []WaveGuide
	Mastermind      string
}

type WaveGuide struct {
	Position     string
	PositionHash string
	Value        int
	Score        int
	Winrate      int
}

func GenerateGuides(smap map[int][]*dynamicdata.Stats, upgrades map[string][]string) []Guide {
	wGuides := make(map[int]WaveGuide, Waves)
	return guideHelper(1, wGuides, smap, upgrades)
}

func guideHelper(wave int, guides map[int]WaveGuide, smap map[int][]*dynamicdata.Stats, upgrades map[string][]string) []Guide {
	if wave > Waves {
		score := 0
		winrate := 0
		waves := []WaveGuide{}
		for i := 1; i <= Waves; i++ {
			score += guides[i].Score
			winrate += guides[i].Winrate
			waves = append(waves, guides[i])
		}
		winrate = int(math.Floor(float64(winrate) / float64(len(guides))))
		guide := Guide{
			Score:   score,
			Winrate: winrate,
			Waves:   waves,
		}
		return []Guide{guide}
	}
	gout := []Guide{}
	for _, s := range smap[wave] {
		if wave == 1 || matchBuildHash(guides[wave-1].PositionHash, s.Hash, upgrades) {
			wg := WaveGuide{
				Position:     s.Position,
				PositionHash: s.Hash,
				Value:        s.TotalValue,
				Score:        s.Score,
				Winrate:      s.Winrate,
			}
			guides[wave] = wg
			gout = append(gout, guideHelper(wave+1, guides, smap, upgrades)...)
		}
	}
	return gout
}

func matchBuildHash(firstHash string, secondHash string, upgrades map[string][]string) bool {
	firstUnits := strings.Split(firstHash, ",")
	secondUnits := strings.Split(secondHash, ",")
	for _, fUnit := range firstUnits {
		fparts := strings.Split(fUnit, ":")
		contains := false
		for _, sUnit := range secondUnits {
			sparts := strings.Split(sUnit, ":")
			if fparts[1] != sparts[1] {
				continue
			}

			if fparts[0] == sparts[0] {
				contains = true
				break
			}
			for _, upg := range upgrades[fparts[0]] {
				if upg == fparts[0] {
					contains = true
					break
				}
			}
			for _, upg := range upgrades[sparts[0]] {
				if upg == sparts[0] {
					contains = true
					break
				}
			}

		}
		if !contains {
			return false
		}
	}
	return true
}

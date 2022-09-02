package util

import (
	"fmt"
	"strings"
)

const CashoutGold = 273

var (
	specialUnits []string = []string{"hell_raiser_buffed_unit_id", "pack_rat_nest_unit_id"}
	Eggsack      string   = "eggsack_unit_id"
	Hydra        string   = "hydra_unit_id"
)

func IsSpecialUnit(u string) bool {
	for _, sp := range specialUnits {
		if sp == u {
			return true
		}
	}

	return false
}

func IsFullHydra(build []string, pos string) bool {
	cords := strings.Split(pos, ":")[1]
	for _, u := range build {
		c := strings.Split(u, ":")
		if c[1] == cords {
			if c[0] != Hydra && c[0] != Eggsack {
				fmt.Printf("egg didn't turn into hydra: %v\n", build)
			}
			return c[2] == "0"
		}
	}

	fmt.Printf("couldn't find the hydra: %v\n", build)
	return false
}

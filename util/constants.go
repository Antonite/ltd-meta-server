package util

var specialUnits = []string{"hell_raiser_buffed_unit_id", "pack_rat_nest_unit_id"}

func IsSpecialUnit(u string) bool {
	for _, sp := range specialUnits {
		if sp == u {
			return true
		}
	}

	return false
}

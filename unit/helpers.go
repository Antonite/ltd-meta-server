package unit

import (
	"errors"
	"fmt"
	"strings"
)

func MostExpensive(build []string, allUnits map[string]*Unit) (string, int, error) {
	expCost := 0
	totalCost := 0
	mostExpensive := ""
	for _, u := range build {
		id := strings.Split(u, ":")[0]
		existing, ok := allUnits[id]
		if !ok {
			return "", 0, errors.New(fmt.Sprintf("couldn't find unit in unit map: %s", id))
		}
		if expCost < existing.TotalValue {
			expCost = existing.TotalValue
			mostExpensive = existing.ID
		}
		totalCost += existing.TotalValue
	}
	if mostExpensive == "" {
		return "", 0, errors.New("failed to compute most expensive unit")
	}
	return mostExpensive, totalCost, nil
}

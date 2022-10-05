package util

import (
	"fmt"
	"strings"
)

func GenerateUnitTableName(u string, w int) string {
	name := strings.TrimSuffix(u, "unit_id")
	name = fmt.Sprintf("%swave_%v", name, w)
	return name
}

func NormalizeVersion(v string) string {
	t := strings.Split(strings.TrimPrefix(v, "v"), ".")
	return t[0] + "." + t[1]
}

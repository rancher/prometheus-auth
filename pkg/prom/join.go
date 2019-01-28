package prom

import (
	"strings"
)

func stringSliceIgnore(strSlice []string, ignore *string) []string {
	return stringSliceFilter(strSlice, func(value *string) bool {
		return *ignore != *value
	})
}

func stringSliceFilter(strSlice []string, filter func(value *string) bool) []string {
	matchNss := make([]string, 0, len(strSlice))
	for _, ns := range strSlice {
		if filter(&ns) {
			matchNss = append(matchNss, ns)
		}
	}

	return matchNss
}

func join(strSlice []string) string {
	return strings.Join(strSlice, "|")
}

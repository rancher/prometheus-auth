package data

import (
	"sort"
	"strings"
)

type Set map[string]struct{}

func (s Set) Values() []string {
	size := len(s)

	ret := make([]string, 0, size)
	if size == 0 {
		return ret
	}

	for key := range s {
		if key == "" {
			continue
		}

		ret = append(ret, key)
	}
	if size > 1 {
		sort.Strings(ret)
	}

	return ret
}

func (s Set) String() string {
	return strings.Join(s.Values(), ",")
}

func NewSet(values ...string) Set {
	ret := make(Set, len(values))
	for _, val := range values {
		ret[val] = struct{}{}
	}

	return ret
}

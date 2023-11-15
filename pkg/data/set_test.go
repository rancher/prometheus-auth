//go:build test

package data

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestSet(t *testing.T) {
	expert := []string{
		"a", "B", "c", "D", "f",
	}

	input := append(expert, expert...)

	s := NewSet()

	for _, e := range input {
		s[e] = struct{}{}
	}

	// not in
	_, exist := s["g"]
	if exist {
		t.Errorf("%v doesn't have g", expert)
	}

	// in
	_, exist = s["a"]
	if !exist {
		t.Errorf("%v has a", expert)
	}

	// len
	if len(s) != len(expert) {
		t.Errorf("the len of %v is %d", expert, len(expert))
	}

	// add an blank key
	s[""] = struct{}{}

	sort.Strings(expert)
	values := s.Values()
	if !reflect.DeepEqual(s.Values(), expert) {
		t.Errorf("expect %v, but got %v", expert, values)
	}

	expertJoins := strings.Join(expert, ",")
	str := s.String()
	if s.String() != expertJoins {
		t.Errorf("expect %v, but got %v", expertJoins, str)
	}
}

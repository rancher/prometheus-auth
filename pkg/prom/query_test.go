//go:build test

package prom

import (
	"fmt"
	"testing"
)

func TestNewExprForCountAllLabels(t *testing.T) {
	cases := []struct {
		input  []string
		expect string
	}{
		{
			[]string{"ns-a", "ns-b", "rx-c"},
			`count ({namespace=~"ns-a|ns-b|rx-c"}) by (__name__)`,
		},
		{
			[]string{},
			`count ({namespace="______"}) by (__name__)`,
		},
	}
	errs := make([]error, 0, len(metrics))

	for _, c := range cases {
		output := NewExprForCountAllLabels(c.input)
		if c.expect != output {
			errs = append(errs, fmt.Errorf("%s => %v, but get %v", c.input, c.expect, output))
		} else {
			fmt.Printf("[passed] %s => %v \n", c.input, output)
		}
	}

	if len(errs) != 0 {
		for _, err := range errs {
			t.Log(err)
		}

		t.Fail()
	}
}

func TestNewInstantVectorSelectorsForNamespace(t *testing.T) {
	cases := []struct {
		input  []string
		expect string
	}{
		{
			[]string{"ns-a", "ns-b", "rx-c"},
			`{namespace=~"ns-a|ns-b|rx-c"}`,
		},
		{
			[]string{},
			`{namespace="______"}`,
		},
	}
	errs := make([]error, 0, len(metrics))

	for _, c := range cases {
		output := NewInstantVectorSelectorsForNamespaces(c.input)
		if c.expect != output {
			errs = append(errs, fmt.Errorf("%s => %v, but get %v", c.input, c.expect, output))
		} else {
			fmt.Printf("[passed] %s => %v \n", c.input, output)
		}
	}

	if len(errs) != 0 {
		for _, err := range errs {
			t.Log(err)
		}

		t.Fail()
	}
}

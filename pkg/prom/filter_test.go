//go:build test

package prom

import (
	"fmt"
	"testing"

	"github.com/juju/errors"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/promql"
	"github.com/rancher/prometheus-auth/pkg/data"
)

var metrics = []struct {
	name   string
	input  string
	expect string
}{
	{
		"not label",
		`a`,
		`a{namespace=~"ns-a|ns-b|rx-c"}`,
	},
	{
		"none namespace label",
		`a{value="value"}`,
		`a{namespace=~"ns-a|ns-b|rx-c",value="value"}`,
	},
	{
		"= without value hitting",
		`a{namespace="ns-x"}`,
		`a{namespace="______"}`,
	},
	{
		"= with value hitting",
		`a{namespace="ns-a"}`,
		`a{namespace="ns-a"}`,
	},
	{
		"!= without value hitting",
		`a{namespace!="ns-x"}`,
		`a{namespace=~"ns-a|ns-b|rx-c"}`,
	},
	{
		"!= with value hitting",
		`a{namespace!="ns-a"}`,
		`a{namespace=~"ns-b|rx-c"}`,
	},
	{
		"=~ without value hitting",
		`a{namespace=~"ns-x"}`,
		`a{namespace="______"}`,
	},
	{
		"=~ with value hitting",
		`a{namespace=~"ns-a"}`,
		`a{namespace="ns-a"}`,
	},
	{
		"=~ with regex value (match)",
		`a{namespace=~"n.*"}`,
		`a{namespace=~"ns-a|ns-b"}`,
	},
	{
		"=~ with regex value (match)",
		`a{namespace=~"^.*-.*$"}`,
		`a{namespace=~"ns-a|ns-b|rx-c"}`,
	},
	{
		"=~ with regex value (not match)",
		`a{namespace=~"t.*"}`,
		`a{namespace="______"}`,
	},
	{
		"=~ with regex value (not match)",
		`a{namespace=~""}`,
		`a{namespace="______"}`,
	},
	{
		"!~ without value hitting",
		`a{namespace!~"ns-x"}`,
		`a{namespace=~"ns-a|ns-b|rx-c"}`,
	},
	{
		"!~ with value hitting",
		`a{namespace!~"ns-a"}`,
		`a{namespace=~"ns-b|rx-c"}`,
	},
	{
		"!~ with regex value (match)",
		`a{namespace!~"n.*"}`,
		`a{namespace="rx-c"}`,
	},
	{
		"!~ with regex value (match)",
		`a{namespace!~"^.*-.*$"}`,
		`a{namespace="______"}`,
	},
	{
		"!~ with regex value (not match)",
		`a{namespace!~"t.*"}`,
		`a{namespace=~"ns-a|ns-b|rx-c"}`,
	},
	{
		"=~ with regex value (not match)",
		`a{namespace!~""}`,
		`a{namespace=~"ns-a|ns-b|rx-c"}`,
	},
}

func fakeNamespaceSet() data.Set {
	return data.NewSet("ns-a", "ns-b", "rx-c")
}

func TestFilterMatchers(t *testing.T) {
	nsSet := fakeNamespaceSet()
	errs := make([]error, 0, len(metrics))

	for _, c := range metrics {
		err := walkExpr(c.name, c.input, c.expect, func(matchers []*labels.Matcher) ([]*labels.Matcher, error) {
			return FilterMatchers(nsSet, matchers), nil
		})
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		for _, err := range errs {
			t.Log(err)
		}

		t.Fail()
	}
}

func TestFilterLabelMatchers(t *testing.T) {
	nsSet := fakeNamespaceSet()
	errs := make([]error, 0, len(metrics))

	for _, c := range metrics {
		err := walkExpr(c.name, c.input, c.expect, func(matchers []*labels.Matcher) ([]*labels.Matcher, error) {
			lm, err := toLabelMatchers(matchers)
			if err != nil {
				return nil, err
			}

			return fromLabelMatchers(FilterLabelMatchers(nsSet, lm))
		})
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		for _, err := range errs {
			t.Log(err)
		}

		t.Fail()
	}
}

func walkExpr(name, input, expect string, change func([]*labels.Matcher) ([]*labels.Matcher, error)) error {
	promlbInputExpr, err := promql.ParseExpr(input)
	if err != nil {
		return errors.Annotatef(err, "%s cannot parse expr from %s", name, input)
	}

	promql.Inspect(promlbInputExpr, func(node promql.Node, _ []promql.Node) error {
		switch n := node.(type) {
		case *promql.VectorSelector:
			ret, err := change(n.LabelMatchers)
			if err != nil {
				return errors.Annotatef(err, "%s causes error", input)
			}

			n.LabelMatchers = ret
		case *promql.MatrixSelector:
			ret, err := change(n.LabelMatchers)
			if err != nil {
				return errors.Annotatef(err, "%s causes error", input)
			}

			n.LabelMatchers = ret
		}
		return nil
	})

	output := promlbInputExpr.String()
	if expect != output {
		return fmt.Errorf("%s => %v, but get %v", input, expect, output)
	}

	fmt.Printf("[passed] %s => %v \n", input, output)
	return nil
}

package prom

import (
	"github.com/caas-team/prometheus-auth/pkg/data"
	promlb "github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/prompb"
)

const (
	namespaceMatchName = "namespace"
)

func FilterMatchers(namespaceSet data.Set, srcMatchers []*promlb.Matcher) []*promlb.Matcher {
	for _, m := range srcMatchers {
		name := m.Name

		if name == namespaceMatchName {
			translateMatcher(namespaceSet, m)
			return srcMatchers
		}
	}

	// append namespace match
	srcMatchers = append(srcMatchers, createMatcher(namespaceMatchName, namespaceSet.Values()))

	return srcMatchers
}

func FilterLabelMatchers(namespaceSet data.Set, srcMatchers []*prompb.LabelMatcher) []*prompb.LabelMatcher {
	for _, m := range srcMatchers {
		name := m.Name

		if name == namespaceMatchName {
			translateLabelMatcher(namespaceSet, m)
			return srcMatchers
		}
	}

	// append caas clustermetrics label
	//caas := []prompb.LabelMatcher{prompb.LabelMatcher{Name: "caas.telekom.de/clustermetric", Value: "entsoe", Type: prompb.LabelMatcher_EQ}}

	// caas := []promlb.Label{"caas.telekom.de/clustermetric","entsoe"}
	// append namespace match
	srcMatchers = append(srcMatchers, createLabelMatcher(namespaceMatchName, namespaceSet.Values()), &prompb.LabelMatcher{Name: "caas.telekom.de/clustermetric", Value: "entsoe"})

	return srcMatchers
}

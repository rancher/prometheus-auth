package prom

import (
	"fmt"
	"regexp"

	promlb "github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/prompb"
	"github.com/rancher/prometheus-auth/pkg/data"
)

func translateMatcher(namespaceSet data.Set, srcMatcher *promlb.Matcher) {
	if namespaceSet == nil || srcMatcher == nil {
		return
	}

	value := srcMatcher.Value
	switch srcMatcher.Type {
	case promlb.MatchEqual: // =
		if _, exist := namespaceSet[value]; !exist {
			srcMatcher.Value = noneNamespace
		}
	case promlb.MatchNotEqual: // !=
		namespaces := namespaceSet.Values()
		if _, exist := namespaceSet[value]; exist {
			namespaces = stringSliceIgnore(namespaces, &value)
		}

		modifyMatcher(srcMatcher, namespaces)
	case promlb.MatchRegexp: // =~
		namespaces := stringSliceFilter(namespaceSet.Values(), func(ns *string) bool {
			return srcMatcher.Matches(*ns)
		})

		modifyMatcher(srcMatcher, namespaces)
	case promlb.MatchNotRegexp: // !~
		namespaces := stringSliceFilter(namespaceSet.Values(), func(ns *string) bool {
			return srcMatcher.Matches(*ns)
		})

		modifyMatcher(srcMatcher, namespaces)
	}
}

func translateLabelMatcher(namespaceSet data.Set, srcMatcher *prompb.LabelMatcher) {
	if namespaceSet == nil || srcMatcher == nil {
		return
	}

	value := srcMatcher.Value
	switch srcMatcher.Type {
	case prompb.LabelMatcher_EQ: // =
		if _, exist := namespaceSet[value]; !exist {
			srcMatcher.Value = noneNamespace
		}
	case prompb.LabelMatcher_NEQ: // !=
		namespaces := namespaceSet.Values()
		if _, exist := namespaceSet[value]; exist {
			namespaces = stringSliceIgnore(namespaces, &value)
		}

		modifyLabelMatcher(srcMatcher, namespaces)
	case prompb.LabelMatcher_RE: // =~
		valueRegexp, err := regexp.Compile(fmt.Sprintf("^(?:%s)$", value))
		if err == nil {
			namespaces := stringSliceFilter(namespaceSet.Values(), func(ns *string) bool {
				return valueRegexp.MatchString(*ns)
			})

			modifyLabelMatcher(srcMatcher, namespaces)
		}
	case prompb.LabelMatcher_NRE: // !~
		valueRegexp, err := regexp.Compile(fmt.Sprintf("^(?:%s)$", value))
		if err == nil {
			namespaces := stringSliceFilter(namespaceSet.Values(), func(ns *string) bool {
				return !valueRegexp.MatchString(*ns)
			})

			modifyLabelMatcher(srcMatcher, namespaces)
		}
	}
}

package prommetric

import (
	promlb "github.com/prometheus/prometheus/pkg/labels"
	"github.com/rancher/prometheus-auth/pkg/kubeauth/view"
)

type LabelMatcherTranslator interface {
	Translate(srcMatcher *promlb.Matcher) *promlb.Matcher
}

type labelMatcherTranslator struct {
	labelMatcherName string
	set              view.SetView
}

func (on *labelMatcherTranslator) Translate(srcMatcher *promlb.Matcher) *promlb.Matcher {
	if srcMatcher == nil {
		return &promlb.Matcher{
			Type:  promlb.MatchRegexp,
			Name:  on.labelMatcherName,
			Value: on.joinAll(),
		}
	}

	value := srcMatcher.Value
	switch srcMatcher.Type {
	case promlb.MatchEqual: // =
		if owned := on.set.Has(value); !owned {
			srcMatcher.Value = noneNamespace
		}
	case promlb.MatchNotEqual: // !=
		srcMatcher.Type = promlb.MatchRegexp
		if owned := on.set.Has(value); owned {
			srcMatcher.Value = on.joinIgnore(&value)
		} else {
			srcMatcher.Value = on.joinAll()
		}
	case promlb.MatchRegexp: // =~
		srcMatcher.Value = on.joinFilter(func(ns *string) bool {
			return srcMatcher.Matches(*ns)
		})
	case promlb.MatchNotRegexp: // !~
		srcMatcher.Value = on.joinFilter(func(ns *string) bool {
			return srcMatcher.Matches(*ns)
		})
		srcMatcher.Type = promlb.MatchRegexp
	}

	return srcMatcher
}

func (on *labelMatcherTranslator) joinAll() string {
	return joins(on.set.GetAll())
}

func (on *labelMatcherTranslator) joinIgnore(ignore *string) string {
	return on.joinFilter(func(value *string) bool {
		return *ignore != *value
	})
}

func (on *labelMatcherTranslator) joinFilter(filter func(value *string) bool) string {
	allNs := on.set.GetAll()

	matchNss := make([]string, 0, len(allNs))
	for _, ns := range allNs {
		if filter(&ns) {
			matchNss = append(matchNss, ns)
		}
	}

	return joins(matchNss)
}

func CreateLabelMatcherTranslator(labelMatcherName string, set view.SetView) LabelMatcherTranslator {
	return &labelMatcherTranslator{
		labelMatcherName: labelMatcherName,
		set:              set,
	}
}

package prommetric

import (
	"fmt"
	"github.com/prometheus/prometheus/prompb"
	"github.com/rancher/prometheus-auth/pkg/kubeauth/view"
	"regexp"
)

type PromPbLabelMatcherTranslator interface {
	Translate(srcMatcher *prompb.LabelMatcher) *prompb.LabelMatcher
}

type prompbLabelMatcherTranslator struct {
	labelMatcherName string
	set              view.SetView
}

func (on *prompbLabelMatcherTranslator) Translate(srcMatcher *prompb.LabelMatcher) *prompb.LabelMatcher {
	if srcMatcher == nil {
		return &prompb.LabelMatcher{
			Type:  prompb.LabelMatcher_RE,
			Name:  on.labelMatcherName,
			Value: on.joinAll(),
		}
	}

	value := srcMatcher.Value
	switch srcMatcher.Type {
	case prompb.LabelMatcher_EQ: // =
		if owned := on.set.Has(value); !owned {
			srcMatcher.Value = noneNamespace
		}
	case prompb.LabelMatcher_NEQ: // !=
		srcMatcher.Type = prompb.LabelMatcher_RE
		if owned := on.set.Has(value); owned {
			srcMatcher.Value = on.joinIgnore(&value)
		} else {
			srcMatcher.Value = on.joinAll()
		}
	case prompb.LabelMatcher_RE: // =~
		valueRegexp, err := regexp.Compile(fmt.Sprintf("^(?:%s)$", value))
		if err == nil {
			srcMatcher.Value = on.joinFilter(func(ns *string) bool {
				return valueRegexp.MatchString(*ns)
			})
		}
	case prompb.LabelMatcher_NRE: // !~
		srcMatcher.Type = prompb.LabelMatcher_RE
		valueRegexp, err := regexp.Compile(fmt.Sprintf("^(?:%s)$", value))
		if err == nil {
			srcMatcher.Value = on.joinFilter(func(ns *string) bool {
				return !valueRegexp.MatchString(*ns)
			})
		}
	}

	return srcMatcher
}

func (on *prompbLabelMatcherTranslator) joinAll() string {
	return joins(on.set.GetAll())
}

func (on *prompbLabelMatcherTranslator) joinIgnore(ignore *string) string {
	return on.joinFilter(func(value *string) bool {
		return *ignore != *value
	})
}

func (on *prompbLabelMatcherTranslator) joinFilter(filter func(value *string) bool) string {
	allNs := on.set.GetAll()

	matchNss := make([]string, 0, len(allNs))
	for _, ns := range allNs {
		if filter(&ns) {
			matchNss = append(matchNss, ns)
		}
	}

	return joins(matchNss)
}

func CreatePromPbLabelMatcherTranslator(labelMatcherName string, set view.SetView) PromPbLabelMatcherTranslator {
	return &prompbLabelMatcherTranslator{
		labelMatcherName: labelMatcherName,
		set:              set,
	}
}

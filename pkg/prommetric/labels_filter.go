package prommetric

import (
	promlb "github.com/prometheus/prometheus/pkg/labels"
)

type LabelMatchersFilter interface {
	Filter(srcMatchers []*promlb.Matcher) []*promlb.Matcher
}

type LabelMatchersDropper interface {
	Drop(srcMatchers []*promlb.Matcher) map[string]*promlb.Matcher
}

type LabelMatchersTranslator interface {
	Translate(srcMatchers map[string]*promlb.Matcher) []*promlb.Matcher
}

type LabelMatcherNameDropper map[string]struct{}

func (d LabelMatcherNameDropper) Drop(srcMatchers []*promlb.Matcher) map[string]*promlb.Matcher {
	ret := make(map[string]*promlb.Matcher, len(srcMatchers))
	for _, m := range srcMatchers {
		name := m.Name
		if d == nil {
			ret[name] = m
		} else if _, ignore := d[name]; !ignore {
			ret[name] = m
		}
	}

	return ret
}

type LabelMatcherNameTranslator map[string]LabelMatcherTranslator

func (d LabelMatcherNameTranslator) Translate(srcMatchers map[string]*promlb.Matcher) []*promlb.Matcher {
	if d != nil {
		for name, t := range d {
			if m, exist := srcMatchers[name]; exist {
				srcMatchers[name] = t.Translate(m)
			} else if r := t.Translate(nil); r != nil {
				srcMatchers[name] = r
			}
		}
	}

	ret := make([]*promlb.Matcher, 0, len(srcMatchers))
	for _, m := range srcMatchers {
		ret = append(ret, m)
	}

	return ret
}

type labelMatchersNameFilter struct {
	dropper    LabelMatchersDropper
	translator LabelMatchersTranslator
}

func (f *labelMatchersNameFilter) Filter(srcMatchers []*promlb.Matcher) []*promlb.Matcher {
	return f.translator.Translate(f.dropper.Drop(srcMatchers))
}

func CreateLabelMatchersNameFilter(dropper LabelMatcherNameDropper, translator LabelMatcherNameTranslator) LabelMatchersFilter {
	return &labelMatchersNameFilter{
		dropper:    dropper,
		translator: translator,
	}
}

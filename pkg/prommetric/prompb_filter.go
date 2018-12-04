package prommetric

import (
	"github.com/prometheus/prometheus/prompb"
)

type PromPbLabelMatchersFilter interface {
	Filter(srcMatchers []*prompb.LabelMatcher) []*prompb.LabelMatcher
}

type PromPbLabelMatchersDropper interface {
	Drop(srcMatchers []*prompb.LabelMatcher) map[string]*prompb.LabelMatcher
}

type PromPbLabelMatchersTranslator interface {
	Translate(srcMatchers map[string]*prompb.LabelMatcher) []*prompb.LabelMatcher
}

type PromPbLabelMatcherNameDropper map[string]struct{}

func (d PromPbLabelMatcherNameDropper) Drop(srcMatchers []*prompb.LabelMatcher) map[string]*prompb.LabelMatcher {
	ret := make(map[string]*prompb.LabelMatcher, len(srcMatchers))
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

type PromPbLabelMatcherNameTranslator map[string]PromPbLabelMatcherTranslator

func (d PromPbLabelMatcherNameTranslator) Translate(srcMatchers map[string]*prompb.LabelMatcher) []*prompb.LabelMatcher {
	if d != nil {
		for name, t := range d {
			if m, exist := srcMatchers[name]; exist {
				srcMatchers[name] = t.Translate(m)
			} else if r := t.Translate(nil); r != nil {
				srcMatchers[name] = r
			}
		}
	}

	ret := make([]*prompb.LabelMatcher, 0, len(srcMatchers))
	for _, m := range srcMatchers {
		ret = append(ret, m)
	}

	return ret
}

type prompbLabelMatchersNameFilter struct {
	dropper    PromPbLabelMatchersDropper
	translator PromPbLabelMatchersTranslator
}

func (f *prompbLabelMatchersNameFilter) Filter(srcMatchers []*prompb.LabelMatcher) []*prompb.LabelMatcher {
	return f.translator.Translate(f.dropper.Drop(srcMatchers))
}

func CreatePromPbLabelMatchersNameFilter(dropper PromPbLabelMatcherNameDropper, translator PromPbLabelMatcherNameTranslator) PromPbLabelMatchersFilter {
	return &prompbLabelMatchersNameFilter{
		dropper:    dropper,
		translator: translator,
	}
}

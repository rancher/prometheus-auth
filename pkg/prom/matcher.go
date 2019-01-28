package prom

import (
	"fmt"

	promlb "github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/prompb"
)

const (
	noneNamespace = "______"
)

func createMatcher(matcherName string, namespaces []string) *promlb.Matcher {
	ret := &promlb.Matcher{
		Name: matcherName,
	}

	modifyMatcher(ret, namespaces)

	return ret
}

func createLabelMatcher(matcherName string, namespaces []string) *prompb.LabelMatcher {
	ret := &prompb.LabelMatcher{
		Name: matcherName,
	}

	modifyLabelMatcher(ret, namespaces)

	return ret
}

func modifyMatcher(srcMatcher *promlb.Matcher, namespaces []string) {
	size := len(namespaces)

	if size == 0 {
		srcMatcher.Type = promlb.MatchEqual
		srcMatcher.Value = noneNamespace
	} else if size == 1 {
		srcMatcher.Type = promlb.MatchEqual
		srcMatcher.Value = namespaces[0]
	} else {
		srcMatcher.Type = promlb.MatchRegexp
		srcMatcher.Value = join(namespaces)
	}
}

func modifyLabelMatcher(srcMatcher *prompb.LabelMatcher, namespaces []string) {
	size := len(namespaces)

	if size == 0 {
		srcMatcher.Type = prompb.LabelMatcher_EQ
		srcMatcher.Value = noneNamespace
	} else if size == 1 {
		srcMatcher.Type = prompb.LabelMatcher_EQ
		srcMatcher.Value = namespaces[0]
	} else {
		srcMatcher.Type = prompb.LabelMatcher_RE
		srcMatcher.Value = join(namespaces)
	}
}

func toLabelMatchers(matchers []*promlb.Matcher) ([]*prompb.LabelMatcher, error) {
	pbMatchers := make([]*prompb.LabelMatcher, 0, len(matchers))
	for _, m := range matchers {
		var mType prompb.LabelMatcher_Type
		switch m.Type {
		case promlb.MatchEqual:
			mType = prompb.LabelMatcher_EQ
		case promlb.MatchNotEqual:
			mType = prompb.LabelMatcher_NEQ
		case promlb.MatchRegexp:
			mType = prompb.LabelMatcher_RE
		case promlb.MatchNotRegexp:
			mType = prompb.LabelMatcher_NRE
		default:
			return nil, fmt.Errorf("invalid matcher type")
		}
		pbMatchers = append(pbMatchers, &prompb.LabelMatcher{
			Type:  mType,
			Name:  m.Name,
			Value: m.Value,
		})
	}
	return pbMatchers, nil
}

func fromLabelMatchers(matchers []*prompb.LabelMatcher) ([]*promlb.Matcher, error) {
	result := make([]*promlb.Matcher, 0, len(matchers))
	for _, matcher := range matchers {
		var mtype promlb.MatchType
		switch matcher.Type {
		case prompb.LabelMatcher_EQ:
			mtype = promlb.MatchEqual
		case prompb.LabelMatcher_NEQ:
			mtype = promlb.MatchNotEqual
		case prompb.LabelMatcher_RE:
			mtype = promlb.MatchRegexp
		case prompb.LabelMatcher_NRE:
			mtype = promlb.MatchNotRegexp
		default:
			return nil, fmt.Errorf("invalid matcher type")
		}
		matcher, err := promlb.NewMatcher(mtype, matcher.Name, matcher.Value)
		if err != nil {
			return nil, err
		}
		result = append(result, matcher)
	}
	return result, nil
}

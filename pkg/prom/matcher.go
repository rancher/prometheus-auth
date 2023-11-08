package prom

import (
	"fmt"

	promlb "github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/prompb"
	log "github.com/sirupsen/logrus"
)

const (
	noneNamespace = "______"
)

func createMatcher(matcherName string, namespaces []string) *promlb.Matcher {
	ret := &promlb.Matcher{
		Name: matcherName,
	}

	return modifyMatcher(ret, namespaces)
}

func createLabelMatcher(matcherName string, namespaces []string) *prompb.LabelMatcher {
	ret := &prompb.LabelMatcher{
		Name: matcherName,
	}

	modifyLabelMatcher(ret, namespaces)

	return ret
}

func modifyMatcher(srcMatcher *promlb.Matcher, namespaces []string) *promlb.Matcher {
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

	matcher, err := promlb.NewMatcher(srcMatcher.Type, srcMatcher.Name, srcMatcher.Value)
	if err != nil {
		log.Errorf("unable to modify matcher from %s%s%s to %s%s%s",
			srcMatcher.Name, srcMatcher.Type, srcMatcher.Value,
			matcher.Name, matcher.Type, matcher.Value,
		)
	}

	return matcher
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

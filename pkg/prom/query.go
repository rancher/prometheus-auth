package prom

import (
	"fmt"
)

func NewExprForCountAllLabels(namespaces []string) string {
	instantVectorSelectors := NewInstantVectorSelectorsForNamespaces(namespaces)

	return fmt.Sprintf(`count (%s) by (__name__)`, instantVectorSelectors)
}

func NewInstantVectorSelectorsForNamespaces(namespaces []string) string {
	ret := createMatcher(namespaceMatchName, namespaces)

	return fmt.Sprintf(`{%s}`, ret.String())
}

package kubeauth

import (
	"fmt"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"sort"
	"strings"
)

type operationKind string
type operationType string

const (
	/**
	unit kind
	*/
	serviceAccount     operationKind = "ServiceAccount"
	roleBinding        operationKind = "RoleBinding"
	clusterRoleBinding operationKind = "ClusterRoleBinding"
	role               operationKind = "Role"
	clusterRole        operationKind = "ClusterRole"
	namespace          operationKind = "Namespace"

	/**
	unit type
	*/
	operationAdd    operationType = "adding"
	operationDelete operationType = "deleting"
)

type unit struct {
	tpy  operationType
	kind operationKind
	key  string
}

func addOperation(kind operationKind, o interface{}) *unit {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(o)
	if err != nil {
		return nil
	}

	return &unit{
		tpy:  operationAdd,
		kind: kind,
		key:  key,
	}
}

func delOperation(kind operationKind, o interface{}) *unit {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(o)
	if err != nil {
		return nil
	}

	return &unit{
		tpy:  operationDelete,
		kind: kind,
		key:  key,
	}
}

func iteratePolicyRules(rules []rbacv1.PolicyRule, call func(rule rbacv1.PolicyRule) (stop bool)) {
	for _, rule := range rules {
		for _, apiGroup := range rule.APIGroups {
			if apiGroup == "" || apiGroup == "*" {
				for _, resource := range rule.Resources {
					if resource == "namespaces" || resource == "*" {
						for _, verb := range rule.Verbs {
							if verb == "get" || verb == "*" {
								if call(rule) {
									return
								}
							}
						}
					}
				}
			}
		}
	}
}

func compareBindingSubjects(left, right []rbacv1.Subject) bool {
	if left == nil && right == nil {
		return true
	}

	if left != nil && right != nil {
		rightLen := len(right)
		if len(left) != rightLen {
			return false
		}

		rightCache := make(map[string]struct{}, rightLen)
		for _, l := range left {
			ls := stringSubject(&l)
			if _, exist := rightCache[ls]; !exist {
				for i := len(rightCache); i < rightLen; i++ {
					rs := stringSubject(&right[i])
					rightCache[rs] = struct{}{}
					if ls == rs {
						exist = true
						break
					}
				}

				if !exist {
					return false
				}
			}
		}

		return true

	}
	return true
}

func compareBindingRoleRef(left, right *rbacv1.RoleRef) bool {
	if left == right {
		return true
	}

	if left != nil && right != nil {
		return left.APIGroup == right.APIGroup && left.Kind == right.Kind && left.Name == right.Name
	}

	return false
}

func compareRolePolicyRules(left, right []rbacv1.PolicyRule) bool {
	if left == nil && right == nil {
		return true
	}

	if left != nil && right != nil {
		rightLen := len(right)
		if len(left) != rightLen {
			return false
		}

		rightCache := make(map[string]struct{}, rightLen)
		for _, l := range left {
			ls := stringPolicyRule(&l)
			if _, exist := rightCache[ls]; !exist {
				for i := len(rightCache); i < rightLen; i++ {
					rs := stringPolicyRule(&right[i])
					rightCache[rs] = struct{}{}
					if ls == rs {
						exist = true
						break
					}
				}

				if !exist {
					return false
				}
			}
		}

		return true
	}

	return false
}

func compareRoleAggregationRule(left, right *rbacv1.AggregationRule) bool {
	if left == right {
		return true
	}

	if left != nil && right != nil {
		leftLabelSelectors := left.ClusterRoleSelectors
		rightLabelSelectors := right.ClusterRoleSelectors

		rightLen := len(rightLabelSelectors)
		if len(leftLabelSelectors) != rightLen {
			return false
		}

		rightCache := make(map[string]struct{}, rightLen)
		for _, l := range leftLabelSelectors {
			ls := stringLabelSelector(&l)
			if _, exist := rightCache[ls]; !exist {
				for i := len(rightCache); i < rightLen; i++ {
					rs := stringLabelSelector(&rightLabelSelectors[i])
					rightCache[rs] = struct{}{}
					if ls == rs {
						exist = true
						break
					}
				}

				if !exist {
					return false
				}
			}
		}

		return true
	}

	return false
}

func stringSubject(subject *rbacv1.Subject) string {
	return fmt.Sprintf("%s-%s-%s-%s", subject.APIGroup, subject.Kind, subject.Name, subject.Namespace)
}

func stringPolicyRule(policyRule *rbacv1.PolicyRule) string {
	sb := &strings.Builder{}

	if len(policyRule.APIGroups) != 0 {
		values := policyRule.APIGroups

		sort.Strings(values)
		sb.WriteString("APIGroups:")
		sb.WriteString(strings.Join(values, "|"))
		sb.WriteString(";")
	}
	if len(policyRule.NonResourceURLs) != 0 {
		values := policyRule.NonResourceURLs

		sort.Strings(values)
		sb.WriteString("NonResourceURLs:")
		sb.WriteString(strings.Join(values, "|"))
		sb.WriteString(";")
	}
	if len(policyRule.ResourceNames) != 0 {
		values := policyRule.ResourceNames

		sort.Strings(values)
		sb.WriteString("ResourceNames:")
		sb.WriteString(strings.Join(values, "|"))
		sb.WriteString(";")
	}
	if len(policyRule.Resources) != 0 {
		values := policyRule.Resources

		sort.Strings(values)
		sb.WriteString("Resources:")
		sb.WriteString(strings.Join(values, "|"))
		sb.WriteString(";")
	}
	if len(policyRule.Verbs) != 0 {
		values := policyRule.Verbs

		sort.Strings(values)
		sb.WriteString("Verbs:")
		sb.WriteString(strings.Join(values, "|"))
		sb.WriteString(";")
	}

	return sb.String()
}

func stringLabelSelector(labelSelector *metav1.LabelSelector) string {
	sb := &strings.Builder{}

	if labelSelector.MatchLabels != nil {
		sb.WriteString("MatchLabels:")

		matchLabelsKeyList := make([]string, 0, len(labelSelector.MatchLabels))
		for matchLabel := range labelSelector.MatchLabels {
			matchLabelsKeyList = append(matchLabelsKeyList, matchLabel)
		}
		sort.Strings(matchLabelsKeyList)

		for _, key := range matchLabelsKeyList {
			sb.WriteString(key + ":" + labelSelector.MatchLabels[key] + ",")
		}

		sb.WriteString(";")
	}

	if len(labelSelector.MatchExpressions) != 0 {
		sb.WriteString("MatchExpressions:")

		matchExpressionList := make([]string, 0, len(labelSelector.MatchExpressions))
		for _, matchExpression := range labelSelector.MatchExpressions {
			if len(matchExpression.Values) != 0 {
				matchExpressionValues := matchExpression.Values
				sort.Strings(matchExpressionValues)
				matchExpressionList = append(matchExpressionList, fmt.Sprintf("%s-%s-%s", matchExpression.Key, matchExpression.Operator, strings.Join(matchExpressionValues, "|")))
			} else {
				matchExpressionList = append(matchExpressionList, fmt.Sprintf("%s-%s", matchExpression.Key, matchExpression.Operator))
			}
		}
		sort.Strings(matchExpressionList)

		sb.WriteString(strings.Join(matchExpressionList, "|"))
		sb.WriteString(";")
	}

	return sb.String()
}

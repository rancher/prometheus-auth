package kubeauth

import (
	"fmt"
	"github.com/juju/errors"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
)

func (v *globalNamespacesOwnedView) handleUnit(u *unit) error {
	switch u.tpy {
	case operationAdd:
		return v.addUnit(u.kind, u.key)
	case operationDelete:
		return v.deleteUnit(u.kind, u.key)
	}

	return nil
}

func (v *globalNamespacesOwnedView) extractTokenFromServiceAccount(namespace, name string) (string, error) {
	sa, err := v.serviceAccountLister.ServiceAccounts(namespace).Get(name)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return "", err
		}

		sa, err = v.serviceAccountRemote.ServiceAccounts(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
	}

	if len(sa.Secrets) != 1 {
		return "", errors.NotFoundf("Secret in ServiceAccount '%s/%s'", namespace, name)
	}

	sec, err := v.secretLister.Secrets(namespace).Get(sa.Secrets[0].Name)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return "", err
		}

		sec, err = v.secretRemote.Secrets(namespace).Get(sa.Secrets[0].Name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
	}

	token := string(sec.Data[corev1.ServiceAccountTokenKey])
	if len(token) == 0 {
		return "", errors.NotFoundf("Token in Secret '%s/%s' of ServiceAccount '%s/%s'", namespace, sec.Name, namespace, name)
	}

	return token, nil
}

func (v *globalNamespacesOwnedView) addUnit(kind operationKind, key string) error {
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(errors.Annotatef(err, "failed to split key %s", key))
		return nil
	}

	switch kind {
	case roleBinding:
		roleBindingObj, err := v.roleBindingLister.RoleBindings(ns).Get(name)
		if err != nil {
			if !k8serrors.IsNotFound(err) {
				return err
			}

			roleBindingObj, err = v.roleBindingRemote.RoleBindings(ns).Get(name, metav1.GetOptions{})
			if err != nil {
				return err
			}
		}

		for _, subject := range roleBindingObj.Subjects {
			if subject.Kind == rbacv1.ServiceAccountKind {
				token, err := v.extractTokenFromServiceAccount(subject.Namespace, subject.Name)
				if err != nil {
					if !k8serrors.IsNotFound(err) {
						return err
					}

					continue
				}

				v.view.Edge(addToken2RoleBinding, token, key)
			}
		}

		if roleBindingObj.RoleRef.Kind == "ClusterRole" {
			v.view.Edge(addRoleBinding2ClusterRole, key, roleBindingObj.RoleRef.Name)
		} else {
			v.view.Edge(addRoleBinding2Role, key, fmt.Sprintf("%s/%s", ns, roleBindingObj.RoleRef.Name))
		}
	case clusterRoleBinding:
		clusterRoleBindingObj, err := v.clusterRoleBindingLister.Get(name)
		if err != nil {
			if !k8serrors.IsNotFound(err) {
				return err
			}

			clusterRoleBindingObj, err = v.clusterRoleBindingRemote.ClusterRoleBindings().Get(name, metav1.GetOptions{})
			if err != nil {
				return err
			}
		}

		for _, subject := range clusterRoleBindingObj.Subjects {
			if subject.Kind == rbacv1.ServiceAccountKind {
				token, err := v.extractTokenFromServiceAccount(subject.Namespace, subject.Name)
				if err != nil {
					if !k8serrors.IsNotFound(err) {
						return err
					}

					continue
				}

				v.view.Edge(addToken2ClusterRoleBinding, token, key)
			}
		}

		v.view.Edge(addClusterRoleBinding2ClusterRole, key, clusterRoleBindingObj.RoleRef.Name)
	case role:
		roleObj, err := v.roleLister.Roles(ns).Get(name)
		if err != nil {
			if !k8serrors.IsNotFound(err) {
				return err
			}

			roleObj, err = v.roleRemote.Roles(ns).Get(name, metav1.GetOptions{})
			if err != nil {
				return err
			}
		}

		iteratePolicyRules(roleObj.Rules, func(rule rbacv1.PolicyRule) (stop bool) {
			v.view.Edge(addRole2Namespace, key, ns)
			stop = true
			return
		})
	case clusterRole:
		clusterRoleObj, err := v.clusterRoleLister.Get(name)
		if err != nil {
			if !k8serrors.IsNotFound(err) {
				return err
			}

			clusterRoleObj, err = v.clusterRoleRemote.ClusterRoles().Get(name, metav1.GetOptions{})
			if err != nil {
				return err
			}
		}

		iteratePolicyRules(clusterRoleObj.Rules, func(rule rbacv1.PolicyRule) (stop bool) {
			if len(rule.ResourceNames) == 0 {
				v.view.Edge(addClusterRole2Namespace, key, "*")
			} else {
				for _, resourceName := range rule.ResourceNames {
					v.view.Edge(addClusterRole2Namespace, key, resourceName)
				}
			}

			return
		})
	default:
		return errors.New(fmt.Sprintf("can't deal with %s kind", kind))
	}

	return nil
}

func (v *globalNamespacesOwnedView) deleteUnit(kind operationKind, key string) error {
	switch kind {
	case serviceAccount:
		ns, name, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			runtime.HandleError(errors.Annotatef(err, "failed to split key '%s'", key))
			return nil
		}

		token, err := v.extractTokenFromServiceAccount(ns, name)
		if err != nil {
			if !k8serrors.IsNotFound(err) {
				return err
			}
		}

		v.view.Vertex(delToken, token)
	case roleBinding:
		v.view.Vertex(delRoleBinding, key)
	case role:
		v.view.Vertex(delRole, key)
	case clusterRoleBinding:
		v.view.Vertex(delClusterRoleBinding, key)
	case clusterRole:
		v.view.Vertex(delClusterRole, key)
	case namespace:
		v.view.Vertex(delNamespace, key)
	default:
		return errors.New(fmt.Sprintf("can't deal with %s kind", kind))
	}

	return nil
}

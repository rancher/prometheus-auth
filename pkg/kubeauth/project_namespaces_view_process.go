package kubeauth

import (
	"fmt"
	"github.com/juju/errors"
)

func (v *projectNamespacesOwnedView) handleUnit(u *unit) error {
	switch u.tpy {
	case operationAdd:
		return v.addUnit(u.kind, u.key)
	case operationDelete:
		return v.deleteUnit(u.kind, u.key)
	}

	return nil
}

func (v *projectNamespacesOwnedView) addUnit(kind operationKind, key string) error {
	switch kind {
	case namespace:
		v.view.Put(key)
	default:
		return errors.New(fmt.Sprintf("can't deal with %s kind", kind))

	}

	return nil
}

func (v *projectNamespacesOwnedView) deleteUnit(kind operationKind, key string) error {
	switch kind {
	case namespace:
		v.view.Del(key)
	default:
		return errors.New(fmt.Sprintf("can't deal with %s kind", kind))
	}

	return nil
}

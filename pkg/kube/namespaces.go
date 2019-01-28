package kube

import (
	"context"

	"github.com/rancher/prometheus-auth/pkg/data"
	"k8s.io/client-go/kubernetes"
)

type Namespaces interface {
	Query(token string) data.Set
}

type namespaces struct {
}

func (n *namespaces) Query(token string) data.Set {
	return data.Set{}
}

func NewNamespaces(ctx context.Context, k8sClient kubernetes.Interface) Namespaces {
	return &namespaces{
	}
}

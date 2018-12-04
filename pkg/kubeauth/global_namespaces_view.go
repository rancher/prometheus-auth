package kubeauth

import (
	"fmt"
	"github.com/juju/errors"
	"github.com/rancher/prometheus-auth/pkg/kubeauth/view"
	"github.com/rancher/prometheus-auth/pkg/utils"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/singleflight"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	corev1remotes "k8s.io/client-go/kubernetes/typed/core/v1"
	rbacv1remotes "k8s.io/client-go/kubernetes/typed/rbac/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	rbacv1listers "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"time"
)

var (
	/**
	vertex kind
	*/
	tokenVertexKind              = view.NewVertexKind("token:", "", true)
	roleVertexKind               = view.NewVertexKind("role:", "", false)
	roleBindingVertexKind        = view.NewVertexKind("roleBinding:", "", false)
	clusterRoleVertexKind        = view.NewVertexKind("clusterRole:", "", false)
	clusterRoleBindingVertexKind = view.NewVertexKind("clusterRoleBinding:", "", false)
	namespaceVertexKind          = view.NewVertexKind("namespace:", "*", false)

	/**
	vertex operation
	*/
	//addToken view.VertexOperation = func(graph view.Graph, value string) {
	//	graph.AddVertex(tokenVertexKind.Wrap(value))
	//}
	//addRole view.VertexOperation = func(graph view.Graph, value string) {
	//	graph.AddVertex(roleVertexKind.Wrap(value))
	//}
	//addRoleBinding view.VertexOperation = func(graph view.Graph, value string) {
	//	graph.AddVertex(roleBindingVertexKind.Wrap(value))
	//}
	//addClusterRole view.VertexOperation = func(graph view.Graph, value string) {
	//	graph.AddVertex(clusterRoleVertexKind.Wrap(value))
	//}
	//addClusterRoleBinding view.VertexOperation = func(graph view.Graph, value string) {
	//	graph.AddVertex(clusterRoleBindingVertexKind.Wrap(value))
	//}
	//addNamespace view.VertexOperation = func(graph view.Graph, value string) {
	//	graph.AddVertex(namespaceVertexKind.Wrap(value))
	//}
	delToken view.VertexOperation = func(graph view.Graph, value string) {
		graph.DelVertex(tokenVertexKind.Wrap(value))
	}
	delRole view.VertexOperation = func(graph view.Graph, value string) {
		graph.DelVertex(roleVertexKind.Wrap(value))
	}
	delRoleBinding view.VertexOperation = func(graph view.Graph, value string) {
		graph.DelVertex(roleBindingVertexKind.Wrap(value))
	}
	delClusterRole view.VertexOperation = func(graph view.Graph, value string) {
		graph.DelVertex(clusterRoleVertexKind.Wrap(value))
	}
	delClusterRoleBinding view.VertexOperation = func(graph view.Graph, value string) {
		graph.DelVertex(clusterRoleBindingVertexKind.Wrap(value))
	}
	delNamespace view.VertexOperation = func(graph view.Graph, value string) {
		graph.DelVertex(namespaceVertexKind.Wrap(value))
	}

	/**
	edge operation
	*/
	addToken2RoleBinding view.EdgeOperation = func(graph view.Graph, from, to string) {
		graph.AddEdge(tokenVertexKind.Wrap(from), roleBindingVertexKind.Wrap(to))
	}
	addToken2ClusterRoleBinding view.EdgeOperation = func(graph view.Graph, from, to string) {
		graph.AddEdge(tokenVertexKind.Wrap(from), clusterRoleBindingVertexKind.Wrap(to))
	}
	addRoleBinding2Role view.EdgeOperation = func(graph view.Graph, from, to string) {
		graph.AddEdge(roleBindingVertexKind.Wrap(from), roleVertexKind.Wrap(to))
	}
	addRoleBinding2ClusterRole view.EdgeOperation = func(graph view.Graph, from, to string) {
		graph.AddEdge(roleBindingVertexKind.Wrap(from), clusterRoleVertexKind.Wrap(to))
	}
	addClusterRoleBinding2ClusterRole view.EdgeOperation = func(graph view.Graph, from, to string) {
		graph.AddEdge(clusterRoleBindingVertexKind.Wrap(from), clusterRoleVertexKind.Wrap(to))
	}
	addRole2Namespace view.EdgeOperation = func(graph view.Graph, from, to string) {
		graph.AddEdge(roleVertexKind.Wrap(from), namespaceVertexKind.Wrap(to))
	}
	addClusterRole2Namespace view.EdgeOperation = func(graph view.Graph, from, to string) {
		graph.AddEdge(clusterRoleVertexKind.Wrap(from), namespaceVertexKind.Wrap(to))
	}
	//delToken2RoleBinding view.EdgeOperation = func(graph view.Graph, from, to string) {
	//	graph.DelEdge(tokenVertexKind.Wrap(from), roleBindingVertexKind.Wrap(to))
	//}
	//delToken2ClusterRoleBinding view.EdgeOperation = func(graph view.Graph, from, to string) {
	//	graph.DelEdge(tokenVertexKind.Wrap(from), clusterRoleBindingVertexKind.Wrap(to))
	//}
	//delRoleBinding2Role view.EdgeOperation = func(graph view.Graph, from, to string) {
	//	graph.DelEdge(roleBindingVertexKind.Wrap(from), roleVertexKind.Wrap(to))
	//}
	//delRoleBinding2ClusterRole view.EdgeOperation = func(graph view.Graph, from, to string) {
	//	graph.DelEdge(roleBindingVertexKind.Wrap(from), clusterRoleVertexKind.Wrap(to))
	//}
	//delClusterRoleBinding2ClusterRole view.EdgeOperation = func(graph view.Graph, from, to string) {
	//	graph.DelEdge(clusterRoleBindingVertexKind.Wrap(from), clusterRoleVertexKind.Wrap(to))
	//}
	//delRole2Namespace view.EdgeOperation = func(graph view.Graph, from, to string) {
	//	graph.DelEdge(roleVertexKind.Wrap(from), namespaceVertexKind.Wrap(to))
	//}
	//delClusterRole2Namespace view.EdgeOperation = func(graph view.Graph, from, to string) {
	//	graph.DelEdge(clusterRoleVertexKind.Wrap(from), namespaceVertexKind.Wrap(to))
	//}
)

type (
	GlobalNamespacesOwnedView interface {
		Own(token string) *OwnedNamespaces
		Run(stopCh <-chan struct{})
	}

	OwnedNamespaces struct {
		sr *view.GraphSearchResult
	}

	globalNamespacesOwnedView struct {
		sfGroup *singleflight.Group

		view                  view.GraphView
		sharedInformerFactory informers.SharedInformerFactory
		queue                 *workqueue.Type

		clusterRoleLister        rbacv1listers.ClusterRoleLister
		clusterRoleBindingLister rbacv1listers.ClusterRoleBindingLister
		roleLister               rbacv1listers.RoleLister
		roleBindingLister        rbacv1listers.RoleBindingLister
		serviceAccountLister     corev1listers.ServiceAccountLister
		secretLister             corev1listers.SecretLister

		clusterRoleRemote        rbacv1remotes.ClusterRolesGetter
		clusterRoleBindingRemote rbacv1remotes.ClusterRoleBindingsGetter
		roleRemote               rbacv1remotes.RolesGetter
		roleBindingRemote        rbacv1remotes.RoleBindingsGetter
		serviceAccountRemote     corev1remotes.ServiceAccountsGetter
		secretRemote             corev1remotes.SecretsGetter
	}
)

func NewGlobalNamespacesOwnedView(sharedInformerFactory informers.SharedInformerFactory, k8sClient kubernetes.Interface) GlobalNamespacesOwnedView {
	runtime.ReallyCrash = false
	runtime.PanicHandlers = []func(interface{}){
		func(i interface{}) {
			if err, ok := i.(error); ok {
				log.Error(errors.ErrorStack(err))
			} else {
				log.Error(i)
			}
		},
	}
	runtime.ErrorHandlers = []func(err error){
		func(err error) {
			log.Error(errors.ErrorStack(err))
		},
	}

	rbacv1Interface := sharedInformerFactory.Rbac().V1()
	corev1Interface := sharedInformerFactory.Core().V1()
	rbacv1Getter := k8sClient.RbacV1()
	corev1Getter := k8sClient.CoreV1()

	v := &globalNamespacesOwnedView{
		sfGroup: &singleflight.Group{},

		view:                  view.NewGraphView(),
		sharedInformerFactory: sharedInformerFactory,
		queue:                 workqueue.NewNamed("global_namespaces"),

		clusterRoleLister:        rbacv1Interface.ClusterRoles().Lister(),
		clusterRoleBindingLister: rbacv1Interface.ClusterRoleBindings().Lister(),
		roleLister:               rbacv1Interface.Roles().Lister(),
		roleBindingLister:        rbacv1Interface.RoleBindings().Lister(),
		serviceAccountLister:     corev1Interface.ServiceAccounts().Lister(),
		secretLister:             corev1Interface.Secrets().Lister(),

		clusterRoleRemote:        rbacv1Getter,
		clusterRoleBindingRemote: rbacv1Getter,
		roleRemote:               rbacv1Getter,
		roleBindingRemote:        rbacv1Getter,
		serviceAccountRemote:     corev1Getter,
		secretRemote:             corev1Getter,
	}

	corev1Interface.ServiceAccounts().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(o interface{}) {
			v.enqueue(delOperation(serviceAccount, o))
		},
	})

	rbacv1Interface.ClusterRoleBindings().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(o interface{}) {
			v.enqueue(addOperation(clusterRoleBinding, o))
		},
		DeleteFunc: func(o interface{}) {
			v.enqueue(delOperation(clusterRoleBinding, o))
		},
		UpdateFunc: func(oo, o interface{}) {
			left := oo.(*rbacv1.ClusterRoleBinding)
			right := o.(*rbacv1.ClusterRoleBinding)

			if !compareBindingRoleRef(&left.RoleRef, &right.RoleRef) || !compareBindingSubjects(left.Subjects, right.Subjects) {
				v.enqueue(delOperation(clusterRoleBinding, o))
				v.enqueue(addOperation(clusterRoleBinding, o))
			}
		},
	})

	rbacv1Interface.RoleBindings().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(o interface{}) {
			v.enqueue(addOperation(roleBinding, o))
		},
		DeleteFunc: func(o interface{}) {
			v.enqueue(delOperation(roleBinding, o))
		},
		UpdateFunc: func(oo, o interface{}) {
			left := oo.(*rbacv1.RoleBinding)
			right := o.(*rbacv1.RoleBinding)

			if !compareBindingRoleRef(&left.RoleRef, &right.RoleRef) || !compareBindingSubjects(left.Subjects, right.Subjects) {
				v.enqueue(delOperation(roleBinding, o))
				v.enqueue(addOperation(roleBinding, o))
			}
		},
	})

	rbacv1Interface.ClusterRoles().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(o interface{}) {
			v.enqueue(addOperation(clusterRole, o))
		},
		DeleteFunc: func(o interface{}) {
			v.enqueue(delOperation(clusterRole, o))
		},
		UpdateFunc: func(oo, o interface{}) {
			left := oo.(*rbacv1.ClusterRole)
			right := o.(*rbacv1.ClusterRole)

			if !compareRolePolicyRules(left.Rules, right.Rules) || !compareRoleAggregationRule(left.AggregationRule, right.AggregationRule) {
				v.enqueue(delOperation(clusterRole, o))
				v.enqueue(addOperation(clusterRole, o))
			}
		},
	})

	rbacv1Interface.Roles().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(o interface{}) {
			v.enqueue(addOperation(role, o))
		},
		DeleteFunc: func(o interface{}) {
			v.enqueue(delOperation(role, o))
		},
		UpdateFunc: func(oo, o interface{}) {
			left := oo.(*rbacv1.Role)
			right := o.(*rbacv1.Role)

			if !compareRolePolicyRules(left.Rules, right.Rules) {
				v.enqueue(delOperation(role, o))
				v.enqueue(addOperation(role, o))
			}
		},
	})

	corev1Interface.Namespaces().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(o interface{}) {
			v.enqueue(delOperation(namespace, o))
		},
	})

	return v
}

func (v *globalNamespacesOwnedView) enqueue(op *unit) {
	if op == nil {
		return
	}

	v.queue.Add(op)
}

func (v *globalNamespacesOwnedView) runWorker() {
	for v.processNextWorkItem() {
	}
}

func (v *globalNamespacesOwnedView) processNextWorkItem() bool {
	obj, shutdown := v.queue.Get()
	if shutdown {
		return false
	}

	err := func(obj *unit) error {
		defer v.queue.Done(obj)

		utils.LogTrace(func() string {
			return fmt.Sprintf("Handling %s (%s) when %s", obj.kind, obj.key, obj.tpy)
		})

		if err := v.handleUnit(obj); err != nil {
			if errors.IsNotFound(err) {
				log.WithError(err).Warn("retry after 2 seconds")

				go func() {
					time.Sleep(2 * time.Second)
					v.queue.Add(obj)
				}()

				return nil
			}

			v.queue.Add(obj)
			return errors.Annotatef(err, "unable to handling %s (%s) when %s", obj.kind, obj.key, obj.tpy)
		}

		utils.LogTrace(func() string {
			return fmt.Sprintf("Successfully handled %s (%s) when %s", obj.kind, obj.key, obj.tpy)
		})

		return nil
	}(obj.(*unit))
	if err != nil {
		runtime.HandleError(err)
	}

	return true
}

func (v *globalNamespacesOwnedView) Own(token string) *OwnedNamespaces {
	tokenWrapped := tokenVertexKind.Wrap(token)
	ret, _, _ := v.sfGroup.Do(tokenWrapped, func() (interface{}, error) {
		return &OwnedNamespaces{sr: v.view.Search(tokenWrapped, view.BFS, namespaceVertexKind)}, nil
	})

	return ret.(*OwnedNamespaces)
}

func (v *globalNamespacesOwnedView) Run(stopCh <-chan struct{}) {
	defer runtime.HandleCrash()
	defer v.queue.ShutDown()

	log.Info("Starting sync for view workers")
	v.sharedInformerFactory.Start(stopCh)
	log.Info("Waiting for syncing")
	v.sharedInformerFactory.WaitForCacheSync(stopCh)

	log.Info("Starting view workers")
	// issue: cause races
	//for i := 0; i < goruntime.NumCPU(); i++ {
	//	go wait.Until(v.runWorker, time.Second, stopCh)
	//}
	go wait.Until(v.runWorker, time.Second, stopCh)

	// Block until the target provider is explicitly canceled.
	log.Info("Started view workers")
	<-stopCh
	log.Info("Shutting down view workers")
}

func (o *OwnedNamespaces) String() string {
	return fmt.Sprintf("owned namespaces => hasAll: %v, values: %v", o.HasAll(), o.Values())
}

func (o *OwnedNamespaces) HasAll() bool {
	return o.sr.QuitEarly()
}

func (o *OwnedNamespaces) Values() []string {
	return o.sr.Values()
}

func (o *OwnedNamespaces) ToSetView() view.SetView {
	return o.sr
}

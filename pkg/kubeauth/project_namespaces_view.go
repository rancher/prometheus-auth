package kubeauth

import (
	"fmt"
	"github.com/juju/errors"
	"github.com/rancher/prometheus-auth/pkg/kubeauth/view"
	"github.com/rancher/prometheus-auth/pkg/utils"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"time"
)

type (
	ProjectNamespacesOwnedView interface {
		view.SetView
		Run(stopCh <-chan struct{})
	}

	projectNamespacesOwnedView struct {
		view                  view.SetView
		sharedInformerFactory informers.SharedInformerFactory
		queue                 *workqueue.Type

		namespaceLister corev1listers.NamespaceLister
	}
)

func NewProjectNamespacesOwnedView(sharedInformerFactory informers.SharedInformerFactory, projectMatches func(obj interface{}) bool) ProjectNamespacesOwnedView {
	runtime.ReallyCrash = false
	runtime.PanicHandlers = []func(interface{}){
		func(i interface{}) {
			if err, ok := i.(error); ok {
				log.Error(errors.Details(err))
			} else {
				log.Error(i)
			}
		},
	}
	runtime.ErrorHandlers = []func(err error){
		func(err error) {
			log.Error(errors.Details(err))
		},
	}

	corev1Interface := sharedInformerFactory.Core().V1()

	v := &projectNamespacesOwnedView{
		view:                  view.NewSetView(),
		sharedInformerFactory: sharedInformerFactory,
		queue:                 workqueue.NewNamed("project_namespaces"),

		namespaceLister: corev1Interface.Namespaces().Lister(),
	}

	corev1Interface.Namespaces().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(o interface{}) {
			if match := projectMatches(o); match {
				v.enqueue(addOperation(namespace, o))
			}
		},
		UpdateFunc: func(oo, no interface{}) {
			oldMatch := projectMatches(oo)
			newMatch := projectMatches(no)
			if oldMatch {
				if !newMatch {
					v.enqueue(delOperation(namespace, oo))
				}
			} else if newMatch {
				v.enqueue(addOperation(namespace, no))
			}
		},
		DeleteFunc: func(o interface{}) {
			if match := projectMatches(o); match {
				v.enqueue(delOperation(namespace, o))
			}
		},
	})

	return v
}

func (v *projectNamespacesOwnedView) enqueue(op *unit) {
	if op == nil {
		return
	}

	v.queue.Add(op)
}

func (v *projectNamespacesOwnedView) runWorker() {
	for v.processNextWorkItem() {
	}
}

func (v *projectNamespacesOwnedView) processNextWorkItem() bool {
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

func (v *projectNamespacesOwnedView) Has(value string) bool {
	return v.view.Has(value)
}

func (v *projectNamespacesOwnedView) Put(value string) {
	v.view.Put(value)
}

func (v *projectNamespacesOwnedView) Del(value string) {
	v.view.Del(value)
}

func (v *projectNamespacesOwnedView) GetAll() []string {
	return v.view.GetAll()
}

func (v *projectNamespacesOwnedView) Run(stopCh <-chan struct{}) {
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

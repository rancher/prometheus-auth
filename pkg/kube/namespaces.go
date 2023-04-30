package kube

import (
	"context"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/juju/errors"
	"github.com/rancher/prometheus-auth/pkg/data"
	log "github.com/sirupsen/logrus"
	authorization "k8s.io/api/authorization/v1"
	core "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/cache"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	clientAuthorization "k8s.io/client-go/kubernetes/typed/authorization/v1"
	clientCache "k8s.io/client-go/tools/cache"
)

const (
	byTokenIndex     = "byToken"
	byProjectIDIndex = "byProjectID"
)

type Namespaces interface {
	Query(token string) data.Set
}

type namespaces struct {
	subjectAccessReviewsClient clientAuthorization.SubjectAccessReviewInterface
	reviewResultTTLCache       *cache.LRUExpireCache
	secretIndexer              clientCache.Indexer
	namespaceIndexer           clientCache.Indexer
}

func (n *namespaces) Query(token string) data.Set {
	ret, err := n.query(token)
	if err != nil {
		log.Warnln("failed to query Namespaces", errors.ErrorStack(err))
	}

	return ret
}

func (n *namespaces) query(token string) (data.Set, error) {
	ret := data.Set{}

	tokenNamespace, err := n.validate(token)
	if err != nil {
		return ret, err
	}

	nsObj, exist, _ := n.namespaceIndexer.GetByKey(tokenNamespace)
	if !exist {
		return ret, errors.New("unknown namespace of token " + tokenNamespace)
	}

	ns := toNamespace(nsObj)
	if ns.DeletionTimestamp != nil {
		return ret, errors.New("deleting namespace of token")
	}

	projectID, exist := getProjectID(ns)
	if !exist {
		return ret, errors.New("unknown project of token")
	}

	nsList, err := n.namespaceIndexer.ByIndex(byProjectIDIndex, projectID)
	if err != nil {
		return ret, errors.Annotatef(err, "invalid project")
	}

	for _, nsObj := range nsList {
		ns := toNamespace(nsObj)
		ret[ns.Name] = struct{}{}
	}
	return ret, nil
}

func (n *namespaces) validate(token string) (string, error) {
	claimNamespace := ""
	// parse token
	tokenJwt, _ := jwt.Parse(token, nil)
	claims, _ := tokenJwt.Claims.(jwt.MapClaims)
	// investigate token type
	switch claims["iss"] {
	// bound token
	case "rke":
		claimNamespace = claims["kubernetes.io"].(map[string]interface{})["namespace"].(string)
	// k3s
	case "https://kubernetes.default.svc.cluster.local":
		claimNamespace = claims["kubernetes.io"].(map[string]interface{})["namespace"].(string)
	// legacy token
	case "kubernetes/serviceaccount":
		claimNamespace = claims["kubernetes.io/serviceaccount/namespace"].(string)
	default:
		return "", errors.New("unknown token claim")
	}

	_, exist := n.reviewResultTTLCache.Get(token)
	if exist {
		return claimNamespace, nil
	}

	projectMonitoringServiceAccountName := "project-monitoring"
	sarUser := fmt.Sprintf("system:serviceaccount:%s:%s", claimNamespace, projectMonitoringServiceAccountName)
	sar := &authorization.SubjectAccessReview{
		Spec: authorization.SubjectAccessReviewSpec{
			ResourceAttributes: &authorization.ResourceAttributes{
				Namespace: claimNamespace,
				Verb:      "view",
				Group:     "monitoring.cattle.io",
				Resource:  "prometheus",
			},
			User: sarUser,
		},
	}
	reviewResult, err := n.subjectAccessReviewsClient.Create(context.TODO(), sar, meta.CreateOptions{})
	if err != nil {
		return "", errors.Annotatef(err, "failed to review token")
	}

	if !reviewResult.Status.Allowed || reviewResult.Status.Denied {
		return "", errors.New("denied token")
	}

	n.reviewResultTTLCache.Add(token, struct{}{}, 5*time.Minute)

	return claimNamespace, nil
}

func NewNamespaces(ctx context.Context, k8sClient kubernetes.Interface) Namespaces {
	// secrets
	sec := k8sClient.CoreV1().Secrets(meta.NamespaceAll)
	secListWatch := &clientCache.ListWatch{
		ListFunc: func(options meta.ListOptions) (object runtime.Object, e error) {
			return sec.List(context.TODO(), options)
		},
		WatchFunc: func(options meta.ListOptions) (i watch.Interface, e error) {
			return sec.Watch(context.TODO(), options)
		},
	}
	secInformer := clientCache.NewSharedIndexInformer(secListWatch, &core.Secret{}, 2*time.Hour, clientCache.Indexers{byTokenIndex: secretByToken})

	// namespaces
	ns := k8sClient.CoreV1().Namespaces()
	nsListWatch := &clientCache.ListWatch{
		ListFunc: func(options meta.ListOptions) (object runtime.Object, e error) {
			return ns.List(context.TODO(), options)
		},
		WatchFunc: func(options meta.ListOptions) (i watch.Interface, e error) {
			return ns.Watch(context.TODO(), options)
		},
	}
	nsInformer := clientCache.NewSharedIndexInformer(nsListWatch, &core.Namespace{}, 10*time.Minute, clientCache.Indexers{byProjectIDIndex: namespaceByProjectID})

	// run
	go secInformer.Run(ctx.Done())
	go nsInformer.Run(ctx.Done())

	return &namespaces{
		subjectAccessReviewsClient: k8sClient.AuthorizationV1().SubjectAccessReviews(),
		reviewResultTTLCache:       cache.NewLRUExpireCache(1024),
		secretIndexer:              secInformer.GetIndexer(),
		namespaceIndexer:           nsInformer.GetIndexer(),
	}
}

func toNamespace(obj interface{}) *core.Namespace {
	ns, ok := obj.(*core.Namespace)
	if !ok {
		return &core.Namespace{}
	}

	return ns
}

func toSecret(obj interface{}) *core.Secret {
	sec, ok := obj.(*core.Secret)
	if !ok {
		return &core.Secret{}
	}

	return sec
}

func getProjectID(ns *core.Namespace) (string, bool) {
	if ns != nil && ns.Labels != nil {
		projectID, exist := ns.Labels["field.cattle.io/projectId"]
		if exist {
			return projectID, true
		}
	}

	return "", false
}

func namespaceByProjectID(obj interface{}) ([]string, error) {
	projectID, exist := getProjectID(toNamespace(obj))
	if exist {
		return []string{projectID}, nil
	}

	return []string{}, nil
}

func secretByToken(obj interface{}) ([]string, error) {
	sec := toSecret(obj)
	if sec.Type == core.SecretTypeServiceAccountToken {
		secretToken := sec.Data[core.ServiceAccountTokenKey]
		if len(secretToken) != 0 {
			return []string{string(secretToken)}, nil
		}
	}

	return []string{}, nil
}

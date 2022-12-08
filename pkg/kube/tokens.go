package kube

import (
	"context"
	"fmt"
	"time"

	authentication "k8s.io/api/authentication/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/cache"
	"k8s.io/client-go/kubernetes"
	clientAuthentication "k8s.io/client-go/kubernetes/typed/authentication/v1"
)

type Tokens interface {
	Authenticate(token string) (authentication.UserInfo, error)
}

type tokens struct {
	tokenReviewClient    clientAuthentication.TokenReviewInterface
	reviewResultTTLCache *cache.LRUExpireCache
}

func (t *tokens) Authenticate(token string) (authentication.UserInfo, error) {
	var userInfo authentication.UserInfo

	userInfoInterface, exist := t.reviewResultTTLCache.Get(token)
	if exist {
		userInfo = userInfoInterface.(authentication.UserInfo)
		return userInfo, nil
	}

	tokenReview, err := t.tokenReviewClient.Create(context.TODO(), toTokenReview(token), meta.CreateOptions{})
	if err != nil {
		return userInfo, err
	}
	userInfo = tokenReview.Status.User
	if !tokenReview.Status.Authenticated {
		return userInfo, fmt.Errorf("user is not authenticated: %s", tokenReview.Status.Error)
	}
	t.reviewResultTTLCache.Add(token, userInfo, 5*time.Minute)
	return userInfo, nil
}

func toTokenReview(token string) *authentication.TokenReview {
	return &authentication.TokenReview{
		Spec: authentication.TokenReviewSpec{
			Token: token,
		},
	}
}

func NewTokens(_ context.Context, k8sClient kubernetes.Interface) Tokens {
	return &tokens{
		tokenReviewClient:    k8sClient.AuthenticationV1().TokenReviews(),
		reviewResultTTLCache: cache.NewLRUExpireCache(1024),
	}
}

func MatchingUsers(userInfoA, userInfoB authentication.UserInfo) bool {
	if userInfoA.Username != userInfoB.Username {
		return false
	}
	if userInfoA.UID != userInfoB.UID {
		return false
	}
	return true
}

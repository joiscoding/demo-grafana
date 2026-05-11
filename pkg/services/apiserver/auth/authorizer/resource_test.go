package authorizer

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apiserver/pkg/authorization/authorizer"

	claims "github.com/grafana/authlib/types"
	"github.com/grafana/grafana/pkg/apimachinery/identity"
)

type fakeAccessChecker struct {
	res claims.CheckResponse
	err error

	gotInfo claims.AuthInfo
	gotReq  claims.CheckRequest
	called  int
}

func (f *fakeAccessChecker) Check(_ context.Context, info claims.AuthInfo, req claims.CheckRequest, _ string) (claims.CheckResponse, error) {
	f.called++
	f.gotInfo = info
	f.gotReq = req
	return f.res, f.err
}

func TestResourceAuthorizer(t *testing.T) {
	t.Run("non resource request returns no opinion without calling the checker", func(t *testing.T) {
		c := &fakeAccessChecker{}
		auth := NewResourceAuthorizer(c)

		dec, reason, err := auth.Authorize(context.Background(), &fakeAttributes{isResourceRequest: false})

		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionNoOpinion, dec)
		require.Equal(t, "", reason)
		require.Zero(t, c.called)
	})

	t.Run("missing identity is denied with an error", func(t *testing.T) {
		c := &fakeAccessChecker{}
		auth := NewResourceAuthorizer(c)

		dec, _, err := auth.Authorize(context.Background(), &fakeAttributes{isResourceRequest: true})

		require.Error(t, err)
		require.Equal(t, authorizer.DecisionDeny, dec)
		require.Zero(t, c.called)
	})

	t.Run("checker error surfaces as deny", func(t *testing.T) {
		c := &fakeAccessChecker{err: errors.New("boom")}
		auth := NewResourceAuthorizer(c)
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{Type: claims.TypeUser, OrgID: 1})

		dec, _, err := auth.Authorize(ctx, &fakeAttributes{isResourceRequest: true})

		require.EqualError(t, err, "boom")
		require.Equal(t, authorizer.DecisionDeny, dec)
		require.Equal(t, 1, c.called)
	})

	t.Run("disallowed response is denied", func(t *testing.T) {
		c := &fakeAccessChecker{res: claims.CheckResponse{Allowed: false}}
		auth := NewResourceAuthorizer(c)
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{Type: claims.TypeUser, OrgID: 1})

		dec, reason, err := auth.Authorize(ctx, &fakeAttributes{isResourceRequest: true})

		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionDeny, dec)
		require.Equal(t, "unauthorized request", reason)
	})

	t.Run("allowed response forwards attributes and returns allow", func(t *testing.T) {
		c := &fakeAccessChecker{res: claims.CheckResponse{Allowed: true}}
		auth := NewResourceAuthorizer(c)
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{Type: claims.TypeUser, OrgID: 1})

		attrs := &fakeAttributes{
			isResourceRequest: true,
			verb:              "get",
			apiGroup:          "dashboard.grafana.app",
			resource:          "dashboards",
			namespace:         "default",
			name:              "d1",
			subresource:       "history",
			path:              "/apis/dashboard.grafana.app/v1/dashboards/d1",
		}

		dec, reason, err := auth.Authorize(ctx, attrs)

		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionAllow, dec)
		require.Equal(t, "", reason)
		require.Equal(t, claims.CheckRequest{
			Verb:        "get",
			Group:       "dashboard.grafana.app",
			Resource:    "dashboards",
			Namespace:   "default",
			Name:        "d1",
			Subresource: "history",
			Path:        "/apis/dashboard.grafana.app/v1/dashboards/d1",
		}, c.gotReq)
		require.NotNil(t, c.gotInfo)
	})
}

package authorizer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apiserver/pkg/authorization/authorizer"

	"github.com/grafana/authlib/authn"
	claims "github.com/grafana/authlib/types"
	"github.com/grafana/grafana/pkg/apimachinery/identity"
)

func newServiceIdentity(t *testing.T, typ claims.IdentityType, perms, delegated []string) context.Context {
	t.Helper()
	return identity.WithRequester(context.Background(), &identity.StaticRequester{
		Type:  typ,
		OrgID: 1,
		AccessTokenClaims: &authn.Claims[authn.AccessTokenClaims]{
			Rest: authn.AccessTokenClaims{
				Permissions:          perms,
				DelegatedPermissions: delegated,
			},
		},
	})
}

func TestServiceAuthorizer(t *testing.T) {
	auth := NewServiceAuthorizer()
	require.NotNil(t, auth)

	t.Run("non resource request returns no opinion", func(t *testing.T) {
		dec, reason, err := auth.Authorize(context.Background(), &fakeAttributes{isResourceRequest: false})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionNoOpinion, dec)
		require.Equal(t, "", reason)
	})

	t.Run("missing identity is denied with an error", func(t *testing.T) {
		dec, _, err := auth.Authorize(context.Background(), &fakeAttributes{isResourceRequest: true})
		require.Error(t, err)
		require.Equal(t, authorizer.DecisionDeny, dec)
	})

	t.Run("access policy with matching token permission is allowed", func(t *testing.T) {
		ctx := newServiceIdentity(t, claims.TypeAccessPolicy,
			[]string{"dashboard.grafana.app/dashboards:get"}, nil)
		dec, reason, err := auth.Authorize(ctx, &fakeAttributes{
			isResourceRequest: true,
			apiGroup:          "dashboard.grafana.app",
			resource:          "dashboards",
			verb:              "get",
		})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionAllow, dec)
		require.Equal(t, "", reason)
	})

	t.Run("access policy lacking permission is denied", func(t *testing.T) {
		ctx := newServiceIdentity(t, claims.TypeAccessPolicy,
			[]string{"dashboard.grafana.app/dashboards:get"}, nil)
		dec, reason, err := auth.Authorize(ctx, &fakeAttributes{
			isResourceRequest: true,
			apiGroup:          "dashboard.grafana.app",
			resource:          "dashboards",
			verb:              "create",
		})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionDeny, dec)
		require.Equal(t, "calling service lacks required permissions", reason)
	})

	t.Run("on behalf of user uses delegated permissions", func(t *testing.T) {
		ctx := newServiceIdentity(t, claims.TypeUser, nil,
			[]string{"dashboard.grafana.app/dashboards:get"})
		dec, _, err := auth.Authorize(ctx, &fakeAttributes{
			isResourceRequest: true,
			apiGroup:          "dashboard.grafana.app",
			resource:          "dashboards",
			verb:              "get",
		})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionAllow, dec)
	})

	t.Run("on behalf of user lacking delegated permission is denied", func(t *testing.T) {
		ctx := newServiceIdentity(t, claims.TypeUser,
			[]string{"dashboard.grafana.app/dashboards:get"}, nil)
		dec, _, err := auth.Authorize(ctx, &fakeAttributes{
			isResourceRequest: true,
			apiGroup:          "dashboard.grafana.app",
			resource:          "dashboards",
			verb:              "get",
		})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionDeny, dec)
	})
}

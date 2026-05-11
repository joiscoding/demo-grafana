package authorizer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apiserver/pkg/authorization/authorizer"

	claims "github.com/grafana/authlib/types"
	"github.com/grafana/grafana/pkg/apimachinery/identity"
)

func TestNamespaceAuthorizer(t *testing.T) {
	auth := newNamespaceAuthorizer()

	t.Run("missing identity is denied", func(t *testing.T) {
		dec, reason, err := auth.Authorize(context.Background(), &fakeAttributes{isResourceRequest: true})
		require.Error(t, err)
		require.Equal(t, authorizer.DecisionDeny, dec)
		require.Equal(t, "missing auth info", reason)
	})

	t.Run("grafana admins always get no opinion", func(t *testing.T) {
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{
			IsGrafanaAdmin: true,
			OrgID:          1,
			Namespace:      "default",
		})
		dec, reason, err := auth.Authorize(ctx, &fakeAttributes{
			isResourceRequest: true,
			namespace:         "stacks-99",
		})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionNoOpinion, dec)
		require.Equal(t, "", reason)
	})

	t.Run("non resource requests are skipped", func(t *testing.T) {
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{
			OrgID:     1,
			Namespace: "default",
			Type:      claims.TypeUser,
		})
		dec, _, err := auth.Authorize(ctx, &fakeAttributes{isResourceRequest: false, namespace: "default"})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionNoOpinion, dec)
	})

	t.Run("anonymous identity is delegated to the next authorizer", func(t *testing.T) {
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{
			Type:      claims.TypeAnonymous,
			OrgID:     1,
			Namespace: "default",
		})
		dec, _, err := auth.Authorize(ctx, &fakeAttributes{
			isResourceRequest: true,
			namespace:         "default",
		})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionNoOpinion, dec)
	})

	t.Run("invalid namespace string is denied", func(t *testing.T) {
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{
			Type:      claims.TypeUser,
			OrgID:     1,
			Namespace: "default",
		})
		dec, reason, err := auth.Authorize(ctx, &fakeAttributes{
			isResourceRequest: true,
			namespace:         "stacks-abc",
		})
		require.Error(t, err)
		require.Equal(t, authorizer.DecisionDeny, dec)
		require.Equal(t, "invalid namespace", reason)
	})

	t.Run("cluster scoped requests get no opinion", func(t *testing.T) {
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{
			Type:      claims.TypeUser,
			OrgID:     1,
			Namespace: "default",
		})
		dec, reason, err := auth.Authorize(ctx, &fakeAttributes{
			isResourceRequest: true,
			namespace:         "",
		})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionNoOpinion, dec)
		require.Equal(t, "", reason)
	})

	t.Run("mismatched org is denied", func(t *testing.T) {
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{
			Type:      claims.TypeUser,
			OrgID:     2,
			Namespace: "org-2",
		})
		dec, reason, err := auth.Authorize(ctx, &fakeAttributes{
			isResourceRequest: true,
			namespace:         "default",
		})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionDeny, dec)
		require.Equal(t, "invalid org", reason)
	})

	t.Run("mismatched namespace is denied", func(t *testing.T) {
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{
			Type:      claims.TypeUser,
			OrgID:     1,
			Namespace: "stacks-1",
		})
		dec, reason, err := auth.Authorize(ctx, &fakeAttributes{
			isResourceRequest: true,
			namespace:         "default",
		})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionDeny, dec)
		require.Equal(t, "invalid namespace", reason)
	})

	t.Run("matching org and namespace returns no opinion", func(t *testing.T) {
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{
			Type:      claims.TypeUser,
			OrgID:     1,
			Namespace: "default",
		})
		dec, reason, err := auth.Authorize(ctx, &fakeAttributes{
			isResourceRequest: true,
			namespace:         "default",
		})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionNoOpinion, dec)
		require.Equal(t, "", reason)
	})
}

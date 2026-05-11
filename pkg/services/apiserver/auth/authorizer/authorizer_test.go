package authorizer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8suser "k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"

	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/services/org"
)

func TestNewAllowAuthorizer(t *testing.T) {
	auth := NewAllowAuthorizer()
	require.NotNil(t, auth)

	t.Run("resource requests are allowed", func(t *testing.T) {
		dec, reason, err := auth.Authorize(context.Background(), &fakeAttributes{isResourceRequest: true})
		require.Equal(t, authorizer.DecisionAllow, dec)
		require.Equal(t, "", reason)
		require.NoError(t, err)
	})

	t.Run("non-resource requests get no opinion", func(t *testing.T) {
		dec, reason, err := auth.Authorize(context.Background(), &fakeAttributes{isResourceRequest: false})
		require.Equal(t, authorizer.DecisionNoOpinion, dec)
		require.Equal(t, "", reason)
		require.NoError(t, err)
	})
}

func TestAuthorizerForAPI(t *testing.T) {
	called := false
	apis := map[string]authorizer.Authorizer{
		"my.group/v1": authorizer.AuthorizerFunc(func(_ context.Context, _ authorizer.Attributes) (authorizer.Decision, string, error) {
			called = true
			return authorizer.DecisionAllow, "matched", nil
		}),
	}
	a := &authorizerForAPI{apis: apis}

	t.Run("forwards to registered authorizer", func(t *testing.T) {
		called = false
		dec, reason, err := a.Authorize(context.Background(), &fakeAttributes{apiGroup: "my.group", apiVersion: "v1"})
		require.NoError(t, err)
		require.True(t, called)
		require.Equal(t, authorizer.DecisionAllow, dec)
		require.Equal(t, "matched", reason)
	})

	t.Run("unknown group returns no opinion", func(t *testing.T) {
		dec, reason, err := a.Authorize(context.Background(), &fakeAttributes{apiGroup: "other", apiVersion: "v1"})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionNoOpinion, dec)
		require.Equal(t, "", reason)
	})
}

func TestGrafanaAuthorizer(t *testing.T) {
	t.Run("constructor wires the union chain", func(t *testing.T) {
		ga := NewGrafanaBuiltInSTAuthorizer()
		require.NotNil(t, ga)
		require.NotNil(t, ga.apis)
		require.NotNil(t, ga.auth)
	})

	t.Run("impersonate is denied before any identity is checked", func(t *testing.T) {
		ga := NewGrafanaBuiltInSTAuthorizer()
		dec, reason, err := ga.Authorize(context.Background(), &fakeAttributes{verb: "impersonate"})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionDeny, dec)
		require.Equal(t, "user impersonation is not supported", reason)
	})

	t.Run("system:masters group bypasses other checks", func(t *testing.T) {
		ga := NewGrafanaBuiltInSTAuthorizer()
		dec, _, err := ga.Authorize(context.Background(), &fakeAttributes{
			verb:              "get",
			isResourceRequest: true,
			user: &k8suser.DefaultInfo{
				Name:   "root",
				Groups: []string{k8suser.SystemPrivilegedGroup},
			},
		})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionAllow, dec)
	})

	t.Run("register dispatches to API authorizer before the role fallback", func(t *testing.T) {
		ga := NewGrafanaBuiltInSTAuthorizer()
		ga.Register(schema.GroupVersion{Group: "custom.grafana.app", Version: "v1"},
			authorizer.AuthorizerFunc(func(_ context.Context, _ authorizer.Attributes) (authorizer.Decision, string, error) {
				return authorizer.DecisionDeny, "custom-api-denied", nil
			}),
		)

		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{
			OrgID:     1,
			Namespace: "default",
			OrgRole:   org.RoleAdmin,
		})
		dec, reason, err := ga.Authorize(ctx, &fakeAttributes{
			verb:              "get",
			apiGroup:          "custom.grafana.app",
			apiVersion:        "v1",
			namespace:         "default",
			isResourceRequest: true,
		})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionDeny, dec)
		require.Equal(t, "custom-api-denied", reason)
	})

	t.Run("falls through to role authorizer when no API authorizer is registered", func(t *testing.T) {
		ga := NewGrafanaBuiltInSTAuthorizer()
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{
			OrgID:     1,
			Namespace: "default",
			OrgRole:   org.RoleAdmin,
		})
		dec, _, err := ga.Authorize(ctx, &fakeAttributes{
			verb:              "get",
			apiGroup:          "anything.grafana.app",
			apiVersion:        "v1",
			namespace:         "default",
			isResourceRequest: true,
		})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionAllow, dec)
	})

	t.Run("missing identity is denied by the namespace authorizer", func(t *testing.T) {
		ga := NewGrafanaBuiltInSTAuthorizer()
		dec, _, err := ga.Authorize(context.Background(), &fakeAttributes{
			verb:              "get",
			isResourceRequest: true,
		})
		require.Error(t, err)
		require.Equal(t, authorizer.DecisionDeny, dec)
	})
}

package authorizer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apiserver/pkg/authorization/authorizer"

	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/services/org"
)

func TestRoleAuthorizer(t *testing.T) {
	auth := NewRoleAuthorizer()
	require.NotNil(t, auth)

	t.Run("missing identity is denied", func(t *testing.T) {
		dec, reason, err := auth.Authorize(context.Background(), &fakeAttributes{verb: "get"})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionDeny, dec)
		require.Contains(t, reason, "error getting signed in user")
	})

	t.Run("admin role allows any verb", func(t *testing.T) {
		ctx := withRole(org.RoleAdmin)
		for _, verb := range []string{"get", "create", "delete", "weird-verb"} {
			dec, _, err := auth.Authorize(ctx, &fakeAttributes{verb: verb})
			require.NoError(t, err, verb)
			require.Equal(t, authorizer.DecisionAllow, dec, verb)
		}
	})

	t.Run("editor role allows write verbs and denies others", func(t *testing.T) {
		ctx := withRole(org.RoleEditor)
		allowed := []string{"get", "list", "watch", "create", "update", "patch", "delete", "put", "post"}
		for _, verb := range allowed {
			dec, _, err := auth.Authorize(ctx, &fakeAttributes{verb: verb})
			require.NoError(t, err, verb)
			require.Equal(t, authorizer.DecisionAllow, dec, verb)
		}

		dec, reason, err := auth.Authorize(ctx, &fakeAttributes{verb: "deletecollection", resource: "things", path: "/api"})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionDeny, dec)
		require.Contains(t, reason, "Grafana org role (Editor)")
		require.Contains(t, reason, "things")
		require.Contains(t, reason, "/api")
	})

	t.Run("viewer role allows reads and denies writes", func(t *testing.T) {
		ctx := withRole(org.RoleViewer)
		for _, verb := range []string{"get", "list", "watch"} {
			dec, _, err := auth.Authorize(ctx, &fakeAttributes{verb: verb})
			require.NoError(t, err, verb)
			require.Equal(t, authorizer.DecisionAllow, dec, verb)
		}

		dec, reason, err := auth.Authorize(ctx, &fakeAttributes{verb: "create"})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionDeny, dec)
		require.Contains(t, reason, "Viewer")
	})

	t.Run("none role denies for unlisted api groups", func(t *testing.T) {
		ctx := withRole(org.RoleNone)
		dec, reason, err := auth.Authorize(ctx, &fakeAttributes{verb: "get", apiGroup: "dashboard.grafana.app"})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionDeny, dec)
		require.Contains(t, reason, "None")
	})

	t.Run("none role allows reads for allowlisted api groups", func(t *testing.T) {
		ctx := withRole(org.RoleNone)
		for _, group := range orgRoleNoneAsViewerAPIGroups {
			for _, verb := range []string{"get", "list", "watch"} {
				dec, _, err := auth.Authorize(ctx, &fakeAttributes{verb: verb, apiGroup: group})
				require.NoError(t, err, group, verb)
				require.Equal(t, authorizer.DecisionAllow, dec, group, verb)
			}

			dec, reason, err := auth.Authorize(ctx, &fakeAttributes{verb: "create", apiGroup: group})
			require.NoError(t, err)
			require.Equal(t, authorizer.DecisionDeny, dec)
			require.Contains(t, reason, "None")
		}
	})

	t.Run("unknown role is denied with empty reason", func(t *testing.T) {
		ctx := withRole(identity.RoleType("totally-unknown"))
		dec, reason, err := auth.Authorize(ctx, &fakeAttributes{verb: "get"})
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionDeny, dec)
		require.Equal(t, "", reason)
	})
}

func withRole(role identity.RoleType) context.Context {
	return identity.WithRequester(context.Background(), &identity.StaticRequester{
		OrgID:   1,
		OrgRole: role,
	})
}

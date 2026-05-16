package authorizer

import (
	"context"
	"testing"

	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8suser "k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
)

func TestGrafanaAuthorizer_RegisterAndAuthorize(t *testing.T) {
	ga := NewGrafanaBuiltInSTAuthorizer()
	gv := schema.GroupVersion{Group: "test.grafana.app", Version: "v1"}

	var apiCalled bool
	ga.Register(gv, authorizer.AuthorizerFunc(func(ctx context.Context, attr authorizer.Attributes) (authorizer.Decision, string, error) {
		apiCalled = true
		return authorizer.DecisionDeny, "api denied", nil
	}))

	attrs := resourceAttrs(func(a *authorizer.AttributesRecord) {
		a.APIGroup = gv.Group
		a.APIVersion = gv.Version
	})
	ctx := ctxWithRequester(testRequester(func(r *identity.StaticRequester) {
		r.OrgRole = org.RoleAdmin
	}))

	decision, reason, err := ga.Authorize(ctx, attrs)
	require.NoError(t, err)
	require.True(t, apiCalled)
	require.Equal(t, authorizer.DecisionDeny, decision)
	require.Equal(t, "api denied", reason)
}

func TestGrafanaBuiltInSTAuthorizer_chain(t *testing.T) {
	ga := NewGrafanaBuiltInSTAuthorizer()

	tests := []struct {
		name         string
		ctx          context.Context
		attrs        authorizer.Attributes
		wantDecision authorizer.Decision
		wantReason   string
	}{
		{
			name:         "impersonate is denied",
			ctx:          ctxWithRequester(testRequester(nil)),
			attrs:        resourceAttrs(func(a *authorizer.AttributesRecord) { a.Verb = "impersonate" }),
			wantDecision: authorizer.DecisionDeny,
			wantReason:   "user impersonation is not supported",
		},
		{
			name: "system masters group is allowed",
			ctx:  context.Background(),
			attrs: resourceAttrs(func(a *authorizer.AttributesRecord) {
				a.User = &k8suser.DefaultInfo{
					Name:   "admin",
					Groups: []string{k8suser.SystemPrivilegedGroup},
				}
			}),
			wantDecision: authorizer.DecisionAllow,
		},
		{
			name:         "editor on unknown API is allowed by role authorizer",
			ctx:          ctxWithRequester(testRequester(func(r *identity.StaticRequester) { r.OrgRole = org.RoleEditor })),
			attrs:        resourceAttrs(func(a *authorizer.AttributesRecord) { a.Verb = "get" }),
			wantDecision: authorizer.DecisionAllow,
		},
		{
			name:         "viewer cannot create",
			ctx:          ctxWithRequester(testRequester(func(r *identity.StaticRequester) { r.OrgRole = org.RoleViewer })),
			attrs:        resourceAttrs(func(a *authorizer.AttributesRecord) { a.Verb = "create" }),
			wantDecision: authorizer.DecisionDeny,
			wantReason: errorMessageForGrafanaOrgRole(org.RoleViewer, resourceAttrs(func(a *authorizer.AttributesRecord) {
				a.Verb = "create"
			})),
		},
		{
			name:         "namespace org mismatch denies before role",
			ctx:          ctxWithRequester(testRequester(func(r *identity.StaticRequester) { r.OrgRole = org.RoleAdmin })),
			attrs:        resourceAttrs(func(a *authorizer.AttributesRecord) { a.Namespace = "org-99" }),
			wantDecision: authorizer.DecisionDeny,
			wantReason:   "invalid org",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, reason, err := ga.Authorize(tt.ctx, tt.attrs)
			require.NoError(t, err)
			require.Equal(t, tt.wantDecision, decision)
			require.Equal(t, tt.wantReason, reason)
		})
	}
}

func TestAuthorizerForAPI_Authorize(t *testing.T) {
	apis := make(map[string]authorizer.Authorizer)
	apiAuth := &authorizerForAPI{apis: apis}

	t.Run("unregistered API returns no opinion", func(t *testing.T) {
		decision, reason, err := apiAuth.Authorize(context.Background(), resourceAttrs(nil))
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionNoOpinion, decision)
		require.Empty(t, reason)
	})

	t.Run("registered API delegates", func(t *testing.T) {
		gv := schema.GroupVersion{Group: "custom.grafana.app", Version: "v1alpha1"}
		apis[gv.String()] = authorizer.AuthorizerFunc(func(ctx context.Context, attr authorizer.Attributes) (authorizer.Decision, string, error) {
			return authorizer.DecisionAllow, "custom allow", nil
		})

		attrs := resourceAttrs(func(a *authorizer.AttributesRecord) {
			a.APIGroup = gv.Group
			a.APIVersion = gv.Version
		})
		decision, reason, err := apiAuth.Authorize(context.Background(), attrs)
		require.NoError(t, err)
		require.Equal(t, authorizer.DecisionAllow, decision)
		require.Equal(t, "custom allow", reason)
	})
}

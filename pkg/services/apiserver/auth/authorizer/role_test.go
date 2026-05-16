package authorizer

import (
	"context"
	"fmt"
	"testing"

	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/services/org"
	"github.com/stretchr/testify/require"
	"k8s.io/apiserver/pkg/authorization/authorizer"
)

func TestNewRoleAuthorizer(t *testing.T) {
	auth := NewRoleAuthorizer()
	require.NotNil(t, auth)
}

func TestRoleAuthorizer_Authorize(t *testing.T) {
	auth := NewRoleAuthorizer()
	attrs := resourceAttrs(func(a *authorizer.AttributesRecord) {
		a.Verb = "get"
		a.Resource = "dashboards"
	})

	tests := []struct {
		name         string
		ctx          context.Context
		attrs        authorizer.Attributes
		wantDecision authorizer.Decision
		wantReason   string
	}{
		{
			name:         "missing requester denies",
			ctx:          context.Background(),
			attrs:        attrs,
			wantDecision: authorizer.DecisionDeny,
			wantReason:   "error getting signed in user: a Requester was not found in the context",
		},
		{
			name:         "admin allows any verb",
			ctx:          ctxWithRequester(testRequester(func(r *identity.StaticRequester) { r.OrgRole = org.RoleAdmin })),
			attrs:        resourceAttrs(func(a *authorizer.AttributesRecord) { a.Verb = "impersonate" }),
			wantDecision: authorizer.DecisionAllow,
		},
		{
			name:         "editor allows standard write verbs",
			ctx:          ctxWithRequester(testRequester(func(r *identity.StaticRequester) { r.OrgRole = org.RoleEditor })),
			attrs:        resourceAttrs(func(a *authorizer.AttributesRecord) { a.Verb = "patch" }),
			wantDecision: authorizer.DecisionAllow,
		},
		{
			name:         "editor denies unsupported verb",
			ctx:          ctxWithRequester(testRequester(func(r *identity.StaticRequester) { r.OrgRole = org.RoleEditor })),
			attrs:        resourceAttrs(func(a *authorizer.AttributesRecord) { a.Verb = "impersonate" }),
			wantDecision: authorizer.DecisionDeny,
			wantReason:   errorMessageForGrafanaOrgRole(org.RoleEditor, resourceAttrs(func(a *authorizer.AttributesRecord) { a.Verb = "impersonate" })),
		},
		{
			name:         "viewer allows read verbs",
			ctx:          ctxWithRequester(testRequester(func(r *identity.StaticRequester) { r.OrgRole = org.RoleViewer })),
			attrs:        resourceAttrs(func(a *authorizer.AttributesRecord) { a.Verb = "watch" }),
			wantDecision: authorizer.DecisionAllow,
		},
		{
			name:         "viewer denies write verbs",
			ctx:          ctxWithRequester(testRequester(func(r *identity.StaticRequester) { r.OrgRole = org.RoleViewer })),
			attrs:        resourceAttrs(func(a *authorizer.AttributesRecord) { a.Verb = "create" }),
			wantDecision: authorizer.DecisionDeny,
			wantReason:   errorMessageForGrafanaOrgRole(org.RoleViewer, resourceAttrs(func(a *authorizer.AttributesRecord) { a.Verb = "create" })),
		},
		{
			name:         "none role denies by default",
			ctx:          ctxWithRequester(testRequester(func(r *identity.StaticRequester) { r.OrgRole = org.RoleNone })),
			attrs:        resourceAttrs(func(a *authorizer.AttributesRecord) { a.Verb = "get" }),
			wantDecision: authorizer.DecisionDeny,
			wantReason:   errorMessageForGrafanaOrgRole(org.RoleNone, resourceAttrs(func(a *authorizer.AttributesRecord) { a.Verb = "get" })),
		},
		{
			name: "none role hotfix allows viewer verbs on playlist API",
			ctx:  ctxWithRequester(testRequester(func(r *identity.StaticRequester) { r.OrgRole = org.RoleNone })),
			attrs: resourceAttrs(func(a *authorizer.AttributesRecord) {
				a.Verb = "list"
				a.APIGroup = "playlist.grafana.app"
			}),
			wantDecision: authorizer.DecisionAllow,
		},
		{
			name: "none role hotfix denies write on playlist API",
			ctx:  ctxWithRequester(testRequester(func(r *identity.StaticRequester) { r.OrgRole = org.RoleNone })),
			attrs: resourceAttrs(func(a *authorizer.AttributesRecord) {
				a.Verb = "delete"
				a.APIGroup = "playlist.grafana.app"
			}),
			wantDecision: authorizer.DecisionDeny,
			wantReason: errorMessageForGrafanaOrgRole(org.RoleNone, resourceAttrs(func(a *authorizer.AttributesRecord) {
				a.Verb = "delete"
				a.APIGroup = "playlist.grafana.app"
			})),
		},
		{
			name:         "unknown role denies without reason",
			ctx:          ctxWithRequester(testRequester(func(r *identity.StaticRequester) { r.OrgRole = identity.RoleType("Unknown") })),
			attrs:        attrs,
			wantDecision: authorizer.DecisionDeny,
			wantReason:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, reason, err := auth.Authorize(tt.ctx, tt.attrs)
			require.Equal(t, tt.wantDecision, decision)
			require.Equal(t, tt.wantReason, reason)
			require.NoError(t, err)
		})
	}
}

func TestErrorMessageForGrafanaOrgRole(t *testing.T) {
	attrs := resourceAttrs(func(a *authorizer.AttributesRecord) {
		a.Verb = "delete"
		a.Resource = "folders"
		a.Path = "/test/path"
	})
	msg := errorMessageForGrafanaOrgRole(org.RoleViewer, attrs)
	require.Equal(t, fmt.Sprintf("Grafana org role (%s) didn't allow %s access on requested resource=%s, path=%s",
		org.RoleViewer, "delete", "folders", "/test/path"), msg)
}

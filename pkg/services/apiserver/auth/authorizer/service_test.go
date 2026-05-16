package authorizer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apiserver/pkg/authorization/authorizer"
)

func TestNewServiceAuthorizer(t *testing.T) {
	auth := NewServiceAuthorizer()

	tests := []struct {
		name         string
		ctx          context.Context
		attrs        authorizer.Attributes
		wantDecision authorizer.Decision
		wantReason   string
		wantErr      bool
	}{
		{
			name:         "non-resource request defers",
			ctx:          ctxWithAuthInfo(accessPolicyWithPermissions("folder.grafana.app/folders:get")),
			attrs:        resourceAttrs(func(a *authorizer.AttributesRecord) { a.ResourceRequest = false }),
			wantDecision: authorizer.DecisionNoOpinion,
		},
		{
			name:         "missing identity denies",
			ctx:          context.Background(),
			attrs:        resourceAttrs(nil),
			wantDecision: authorizer.DecisionDeny,
			wantErr:      true,
		},
		{
			name: "service with permission allows",
			ctx: ctxWithAuthInfo(accessPolicyWithPermissions(
				"folder.grafana.app/folders:get",
			)),
			attrs:        resourceAttrs(nil),
			wantDecision: authorizer.DecisionAllow,
		},
		{
			name: "service without permission denies",
			ctx: ctxWithAuthInfo(accessPolicyWithPermissions(
				"other.grafana.app/things:get",
			)),
			attrs:        resourceAttrs(nil),
			wantDecision: authorizer.DecisionDeny,
			wantReason:   "calling service lacks required permissions",
		},
		{
			name: "on-behalf-of uses delegated permissions",
			ctx: ctxWithAuthInfo(userWithDelegatedPermissions(
				"folder.grafana.app/folders:get",
			)),
			attrs:        resourceAttrs(nil),
			wantDecision: authorizer.DecisionAllow,
		},
		{
			name: "on-behalf-of without delegated permission denies",
			ctx:  ctxWithAuthInfo(userWithDelegatedPermissions()),
			attrs: resourceAttrs(func(a *authorizer.AttributesRecord) {
				a.Verb = "delete"
			}),
			wantDecision: authorizer.DecisionDeny,
			wantReason:   "calling service lacks required permissions",
		},
		{
			name: "list verb accepts get permission",
			ctx: ctxWithAuthInfo(accessPolicyWithPermissions(
				"folder.grafana.app/folders:get",
			)),
			attrs:        resourceAttrs(func(a *authorizer.AttributesRecord) { a.Verb = "list" }),
			wantDecision: authorizer.DecisionAllow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, reason, err := auth.Authorize(tt.ctx, tt.attrs)
			require.Equal(t, tt.wantDecision, decision)
			require.Equal(t, tt.wantReason, reason)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

package authorizer

import (
	"context"
	"testing"

	"github.com/grafana/authlib/types"
	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/stretchr/testify/require"
	"k8s.io/apiserver/pkg/authorization/authorizer"
)

func TestNamespaceAuthorizer_Authorize(t *testing.T) {
	auth := newNamespaceAuthorizer()

	tests := []struct {
		name         string
		ctx          context.Context
		attrs        authorizer.Attributes
		wantDecision authorizer.Decision
		wantReason   string
		wantErr      bool
	}{
		{
			name:         "missing requester denies",
			ctx:          context.Background(),
			attrs:        resourceAttrs(nil),
			wantDecision: authorizer.DecisionDeny,
			wantReason:   "missing auth info",
			wantErr:      true,
		},
		{
			name: "grafana admin defers",
			ctx: ctxWithRequester(testRequester(func(r *identity.StaticRequester) {
				r.IsGrafanaAdmin = true
			})),
			attrs:        resourceAttrs(nil),
			wantDecision: authorizer.DecisionNoOpinion,
		},
		{
			name: "non-resource request defers",
			ctx:  ctxWithRequester(testRequester(nil)),
			attrs: resourceAttrs(func(a *authorizer.AttributesRecord) {
				a.ResourceRequest = false
			}),
			wantDecision: authorizer.DecisionNoOpinion,
		},
		{
			name: "anonymous user defers",
			ctx: ctxWithRequester(testRequester(func(r *identity.StaticRequester) {
				r.Type = types.TypeAnonymous
			})),
			attrs:        resourceAttrs(nil),
			wantDecision: authorizer.DecisionNoOpinion,
		},
		{
			name:         "invalid namespace denies",
			ctx:          ctxWithRequester(testRequester(nil)),
			attrs:        resourceAttrs(func(a *authorizer.AttributesRecord) { a.Namespace = "org-invalid" }),
			wantDecision: authorizer.DecisionDeny,
			wantReason:   "invalid namespace",
			wantErr:      true,
		},
		{
			name:         "empty namespace defers for cluster scope",
			ctx:          ctxWithRequester(testRequester(nil)),
			attrs:        resourceAttrs(func(a *authorizer.AttributesRecord) { a.Namespace = "" }),
			wantDecision: authorizer.DecisionNoOpinion,
		},
		{
			name: "org mismatch denies",
			ctx: ctxWithRequester(testRequester(func(r *identity.StaticRequester) {
				r.OrgID = 1
				r.Namespace = "org-1"
			})),
			attrs:        resourceAttrs(func(a *authorizer.AttributesRecord) { a.Namespace = "org-2" }),
			wantDecision: authorizer.DecisionDeny,
			wantReason:   "invalid org",
		},
		{
			name: "namespace mismatch denies",
			ctx: ctxWithRequester(testRequester(func(r *identity.StaticRequester) {
				r.Namespace = "org-3"
			})),
			attrs:        resourceAttrs(func(a *authorizer.AttributesRecord) { a.Namespace = "org-2" }),
			wantDecision: authorizer.DecisionDeny,
			wantReason:   "invalid namespace",
		},
		{
			name: "matching org namespace defers",
			ctx:  ctxWithRequester(testRequester(nil)),
			attrs: resourceAttrs(func(a *authorizer.AttributesRecord) {
				a.Namespace = "org-2"
			}),
			wantDecision: authorizer.DecisionNoOpinion,
		},
		{
			name: "wildcard identity namespace allows any org namespace",
			ctx: ctxWithRequester(testRequester(func(r *identity.StaticRequester) {
				r.Namespace = "*"
			})),
			attrs:        resourceAttrs(func(a *authorizer.AttributesRecord) { a.Namespace = "org-2" }),
			wantDecision: authorizer.DecisionNoOpinion,
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

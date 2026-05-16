package authorizer

import (
	"context"
	"testing"

	"github.com/grafana/authlib/types"
	"github.com/stretchr/testify/require"
	"k8s.io/apiserver/pkg/authorization/authorizer"
)

func TestResourceAuthorizer_Authorize(t *testing.T) {
	user := testRequester(nil)
	attrs := resourceAttrs(func(a *authorizer.AttributesRecord) {
		a.Verb = "create"
		a.Name = "fold-1"
		a.Subresource = "status"
	})

	tests := []struct {
		name         string
		ctx          context.Context
		checker      *mockAccessChecker
		attrs        authorizer.Attributes
		wantDecision authorizer.Decision
		wantReason   string
		wantErr      bool
	}{
		{
			name:         "non-resource request defers",
			ctx:          ctxWithAuthInfo(user),
			checker:      &mockAccessChecker{},
			attrs:        resourceAttrs(func(a *authorizer.AttributesRecord) { a.ResourceRequest = false }),
			wantDecision: authorizer.DecisionNoOpinion,
		},
		{
			name:         "missing identity denies",
			ctx:          context.Background(),
			checker:      &mockAccessChecker{},
			attrs:        attrs,
			wantDecision: authorizer.DecisionDeny,
			wantErr:      true,
		},
		{
			name: "access check error denies",
			ctx:  ctxWithAuthInfo(user),
			checker: &mockAccessChecker{checkFunc: func(ctx context.Context, ident types.AuthInfo, req types.CheckRequest, folder string) (types.CheckResponse, error) {
				return types.CheckResponse{}, errCheckFailed
			}},
			attrs:        attrs,
			wantDecision: authorizer.DecisionDeny,
			wantErr:      true,
		},
		{
			name: "denied check returns unauthorized",
			ctx:  ctxWithAuthInfo(user),
			checker: &mockAccessChecker{checkFunc: func(ctx context.Context, ident types.AuthInfo, req types.CheckRequest, folder string) (types.CheckResponse, error) {
				return types.CheckResponse{Allowed: false}, nil
			}},
			attrs:        attrs,
			wantDecision: authorizer.DecisionDeny,
			wantReason:   "unauthorized request",
		},
		{
			name: "allowed check permits request",
			ctx:  ctxWithAuthInfo(user),
			checker: &mockAccessChecker{checkFunc: func(ctx context.Context, ident types.AuthInfo, req types.CheckRequest, folder string) (types.CheckResponse, error) {
				require.Equal(t, user, ident)
				require.Equal(t, "create", req.Verb)
				require.Equal(t, "folder.grafana.app", req.Group)
				require.Equal(t, "folders", req.Resource)
				require.Equal(t, "org-2", req.Namespace)
				require.Equal(t, "fold-1", req.Name)
				require.Equal(t, "status", req.Subresource)
				require.Equal(t, attrs.GetPath(), req.Path)
				require.Empty(t, folder)
				return types.CheckResponse{Allowed: true}, nil
			}},
			attrs:        attrs,
			wantDecision: authorizer.DecisionAllow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewResourceAuthorizer(tt.checker)
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

func TestNewResourceAuthorizer(t *testing.T) {
	checker := &mockAccessChecker{}
	auth := NewResourceAuthorizer(checker)
	require.Equal(t, ResourceAuthorizer{checker}, auth)
}

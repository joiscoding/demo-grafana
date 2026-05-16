package authorizer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apiserver/pkg/authorization/authorizer"
)

func TestNewAllowAuthorizer(t *testing.T) {
	auth := NewAllowAuthorizer()

	tests := []struct {
		name         string
		attrs        authorizer.Attributes
		wantDecision authorizer.Decision
		wantReason   string
		wantErr      bool
	}{
		{
			name:         "non-resource request returns no opinion",
			attrs:        resourceAttrs(func(a *authorizer.AttributesRecord) { a.ResourceRequest = false }),
			wantDecision: authorizer.DecisionNoOpinion,
		},
		{
			name:         "resource request is allowed",
			attrs:        resourceAttrs(nil),
			wantDecision: authorizer.DecisionAllow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, reason, err := auth.Authorize(context.Background(), tt.attrs)
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

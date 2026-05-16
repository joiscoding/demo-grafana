package request_test

import (
	"context"
	"testing"

	claims "github.com/grafana/authlib/types"
	"github.com/stretchr/testify/require"
	k8srequest "k8s.io/apiserver/pkg/endpoints/request"

	"github.com/grafana/grafana/pkg/apimachinery/identity"
	"github.com/grafana/grafana/pkg/services/apiserver/endpoints/request"
	"github.com/grafana/grafana/pkg/setting"
)

func TestNamespaceMapper(t *testing.T) {
	tests := []struct {
		name     string
		cfg      string
		orgId    int64
		expected string
	}{
		{
			name:     "default namespace",
			orgId:    1,
			expected: "default",
		},
		{
			name:     "with org",
			orgId:    123,
			expected: "org-123",
		},
		// an invalid use-case, but just documenting that it's handled as stacks-0
		// this currently prevents the need to have the Mapper return (mapped, err) instead of just mapped.
		// err checking is avoided for now to keep the usage fluent
		{
			name:     "with stackId",
			cfg:      "abc",
			orgId:    123,        // ignored
			expected: "stacks-0", // we parse to int and default to 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := request.GetNamespaceMapper(&setting.Cfg{StackID: tt.cfg})
			require.Equal(t, tt.expected, mapper(tt.orgId))
		})
	}
}

func TestGetNamespaceMapper_nilCfg(t *testing.T) {
	mapper := request.GetNamespaceMapper(nil)
	require.Equal(t, "default", mapper(1))
	require.Equal(t, "org-123", mapper(123))
}

func TestGetNamespaceMapper_validStackID(t *testing.T) {
	mapper := request.GetNamespaceMapper(&setting.Cfg{StackID: "42"})
	require.Equal(t, "stacks-42", mapper(1))
	require.Equal(t, "stacks-42", mapper(999))
}

func TestNamespaceInfoFrom(t *testing.T) {
	tests := []struct {
		name         string
		namespace    string
		requireOrgID bool
		wantOrgID    int64
		wantErr      string
	}{
		{
			name:         "default namespace with required org",
			namespace:    "default",
			requireOrgID: true,
			wantOrgID:    1,
		},
		{
			name:         "org namespace",
			namespace:    "org-123",
			requireOrgID: true,
			wantOrgID:    123,
		},
		{
			name:         "org not required with empty namespace",
			namespace:    "",
			requireOrgID: false,
			wantOrgID:    -1,
		},
		{
			name:         "missing org when required",
			namespace:    "",
			requireOrgID: true,
			wantOrgID:    -1,
			wantErr:      "expected valid orgId in namespace",
		},
		{
			name:         "invalid org namespace when required",
			namespace:    "org-invalid",
			requireOrgID: true,
			wantErr:      "invalid org id",
		},
		{
			name:         "cloud stack namespace",
			namespace:    "stacks-99",
			requireOrgID: true,
			wantOrgID:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := k8srequest.WithNamespace(context.Background(), tt.namespace)
			info, err := request.NamespaceInfoFrom(ctx, tt.requireOrgID)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantOrgID, info.OrgID)
			require.Equal(t, tt.namespace, info.Value)
		})
	}
}

func TestOrgIDForList(t *testing.T) {
	t.Run("from namespace", func(t *testing.T) {
		ctx := k8srequest.WithNamespace(context.Background(), "org-42")
		orgID, err := request.OrgIDForList(ctx)
		require.NoError(t, err)
		require.Equal(t, int64(42), orgID)
	})

	t.Run("empty namespace uses requester org", func(t *testing.T) {
		ctx := identity.WithRequester(context.Background(), &identity.StaticRequester{OrgID: 7})
		orgID, err := request.OrgIDForList(ctx)
		require.NoError(t, err)
		require.Equal(t, int64(7), orgID)
	})

	t.Run("empty namespace without requester", func(t *testing.T) {
		orgID, err := request.OrgIDForList(context.Background())
		require.Error(t, err)
		require.Equal(t, int64(-1), orgID)
	})

	t.Run("default namespace", func(t *testing.T) {
		ctx := k8srequest.WithNamespace(context.Background(), claims.OrgNamespaceFormatter(1))
		orgID, err := request.OrgIDForList(ctx)
		require.NoError(t, err)
		require.Equal(t, int64(1), orgID)
	})

	t.Run("invalid namespace", func(t *testing.T) {
		ctx := k8srequest.WithNamespace(context.Background(), "org-invalid")
		orgID, err := request.OrgIDForList(ctx)
		require.Error(t, err)
		require.Equal(t, int64(-1), orgID)
	})
}

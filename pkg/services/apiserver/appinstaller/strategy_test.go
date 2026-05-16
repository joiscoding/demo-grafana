package appinstaller

import (
	"testing"

	"github.com/stretchr/testify/require"
	genericrest "k8s.io/apiserver/pkg/registry/rest"
)

func TestUpdateStrategyWrapper(t *testing.T) {
	tests := []struct {
		name                         string
		delegateAllowCreateOnUpdate  bool
		delegateAllowUnconditional   bool
		wantAllowCreateOnUpdate      bool
		wantAllowUnconditionalUpdate bool
	}{
		{
			name:                         "delegates false but wrapper allows create and unconditional update",
			delegateAllowCreateOnUpdate:  false,
			delegateAllowUnconditional:   false,
			wantAllowCreateOnUpdate:      true,
			wantAllowUnconditionalUpdate: true,
		},
		{
			name:                         "delegates true and wrapper still allows",
			delegateAllowCreateOnUpdate:  true,
			delegateAllowUnconditional:   true,
			wantAllowCreateOnUpdate:      true,
			wantAllowUnconditionalUpdate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delegate := &mockRESTUpdateStrategy{
				allowCreateOnUpdate:    tt.delegateAllowCreateOnUpdate,
				allowUnconditionalUpdate: tt.delegateAllowUnconditional,
			}
			wrapper := &updateStrategyWrapper{RESTUpdateStrategy: delegate}

			require.Equal(t, tt.wantAllowCreateOnUpdate, wrapper.AllowCreateOnUpdate())
			require.Equal(t, tt.wantAllowUnconditionalUpdate, wrapper.AllowUnconditionalUpdate())
		})
	}
}

type mockRESTUpdateStrategy struct {
	genericrest.RESTUpdateStrategy
	allowCreateOnUpdate      bool
	allowUnconditionalUpdate bool
}

func (m *mockRESTUpdateStrategy) AllowCreateOnUpdate() bool {
	return m.allowCreateOnUpdate
}

func (m *mockRESTUpdateStrategy) AllowUnconditionalUpdate() bool {
	return m.allowUnconditionalUpdate
}

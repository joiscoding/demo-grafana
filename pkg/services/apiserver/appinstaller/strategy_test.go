package appinstaller

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateStrategyWrapper(t *testing.T) {
	w := &updateStrategyWrapper{}
	require.True(t, w.AllowCreateOnUpdate())
	require.True(t, w.AllowUnconditionalUpdate())
}

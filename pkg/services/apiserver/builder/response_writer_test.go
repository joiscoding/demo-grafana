package builder

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponseWriterWithStatus(t *testing.T) {
	t.Parallel()

	t.Run("WriteHeader sets status", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		w := newResponseWriterWithStatus(rec)
		w.WriteHeader(http.StatusAccepted)
		require.Equal(t, http.StatusAccepted, w.StatusCode())
	})

	t.Run("Write without WriteHeader defaults to OK", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		w := newResponseWriterWithStatus(rec)
		_, err := w.Write([]byte("ok"))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, w.StatusCode())
	})
}

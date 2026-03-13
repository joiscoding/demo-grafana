package sqlengtest

import (
	"encoding/json"
	"errors"
	"net"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	postgresqlsqleng "github.com/grafana/grafana/pkg/tsdb/grafana-postgresql-datasource/sqleng"
	mssqlsqleng "github.com/grafana/grafana/pkg/tsdb/mssql/sqleng"
	mysqlsqleng "github.com/grafana/grafana/pkg/tsdb/mysql/sqleng"
	"github.com/stretchr/testify/require"
)

type errToHealthCheckResultFunc func(error) (*backend.CheckHealthResult, error)

func sqlSourceErrorHandlers() map[string]errToHealthCheckResultFunc {
	return map[string]errToHealthCheckResultFunc{
		"grafana-postgresql-datasource": postgresqlsqleng.ErrToHealthCheckResult,
		"mssql":                         mssqlsqleng.ErrToHealthCheckResult,
		"mysql":                         mysqlsqleng.ErrToHealthCheckResult,
	}
}

func TestSQLSourcesErrToHealthCheckResultContract(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wantMsgPrefix  string
		wantMsgContain string
	}{
		{
			name:          "nil error fallback",
			wantMsgPrefix: "Internal Server Error",
		},
		{
			name:           "generic error passthrough",
			err:            errors.New("boom"),
			wantMsgContain: "boom",
		},
		{
			name: "network error is user-friendly",
			err: errors.Join(
				errors.New("wrapped"),
				&net.OpError{Op: "dial", Net: "tcp", Err: errors.New("some op")},
			),
			wantMsgPrefix: "Network error: Failed to connect to the server",
		},
	}

	for sourceName, handler := range sqlSourceErrorHandlers() {
		sourceName := sourceName
		handler := handler

		t.Run(sourceName, func(t *testing.T) {
			for _, tt := range tests {
				tt := tt
				t.Run(tt.name, func(t *testing.T) {
					res, err := handler(tt.err)
					require.NoError(t, err)
					require.NotNil(t, res)
					require.Equal(t, backend.HealthStatusError, res.Status)
					require.NotEmpty(t, res.Message)

					if tt.wantMsgPrefix != "" {
						require.Truef(
							t,
							strings.HasPrefix(res.Message, tt.wantMsgPrefix),
							"expected message prefix %q, got %q",
							tt.wantMsgPrefix,
							res.Message,
						)
					}
					if tt.wantMsgContain != "" {
						require.Contains(t, res.Message, tt.wantMsgContain)
					}

					if len(res.JSONDetails) > 0 {
						var details map[string]string
						require.NoError(t, json.Unmarshal(res.JSONDetails, &details))
					}

					// Network errors should include actionable troubleshooting metadata for all SQL sources.
					if tt.name == "network error is user-friendly" {
						var details map[string]string
						require.NoError(t, json.Unmarshal(res.JSONDetails, &details))
						require.NotEmpty(t, details["errorDetailsLink"])
						require.NotEmpty(t, details["verboseMessage"])
					}
				})
			}
		})
	}
}

func TestAllSQLSourceErrorHandlersAreCoveredByContract(t *testing.T) {
	root := repositoryRoot(t)
	handlerPaths, err := filepath.Glob(filepath.Join(root, "pkg", "tsdb", "*", "sqleng", "handler_checkhealth.go"))
	require.NoError(t, err)
	require.NotEmpty(t, handlerPaths)

	actualSources := make(map[string]struct{}, len(handlerPaths))
	for _, path := range handlerPaths {
		sourceName := filepath.Base(filepath.Dir(filepath.Dir(path)))
		actualSources[sourceName] = struct{}{}
	}

	coveredSources := make(map[string]struct{}, len(sqlSourceErrorHandlers()))
	for sourceName := range sqlSourceErrorHandlers() {
		coveredSources[sourceName] = struct{}{}
	}

	var missingCoverage []string
	for sourceName := range actualSources {
		if _, ok := coveredSources[sourceName]; !ok {
			missingCoverage = append(missingCoverage, sourceName)
		}
	}

	var staleCoverage []string
	for sourceName := range coveredSources {
		if _, ok := actualSources[sourceName]; !ok {
			staleCoverage = append(staleCoverage, sourceName)
		}
	}

	sort.Strings(missingCoverage)
	sort.Strings(staleCoverage)

	require.Emptyf(
		t,
		missingCoverage,
		"new SQL source(s) missing error handling contract coverage: %v. Add them to sqlSourceErrorHandlers.",
		missingCoverage,
	)
	require.Emptyf(
		t,
		staleCoverage,
		"contract coverage contains removed/renamed SQL source(s): %v. Clean up sqlSourceErrorHandlers.",
		staleCoverage,
	)
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}

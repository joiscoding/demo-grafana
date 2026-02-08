# Structured logging cleanup

Grafana's `log.Logger` interface uses structured logging with key-value pairs.
The correct usage is a static message string followed by alternating key-value pairs:

```go
logger.Info("Starting application", "name", appName, "version", buildVersion)
```

An audit of the codebase found three categories of `fmt.Sprintf` usage inside logging calls.

## Category 1: Values baked into the message string

**54 call sites across 22 files**

These calls use `fmt.Sprintf` to interpolate dynamic values directly into the message string.
This defeats structured logging because the values can't be filtered, indexed,
or queried as separate fields in log aggregation tools like Loki.

```go
// Before -- values buried in the message string
logger.Info(fmt.Sprintf("reading credential cache lookup: %s", lookupFile))

// After -- values as structured key-value pairs
logger.Info("Reading credential cache lookup", "file", lookupFile)
```

| File | Call sites |
|------|-----------|
| `pkg/services/cloudmigration/cloudmigrationimpl/snapshot_mgmt.go` | 10 |
| `pkg/services/secrets/migrator/reencrypt.go` | 6 |
| `pkg/services/secrets/migrator/rollback.go` | 6 |
| `pkg/setting/setting.go` | 6 |
| `pkg/tsdb/mssql/kerberos/kerberos.go` | 5 |
| `pkg/services/sqlstore/migrations/accesscontrol/dashboard_permissions.go` | 3 |
| `pkg/services/sqlstore/migrations/accesscontrol/alerting.go` | 3 |
| `pkg/tsdb/influxdb/fsql/fsql.go` | 2 |
| `pkg/services/notifications/notifications.go` | 2 |
| `pkg/login/social/connectors/azuread_oauth.go` | 2 |
| `pkg/tsdb/influxdb/influxdb.go` | 1 |
| `pkg/tsdb/influxdb/influxql/influxql.go` | 1 |
| `pkg/tsdb/influxdb/models/model_parser.go` | 1 |
| `pkg/promlib/querydata/request.go` | 1 |
| `pkg/services/sqlstore/migrator/sqlite_dialect.go` | 1 |
| `pkg/services/sqlstore/migrations/ualert/alert_rule_version_guid_mig.go` | 1 |
| `pkg/services/sqlstore/migrations/accesscontrol/action_set_migration.go` | 1 |
| `pkg/services/sqlstore/migrations/accesscontrol/datasource_drilldown_removal.go` | 1 |
| `pkg/services/provisioning/alerting/config_reader.go` | 1 |
| `pkg/services/ngalert/backtesting/engine.go` | 1 |
| `pkg/plugins/manager/installer.go` | 1 |
| `pkg/cmd/grafana-cli/commands/datamigrations/to_unified_storage.go` | 1 |

Five of the calls in `pkg/setting/setting.go` also pass `err` as a bare value
without a key name, which produces broken key-value pairs:

```go
// Before -- fmt.Sprintf in message AND bare err without a key
cfg.Logger.Error(fmt.Sprintf("could not set environment variable '%s'", envVarName), err)

// After -- static message with proper key-value pairs
cfg.Logger.Error("Could not set environment variable", "name", envVarName, "error", err)
```

## Category 2: Unnecessary string conversion of values

**21 call sites across 12 files**

These calls use the correct structured logging pattern (static message with key-value pairs),
but wrap a value in `fmt.Sprintf("%v", ...)` or `fmt.Sprintf("%+v", ...)` before passing it.
This is unnecessary because Grafana's text logger already formats complex types internally.
Pre-converting to a string loses type information, which prevents log tools from treating
the value as its native type (for example, querying numbers with greater-than filters).

```go
// Before -- unnecessary string conversion
s.log.Debug("AzureAD OAuth: extracted groups", "groups", fmt.Sprintf("%v", groups))

// After -- pass the value directly
s.log.Debug("AzureAD OAuth: extracted groups", "groups", groups)
```

| File | Call sites |
|------|-----------|
| `pkg/login/social/connectors/gitlab_oauth.go` | 3 |
| `pkg/services/oauthtoken/oauth_token.go` | 3 |
| `pkg/login/social/connectors/org_role_mapper.go` | 3 |
| `pkg/login/social/connectors/azuread_oauth.go` | 2 |
| `pkg/login/social/connectors/google_oauth.go` | 2 |
| `pkg/services/ldap/ldap.go` | 2 |
| `pkg/login/social/connectors/generic_oauth.go` | 1 |
| `pkg/services/cleanup/cleanup.go` | 1 |
| `pkg/services/authn/authnimpl/sync/user_sync.go` | 1 |
| `pkg/services/accesscontrol/resourcepermissions/api_adapter.go` | 1 |
| `pkg/tsdb/elasticsearch/client/client.go` | 1 |
| `pkg/services/ngalert/sender/notifier.go` | 1 |

## Category 3: Intentional adapter/wrapper code

**3 files (~12 internal call sites) -- no changes needed**

These files exist to bridge external libraries that use format-style logging APIs
(like xorm and the plugin SDK) into Grafana's structured logger.
The `fmt.Sprintf` in these files is intentional and is the core of their translation logic.

| File | Purpose |
|------|---------|
| `pkg/plugins/log/infra_wrapper.go` | Bridges the plugin SDK's `Printf`-style log interface to Grafana's structured logger |
| `pkg/services/sqlstore/logger.go` | Bridges the xorm ORM's format-style log interface to Grafana's structured logger |
| `pkg/util/xorm/syslogger.go` | Bridges xorm's syslog output to Grafana's structured logger |

## Summary

| Category | Description | Call sites | Files |
|----------|-------------|-----------|-------|
| 1 | Values baked into message string | 54 | 22 |
| 2 | Unnecessary string conversion of values | 21 | 12 |
| 3 | Intentional adapter code (no fix needed) | ~12 | 3 |
| **Total to fix** | | **75** | **32** |

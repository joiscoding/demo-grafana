---
name: react-router-migration
description: Migrate Grafana components from locationService to React Router's useNavigate hook. Use when migrating navigation calls, adding useNavigate to components, or when the user mentions React Router migration, locationService.push, or useNavigate.
---

# React Router Migration

Migrate Grafana frontend components from `locationService` to React Router's `useNavigate`, gated behind feature flags for safe rollout.

## Quick Reference

```typescript
// Before (legacy)
import { locationService } from '@grafana/runtime';
locationService.push('/path');

// After (with feature flag)
import { config, locationService } from '@grafana/runtime';
import { useNavigate } from 'react-router-dom-v5-compat';

const navigate = useNavigate();

if (config.featureToggles.yourFeatureFlag) {
  navigate('/path');
} else {
  locationService.push('/path');
}
```

## Migration Workflow

### Step 1: Add Feature Flag (Backend)

Edit `pkg/services/featuremgmt/registry.go` and add to `standardFeatureFlags`:

```go
{
    Name:         "yourFeatureUseNavigate",
    Description:  "Use React Router's useNavigate hook instead of locationService in [feature] pages",
    Stage:        FeatureStageExperimental,
    FrontendOnly: true,
    Owner:        grafanaDashboardsSquad,
    Expression:   "false",
},
```

Place near related feature flags. Then run:

```bash
make gen-feature-toggles
```

### Step 2: Update Component Imports

Add these imports:

```typescript
import { useNavigate } from 'react-router-dom-v5-compat';
import { config, locationService } from '@grafana/runtime';
```

If already importing from `react-router-dom-v5-compat`, add `useNavigate` to existing import:

```typescript
import { useNavigate, useParams } from 'react-router-dom-v5-compat';
```

### Step 3: Add Hook Call

Inside the component function, add the hook call near other hooks:

```typescript
export const YourPage = () => {
  const navigate = useNavigate();
  // ... rest of component
};
```

### Step 4: Update Navigation Calls

Replace each `locationService.push()` call:

```typescript
// Before
locationService.push('/path');

// After
if (config.featureToggles.yourFeatureUseNavigate) {
  navigate('/path');
} else {
  locationService.push('/path');
}
```

For `locationService.replace()`:

```typescript
if (config.featureToggles.yourFeatureUseNavigate) {
  navigate('/path', { replace: true });
} else {
  locationService.replace('/path');
}
```

## Testing Pattern

Mock `useNavigate` in test files:

```typescript
const mockNavigate = jest.fn();
jest.mock('react-router-dom-v5-compat', () => ({
  ...jest.requireActual('react-router-dom-v5-compat'),
  useNavigate: () => mockNavigate,
}));

describe('YourComponent', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockNavigate.mockClear();
    config.featureToggles.yourFeatureUseNavigate = false;
  });

  afterEach(() => {
    config.featureToggles.yourFeatureUseNavigate = false;
  });

  describe('when feature flag is disabled', () => {
    it('should use locationService.push', async () => {
      // ... trigger navigation
      expect(locationService.getLocation().pathname).toEqual('/expected');
      expect(mockNavigate).not.toHaveBeenCalled();
    });
  });

  describe('when feature flag is enabled', () => {
    beforeEach(() => {
      config.featureToggles.yourFeatureUseNavigate = true;
    });

    it('should use navigate', async () => {
      // ... trigger navigation
      expect(mockNavigate).toHaveBeenCalledWith('/expected');
    });
  });
});
```

## Common Errors to Avoid

### 1. Missing Hook Call

Hooks must be called at the top level of components:

```typescript
// BAD - hook inside callback
const onSubmit = async () => {
  const navigate = useNavigate(); // Error!
  navigate('/path');
};

// GOOD - hook at component top level
const navigate = useNavigate();
const onSubmit = async () => {
  navigate('/path');
};
```

### 2. Forgetting to Keep locationService Import

Keep both imports for the feature-flagged fallback:

```typescript
// BAD - removed locationService
import { config } from '@grafana/runtime';

// GOOD - keep both
import { config, locationService } from '@grafana/runtime';
```

### 3. Wrong Import Source

Use `react-router-dom-v5-compat`, not `react-router-dom`:

```typescript
// BAD
import { useNavigate } from 'react-router-dom';

// GOOD
import { useNavigate } from 'react-router-dom-v5-compat';
```

### 4. Not Running Feature Toggle Generation

After adding a feature flag, always run:

```bash
make gen-feature-toggles
```

This updates:
- `pkg/services/featuremgmt/toggles_gen.go`
- Frontend TypeScript types

## Feature Flag Naming Convention

Use pattern: `{feature}UseNavigate`

Examples:
- `playlistUseNavigate`
- `dashboardUseNavigate`
- `exploreUseNavigate`

## Key Files

| File | Purpose |
|------|---------|
| `pkg/services/featuremgmt/registry.go` | Feature flag definitions |
| `packages/grafana-runtime/src/services/LocationService.tsx` | Legacy navigation service |
| `public/app/routes/RoutesWrapper.tsx` | Router setup |
| `docs/react-router-architecture.md` | Full architecture documentation |

## Migration Status Reference

For detailed architecture and migration status of all routing components, see `docs/react-router-architecture.md`.

Current state:
- **~40% complete**: Hooks and components use v6 APIs via compat layer
- **~60% remaining**: Infrastructure, LocationService, and route definitions need migration
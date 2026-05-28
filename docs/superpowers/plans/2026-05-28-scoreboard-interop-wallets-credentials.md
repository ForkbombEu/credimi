# Scoreboard Interop Wallets x Credentials Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `wallets_credentials` mode to the scoreboard interop API/page, keep `wallets_issuers`, and expose a unified row/column metadata contract (`id`, `name`, `subtitle?`, `avatar_url?`, `path`) with backend enrichment for credential issuer subtitle and avatar fallback.

**Architecture:** Refactor interop backend into a mode-configured pipeline that reuses one aggregation core and plugs row/column metadata resolvers per mode. Extend frontend types/components to consume generic entity metadata and add mode selection with default `wallets_credentials`. Keep aggregation semantics and status thresholds unchanged.

**Tech Stack:** Go 1.24, PocketBase handlers/tests (`stretchr/testify`), existing routing/apierror middleware, SvelteKit + TypeScript, Paraglide i18n, Tailwind.

---

## File map

| File | Responsibility |
|------|----------------|
| `pkg/internal/apis/handlers/scoreboard_interop.go` | Add mode registry, unified entity DTO fields, mode-aware cache loading, metadata resolvers for wallets/issuers/credentials |
| `pkg/internal/apis/handlers/scoreboard_interop_test.go` | Unit tests for mode validation, aggregation parity, metadata fallback rules, handler response shape |
| `webapp/src/lib/scoreboard/interop/types.ts` | Extend matrix entity type with `subtitle` and `avatar_url`; include new mode union |
| `webapp/src/lib/scoreboard/interop/matrix-grid.svelte` | Render generic header metadata (name/subtitle/avatar) for rows/columns |
| `webapp/src/routes/(public)/scoreboard/interop/+page.ts` | Read query mode, default to `wallets_credentials`, call API with selected mode |
| `webapp/src/routes/(public)/scoreboard/interop/+page.svelte` | Add mode pills/tabs and bind to query parameter |
| `webapp/messages/en.json` | Add/adjust interop mode labels and empty-state strings if missing |
| `docs/src/content/docs/software-architecture/scoreboard.md` | Update API/page docs for new mode and metadata contract |

---

### Task 1: Backend mode model and unified entity contract

**Files:**
- Modify: `pkg/internal/apis/handlers/scoreboard_interop.go`
- Modify: `pkg/internal/apis/handlers/scoreboard_interop_test.go`

- [ ] **Step 1: Write failing tests for mode parsing and entity JSON fields**

```go
func TestInteropModeValidation(t *testing.T) {
	t.Parallel()
	require.True(t, isSupportedInteropMode(interopModeWalletsIssuers))
	require.True(t, isSupportedInteropMode(interopModeWalletsCredentials))
	require.False(t, isSupportedInteropMode(interopMode("bad_mode")))
}

func TestInteropMatrixEntityJSONShape(t *testing.T) {
	t.Parallel()
	entity := InteropMatrixEntity{
		ID: "rec1",
		Name: "Entity",
		Path: "org/entities/entity",
		Subtitle: ptr.To("subtitle"),
		AvatarURL: ptr.To("https://example.com/avatar.png"),
	}
	raw, err := json.Marshal(entity)
	require.NoError(t, err)
	payload := string(raw)
	require.Contains(t, payload, `"id":"rec1"`)
	require.Contains(t, payload, `"name":"Entity"`)
	require.Contains(t, payload, `"path":"org/entities/entity"`)
	require.Contains(t, payload, `"subtitle":"subtitle"`)
	require.Contains(t, payload, `"avatar_url":"https://example.com/avatar.png"`)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run 'TestInteropModeValidation|TestInteropMatrixEntityJSONShape' -v`  
Expected: FAIL with undefined `interopModeWalletsCredentials`, missing helper, or DTO fields.

- [ ] **Step 3: Implement mode constants, validator, and unified entity fields**

```go
type interopMode string

const (
	interopModeWalletsIssuers     interopMode = "wallets_issuers"
	interopModeWalletsCredentials interopMode = "wallets_credentials"
)

func isSupportedInteropMode(mode interopMode) bool {
	switch mode {
	case interopModeWalletsIssuers, interopModeWalletsCredentials:
		return true
	default:
		return false
	}
}

type InteropMatrixEntity struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Subtitle  *string `json:"subtitle,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	Path      string  `json:"path"`
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run 'TestInteropModeValidation|TestInteropMatrixEntityJSONShape' -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/internal/apis/handlers/scoreboard_interop.go pkg/internal/apis/handlers/scoreboard_interop_test.go
git commit -m "feat(scoreboard): add interop mode enum and unified entity metadata fields"
```

---

### Task 2: Mode registry and generic cache relation loading

**Files:**
- Modify: `pkg/internal/apis/handlers/scoreboard_interop.go`
- Modify: `pkg/internal/apis/handlers/scoreboard_interop_test.go`

- [ ] **Step 1: Write failing tests for mode-specific relation selection**

```go
func TestInteropModeConfigRelations(t *testing.T) {
	t.Parallel()
	cfg, ok := getInteropModeConfig(interopModeWalletsIssuers)
	require.True(t, ok)
	require.Equal(t, "wallets", cfg.rowRelationField)
	require.Equal(t, "issuers", cfg.columnRelationField)

	cfg, ok = getInteropModeConfig(interopModeWalletsCredentials)
	require.True(t, ok)
	require.Equal(t, "wallets", cfg.rowRelationField)
	require.Equal(t, "credentials", cfg.columnRelationField)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run TestInteropModeConfigRelations -v`  
Expected: FAIL with undefined config types/helpers.

- [ ] **Step 3: Implement mode config registry and use it in handler path**

```go
type interopModeConfig struct {
	rowAxis             string
	columnAxis          string
	rowRelationField    string
	columnRelationField string
}

var interopModeConfigs = map[interopMode]interopModeConfig{
	interopModeWalletsIssuers: {
		rowAxis: "wallet", columnAxis: "issuer",
		rowRelationField: "wallets", columnRelationField: "issuers",
	},
	interopModeWalletsCredentials: {
		rowAxis: "wallet", columnAxis: "credential",
		rowRelationField: "wallets", columnRelationField: "credentials",
	},
}

func getInteropModeConfig(mode interopMode) (interopModeConfig, bool) {
	cfg, ok := interopModeConfigs[mode]
	return cfg, ok
}
```

Use this config in `loadInteropMatrixFromCache`:

```go
cfg, ok := getInteropModeConfig(mode)
if !ok {
	return InteropMatrixResponse{}, fmt.Errorf("unsupported mode: %s", mode)
}
rowIDs := rec.GetStringSlice(cfg.rowRelationField)
colIDs := rec.GetStringSlice(cfg.columnRelationField)
```

- [ ] **Step 4: Run tests to verify it passes**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run TestInteropModeConfigRelations -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/internal/apis/handlers/scoreboard_interop.go pkg/internal/apis/handlers/scoreboard_interop_test.go
git commit -m "refactor(scoreboard): add interop mode registry for relation mapping"
```

---

### Task 3: Credential metadata enrichment (issuer subtitle + avatar fallback)

**Files:**
- Modify: `pkg/internal/apis/handlers/scoreboard_interop.go`
- Modify: `pkg/internal/apis/handlers/scoreboard_interop_test.go`

- [ ] **Step 1: Write failing resolver tests for fallback order**

```go
func TestResolveCredentialEntityMetadata_AvatarFallbackOrder(t *testing.T) {
	t.Parallel()
	credentialAvatar := "https://cdn/credential.png"
	issuerAvatar := "https://cdn/issuer.png"
	issuerName := "Issuer A"

	entity := buildCredentialEntityMetadata(
		"cred1", "Credential A", "org/credentials/credential-a",
		ptr.To(credentialAvatar), ptr.To(issuerName), ptr.To(issuerAvatar),
	)
	require.Equal(t, "Credential A", entity.Name)
	require.NotNil(t, entity.Subtitle)
	require.Equal(t, issuerName, *entity.Subtitle)
	require.NotNil(t, entity.AvatarURL)
	require.Equal(t, credentialAvatar, *entity.AvatarURL)

	entity = buildCredentialEntityMetadata(
		"cred2", "Credential B", "org/credentials/credential-b",
		nil, ptr.To(issuerName), ptr.To(issuerAvatar),
	)
	require.NotNil(t, entity.AvatarURL)
	require.Equal(t, issuerAvatar, *entity.AvatarURL)

	entity = buildCredentialEntityMetadata(
		"cred3", "Credential C", "org/credentials/credential-c",
		nil, nil, nil,
	)
	require.Nil(t, entity.AvatarURL)
	require.Nil(t, entity.Subtitle)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run TestResolveCredentialEntityMetadata_AvatarFallbackOrder -v`  
Expected: FAIL with undefined helper.

- [ ] **Step 3: Implement credential metadata builder and resolver**

```go
func buildCredentialEntityMetadata(
	id string,
	name string,
	path string,
	credentialAvatarURL *string,
	issuerName *string,
	issuerAvatarURL *string,
) InteropMatrixEntity {
	avatar := credentialAvatarURL
	if avatar == nil {
		avatar = issuerAvatarURL
	}
	return InteropMatrixEntity{
		ID: id,
		Name: name,
		Subtitle: issuerName,
		AvatarURL: avatar,
		Path: path,
	}
}
```

Integrate in column resolver for `wallets_credentials`:

```go
columnEntities[id] = buildCredentialEntityMetadata(
	credRec.Id,
	credRec.GetString("name"),
	credPath,
	resolveRecordAvatarURL(app, credRec, "avatar"),
	issuerNamePtr,
	issuerAvatarPtr,
)
```

- [ ] **Step 4: Run tests to verify it passes**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run TestResolveCredentialEntityMetadata_AvatarFallbackOrder -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/internal/apis/handlers/scoreboard_interop.go pkg/internal/apis/handlers/scoreboard_interop_test.go
git commit -m "feat(scoreboard): enrich credential columns with issuer subtitle and avatar fallback"
```

---

### Task 4: Handler integration tests for both modes and invalid mode

**Files:**
- Modify: `pkg/internal/apis/handlers/scoreboard_interop_test.go`

- [ ] **Step 1: Write failing HTTP tests for wallets_credentials**

```go
func TestHandleInteropMatrix_WalletsCredentials(t *testing.T) {
	app := setupScoreboardInteropApp(t)
	defer app.Cleanup()

	// Arrange PB fixtures:
	// - wallet record
	// - credential_issuer record
	// - credential record linked to issuer
	// - pipeline_scoreboard_cache row with wallets + credentials + totals

	req := httptest.NewRequest(http.MethodGet, "/api/scoreboard/interop?mode=wallets_credentials", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"mode":"wallets_credentials"`)
	require.Contains(t, rec.Body.String(), `"column_axis":"credential"`)
	require.Contains(t, rec.Body.String(), `"subtitle":"`)
	require.Contains(t, rec.Body.String(), `"avatar_url":"`)
}

func TestHandleInteropMatrix_InvalidMode(t *testing.T) {
	app := setupScoreboardInteropApp(t)
	defer app.Cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/scoreboard/interop?mode=unknown", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "mode")
}
```

- [ ] **Step 2: Run tests to verify failure**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run 'TestHandleInteropMatrix_WalletsCredentials|TestHandleInteropMatrix_InvalidMode' -v`  
Expected: FAIL until fixture mapping and mode branch are complete.

- [ ] **Step 3: Complete handler mode branch and collection lookups**

Use `HandleInteropMatrix` mode guard:

```go
mode := interopMode(e.Request.URL.Query().Get("mode"))
if !isSupportedInteropMode(mode) {
	return apierror.New(
		http.StatusBadRequest,
		"mode",
		"unsupported or missing mode",
		"use mode=wallets_credentials or mode=wallets_issuers",
	).JSON(e)
}
```

Ensure `loadInteropMatrixFromCache(app, mode)` correctly resolves both mode metadata sets and returns `row_axis`/`column_axis` from mode config.

- [ ] **Step 4: Run the full interop handler test subset**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run 'TestHandleInteropMatrix|TestBuildInteropMatrix|TestInteropMode' -v`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/internal/apis/handlers/scoreboard_interop_test.go pkg/internal/apis/handlers/scoreboard_interop.go
git commit -m "test(scoreboard): cover interop wallets_credentials handler and mode validation"
```

---

### Task 5: Frontend types and loader mode default

**Files:**
- Modify: `webapp/src/lib/scoreboard/interop/types.ts`
- Modify: `webapp/src/routes/(public)/scoreboard/interop/+page.ts`

- [ ] **Step 1: Add failing type-level tests (or compile checks) by using new fields in page loader return type**

```ts
export type InteropMode = 'wallets_credentials' | 'wallets_issuers';

export type InteropMatrixEntity = {
	id: string;
	name: string;
	path: string;
	subtitle?: string;
	avatar_url?: string;
};
```

In `+page.ts`, require `mode` to be `InteropMode` and default to `wallets_credentials`.

- [ ] **Step 2: Run check to verify current code fails before edits**

Run: `cd webapp && bun run check`  
Expected: FAIL for missing `InteropMode` or mismatched entity shape in consumers.

- [ ] **Step 3: Implement loader query parsing with new default**

```ts
import type { InteropMode, InteropMatrixResponse } from '$lib/scoreboard/interop/types';

const SUPPORTED_MODES: InteropMode[] = ['wallets_credentials', 'wallets_issuers'];

function normalizeMode(value: string | null): InteropMode {
	return value && SUPPORTED_MODES.includes(value as InteropMode)
		? (value as InteropMode)
		: 'wallets_credentials';
}

export const load = async ({ fetch, url }) => {
	const mode = normalizeMode(url.searchParams.get('mode'));
	const res = await fetch(`/api/scoreboard/interop?mode=${mode}`);
	if (!res.ok) throw new Error(`interop matrix: ${res.status}`);
	const matrix: InteropMatrixResponse = await res.json();
	return { matrix, mode };
};
```

- [ ] **Step 4: Re-run check**

Run: `cd webapp && bun run check`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/scoreboard/interop/types.ts webapp/src/routes/(public)/scoreboard/interop/+page.ts
git commit -m "feat(webapp): add interop mode typing and default wallets_credentials loader"
```

---

### Task 6: Frontend matrix headers and mode selector UI

**Files:**
- Modify: `webapp/src/lib/scoreboard/interop/matrix-grid.svelte`
- Modify: `webapp/src/routes/(public)/scoreboard/interop/+page.svelte`
- Modify: `webapp/messages/en.json`

- [ ] **Step 1: Add failing UI expectations in component tests (if present) or wire compile-time usage of metadata fields**

Use `matrix-grid.svelte` header cell markup requiring:

```svelte
{#if entity.avatar_url}
	<img src={entity.avatar_url} alt="" class="size-5 rounded-full" />
{/if}
<div class="min-w-0">
	<p class="truncate font-medium">{entity.name}</p>
	{#if entity.subtitle}
		<p class="truncate text-xs text-muted-foreground">{entity.subtitle}</p>
	{/if}
</div>
```

Add mode selector in page:

```svelte
<a href={resolve(`/scoreboard/interop?mode=wallets_credentials`)}>{m.interop_mode_wallets_credentials()}</a>
<a href={resolve(`/scoreboard/interop?mode=wallets_issuers`)}>{m.interop_mode_wallets_issuers()}</a>
```

- [ ] **Step 2: Run webapp check to surface missing keys/types**

Run: `cd webapp && bun run check`  
Expected: FAIL for missing i18n keys or prop types.

- [ ] **Step 3: Implement i18n keys**

Add to `webapp/messages/en.json`:

```json
"interop_mode_wallets_credentials": "Wallet x Credential",
"interop_mode_wallets_issuers": "Wallet x Issuer"
```

Keep existing matrix strings unchanged unless required by component refactor.

- [ ] **Step 4: Re-run check and lint**

Run: `cd webapp && bun run check && bun run lint`  
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/scoreboard/interop/matrix-grid.svelte webapp/src/routes/(public)/scoreboard/interop/+page.svelte webapp/messages/en.json
git commit -m "feat(webapp): render generic interop metadata and add mode selector"
```

---

### Task 7: Verification and documentation update

**Files:**
- Modify: `docs/src/content/docs/software-architecture/scoreboard.md`

- [ ] **Step 1: Run focused Go tests**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run Interop -v`  
Expected: PASS.

- [ ] **Step 2: Run frontend verification**

Run: `cd webapp && bun run check && bun run lint`  
Expected: PASS.

- [ ] **Step 3: Update architecture docs**

Add a concise section documenting:

- `/scoreboard/interop` supports `mode=wallets_credentials|wallets_issuers`
- default UI mode is `wallets_credentials`
- row/column metadata contract (`id`, `name`, `subtitle?`, `avatar_url?`, `path`)
- `wallets_credentials` subtitle/avatar behavior

- [ ] **Step 4: Commit docs + final verification evidence**

```bash
git add docs/src/content/docs/software-architecture/scoreboard.md
git commit -m "docs(scoreboard): document wallets_credentials interop mode and metadata contract"
```

---

## Spec coverage checklist

| Spec requirement | Plan task |
|------------------|-----------|
| Add `wallets_credentials` mode on same API/page | Tasks 1, 2, 4, 5, 6 |
| Keep `wallets_issuers` mode | Tasks 2, 4, 6 |
| Unified metadata interface (`id`, `name`, `subtitle?`, `avatar_url?`, `path`) | Tasks 1, 5, 6 |
| Credential subtitle = issuer name | Task 3 |
| Avatar fallback credential -> issuer -> empty | Task 3 |
| Cartesian + run-weighted semantics unchanged | Tasks 2, 4 |
| Default UI mode `wallets_credentials` | Task 5 |
| Error handling for invalid mode | Task 4 |
| Testing and docs update | Task 7 |

---

## Self-review

1. **Spec coverage:** Every approved design requirement is mapped in the checklist above; no missing feature slice.
2. **Placeholder scan:** No TODO/TBD markers; each task has concrete files, commands, and expected outcomes.
3. **Type consistency:** Mode names and metadata keys are consistent across backend, frontend, and docs (`wallets_credentials`, `wallets_issuers`, `subtitle`, `avatar_url`).

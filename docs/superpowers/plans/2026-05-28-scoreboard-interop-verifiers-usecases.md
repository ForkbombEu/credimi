# Scoreboard Interop Verifiers & Use Case Verifications Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `wallets_verifiers` and `wallets_use_case_verifications` modes to the scoreboard interop API/page, reusing the existing mode registry, aggregation pipeline, entity metadata enrichment, and frontend rendering.

**Architecture:** Two new entries in `interopModeConfigs` (no structural changes), one new `use_cases_verifications` branch in `interopEntityFromRecord` mirroring the existing `credentials` pattern, a one-line `AvatarURL` addition to the generic entity fallback, and a rename of `buildCredentialEntityMetadata` → `buildEnrichedEntityMetadata`. Frontend: extend the `InteropMode` union and `SUPPORTED_MODES` array, add 2 i18n keys.

**Tech Stack:** Go 1.24, PocketBase handlers/tests (`stretchr/testify`), SvelteKit + TypeScript, Paraglide i18n.

**Spec:** `docs/superpowers/specs/2026-05-28-scoreboard-interop-verifiers-usecases-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `pkg/internal/apis/handlers/scoreboard_interop.go` | Add 2 mode constants + configs, 1 new collection branch, rename function, AvatarURL on fallback |
| `pkg/internal/apis/handlers/scoreboard_interop_test.go` | Extend mode validation/config tests, add use_case metadata resolver tests, add verifier AvatarURL test |
| `webapp/src/lib/scoreboard/interop/types.ts` | Extend `InteropMode` union and `SUPPORTED_MODES` array |
| `webapp/src/routes/(public)/scoreboard/interop/+page.ts` | Extend `SUPPORTED_MODES` array |
| `webapp/messages/en.json` | Add 2 i18n keys for new mode labels |

---

### Task 1: Backend mode constants and config registry

**Files:**
- Modify: `pkg/internal/apis/handlers/scoreboard_interop.go`
- Modify: `pkg/internal/apis/handlers/scoreboard_interop_test.go`

- [ ] **Step 1: Write failing tests for new mode constants and configs**

```go
func TestInteropModeValidation(t *testing.T) {
	t.Parallel()
	require.True(t, isSupportedInteropMode(interopModeWalletsIssuers))
	require.True(t, isSupportedInteropMode(interopModeWalletsCredentials))
	require.True(t, isSupportedInteropMode(interopModeWalletsVerifiers))
	require.True(t, isSupportedInteropMode("wallets_verifiers"))
	require.True(t, isSupportedInteropMode(interopModeWalletsUseCaseVerifications))
	require.True(t, isSupportedInteropMode("wallets_use_case_verifications"))
	require.False(t, isSupportedInteropMode(interopMode("")))
	require.False(t, isSupportedInteropMode(interopMode("bad_mode")))
}

func TestInteropModeConfigRelations(t *testing.T) {
	t.Parallel()
	cfg, ok := getInteropModeConfig(interopModeWalletsIssuers)
	require.True(t, ok)
	require.Equal(t, "wallets", cfg.RowRelationField)
	require.Equal(t, "issuers", cfg.ColumnRelationField)

	cfg, ok = getInteropModeConfig(interopModeWalletsCredentials)
	require.True(t, ok)
	require.Equal(t, "wallets", cfg.RowRelationField)
	require.Equal(t, "credentials", cfg.ColumnRelationField)

	cfg, ok = getInteropModeConfig(interopModeWalletsVerifiers)
	require.True(t, ok)
	require.Equal(t, "wallets", cfg.RowRelationField)
	require.Equal(t, "verifiers", cfg.ColumnRelationField)
	require.Equal(t, "verifier", cfg.ColumnAxis)
	require.Equal(t, "verifiers", cfg.ColumnCollection)

	cfg, ok = getInteropModeConfig(interopModeWalletsUseCaseVerifications)
	require.True(t, ok)
	require.Equal(t, "wallets", cfg.RowRelationField)
	require.Equal(t, "use_case_verifications", cfg.ColumnRelationField)
	require.Equal(t, "use_case_verification", cfg.ColumnAxis)
	require.Equal(t, "use_cases_verifications", cfg.ColumnCollection)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run 'TestInteropModeValidation|TestInteropModeConfigRelations' -v`
Expected: FAIL with undefined `interopModeWalletsVerifiers`, `interopModeWalletsUseCaseVerifications`.

- [ ] **Step 3: Add mode constants and config entries**

```go
const (
	interopModeWalletsIssuers              interopMode = "wallets_issuers"
	interopModeWalletsCredentials          interopMode = "wallets_credentials"
	interopModeWalletsVerifiers            interopMode = "wallets_verifiers"
	interopModeWalletsUseCaseVerifications interopMode = "wallets_use_case_verifications"
)

var interopModeConfigs = map[interopMode]interopModeConfig{
	interopModeWalletsIssuers: {
		RowRelationField:    "wallets",
		ColumnRelationField: "issuers",
		RowAxis:             "wallet",
		ColumnAxis:          "issuer",
		RowCollection:       "wallets",
		ColumnCollection:    "credential_issuers",
	},
	interopModeWalletsCredentials: {
		RowRelationField:    "wallets",
		ColumnRelationField: "credentials",
		RowAxis:             "wallet",
		ColumnAxis:          "credential",
		RowCollection:       "wallets",
		ColumnCollection:    "credentials",
	},
	interopModeWalletsVerifiers: {
		RowRelationField:    "wallets",
		ColumnRelationField: "verifiers",
		RowAxis:             "wallet",
		ColumnAxis:          "verifier",
		RowCollection:       "wallets",
		ColumnCollection:    "verifiers",
	},
	interopModeWalletsUseCaseVerifications: {
		RowRelationField:    "wallets",
		ColumnRelationField: "use_case_verifications",
		RowAxis:             "wallet",
		ColumnAxis:          "use_case_verification",
		RowCollection:       "wallets",
		ColumnCollection:    "use_cases_verifications",
	},
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run 'TestInteropModeValidation|TestInteropModeConfigRelations' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/internal/apis/handlers/scoreboard_interop.go pkg/internal/apis/handlers/scoreboard_interop_test.go
git commit -m "feat(scoreboard): add wallets_verifiers and wallets_use_case_verifications mode configs"
```

---

### Task 2: `use_cases_verifications` metadata branch and AvatarURL fallback

**Files:**
- Modify: `pkg/internal/apis/handlers/scoreboard_interop.go`
- Modify: `pkg/internal/apis/handlers/scoreboard_interop_test.go`

- [ ] **Step 1: Write failing tests for use_case metadata and verifier AvatarURL**

Add these new test functions (keep existing tests):

```go
func TestBuildEnrichedEntityMetadata_UseCaseVerification(t *testing.T) {
	t.Parallel()

	useCaseLogo := "https://cdn/usecase-logo.png"
	verifierLogo := "https://cdn/verifier-logo.png"
	verifierName := "Verifier A"

	entity := buildEnrichedEntityMetadata(
		"uc1", "PID Verification", "org/v/p",
		ptr.To(useCaseLogo), ptr.To(verifierName), ptr.To(verifierLogo),
	)
	require.Equal(t, "PID Verification", entity.Name)
	require.NotNil(t, entity.Subtitle)
	require.Equal(t, verifierName, *entity.Subtitle)
	require.NotNil(t, entity.AvatarURL)
	require.Equal(t, useCaseLogo, *entity.AvatarURL)

	entity = buildEnrichedEntityMetadata(
		"uc2", "PID Verification", "org/v/p",
		nil, ptr.To(verifierName), ptr.To(verifierLogo),
	)
	require.NotNil(t, entity.AvatarURL)
	require.Equal(t, verifierLogo, *entity.AvatarURL)

	entity = buildEnrichedEntityMetadata(
		"uc3", "PID Verification", "org/v/p",
		nil, nil, nil,
	)
	require.Nil(t, entity.AvatarURL)
	require.Nil(t, entity.Subtitle)
}

func TestInteropEntityFromRecord_VerifierAvatarURL(t *testing.T) {
	t.Parallel()
	// Test that the generic fallback populates AvatarURL from logo field.
	// This test exercises the fallback path when collection is "verifiers"
	// by verifying the InteropMatrixEntity includes AvatarURL.
	app := setupScoreboardInteropApp(t)
	defer app.Cleanup()

	verifierColl, err := app.FindCollectionByNameOrId("verifiers")
	require.NoError(t, err)

	rec := core.NewRecord(verifierColl)
	rec.Set("name", "Test Verifier")
	rec.Set("canonified_name", "test-verifier")
	rec.Set("logo", "logo_test.png")
	require.NoError(t, app.Save(rec))

	entity, err := interopEntityFromRecord(app, rec, "verifiers")
	require.NoError(t, err)
	require.Equal(t, "Test Verifier", entity.Name)
	require.NotNil(t, entity.AvatarURL)
	require.Contains(t, *entity.AvatarURL, "logo_test")
	require.Nil(t, entity.Subtitle)
}

func TestBuildEnrichedEntityMetadata_CredentialAvatarFallback(t *testing.T) {
	t.Parallel()

	credentialAvatar := "https://cdn/credential.png"
	issuerAvatar := "https://cdn/issuer.png"
	issuerName := "Issuer A"

	entity := buildEnrichedEntityMetadata(
		"cred1", "Credential A", "org/credentials/credential-a",
		ptr.To(credentialAvatar), ptr.To(issuerName), ptr.To(issuerAvatar),
	)
	require.Equal(t, "Credential A", entity.Name)
	require.NotNil(t, entity.Subtitle)
	require.Equal(t, issuerName, *entity.Subtitle)
	require.NotNil(t, entity.AvatarURL)
	require.Equal(t, credentialAvatar, *entity.AvatarURL)

	entity = buildEnrichedEntityMetadata(
		"cred2", "Credential B", "org/credentials/credential-b",
		nil, ptr.To(issuerName), ptr.To(issuerAvatar),
	)
	require.NotNil(t, entity.AvatarURL)
	require.Equal(t, issuerAvatar, *entity.AvatarURL)

	entity = buildEnrichedEntityMetadata(
		"cred3", "Credential C", "org/credentials/credential-c",
		nil, nil, nil,
	)
	require.Nil(t, entity.AvatarURL)
	require.Nil(t, entity.Subtitle)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run 'TestBuildEnrichedEntityMetadata|TestInteropEntityFromRecord_VerifierAvatarURL' -v`
Expected: FAIL with undefined `buildEnrichedEntityMetadata`.

- [ ] **Step 3: Rename `buildCredentialEntityMetadata` → `buildEnrichedEntityMetadata`**

In `pkg/internal/apis/handlers/scoreboard_interop.go`, rename the function and update its call site in `interopEntityFromRecord`:

```go
// Before (line ~393):
return buildCredentialEntityMetadata(
    record.Id,
    record.GetString("name"),
    path,
    firstNonEmptyStringPtr(record.GetString("avatar"), record.GetString("logo")),
    issuerName,
    issuerAvatarURL,
), nil

// After:
return buildEnrichedEntityMetadata(
    record.Id,
    record.GetString("name"),
    path,
    firstNonEmptyStringPtr(record.GetString("avatar"), record.GetString("logo")),
    issuerName,
    issuerAvatarURL,
), nil
```

Function definition rename:

```go
// Before:
func buildCredentialEntityMetadata(

// After:
func buildEnrichedEntityMetadata(
```

- [ ] **Step 4: Add `use_cases_verifications` branch in `interopEntityFromRecord`**

Add a new `if collection == "use_cases_verifications"` block above the existing `if collection == "credentials"` block:

```go
if collection == "use_cases_verifications" {
	var verifierName *string
	var verifierLogoURL *string
	verifierID := strings.TrimSpace(record.GetString("verifier"))
	if verifierID != "" {
		verifierRecord, err := app.FindRecordById("verifiers", verifierID)
		if err == nil {
			verifierName = optionalTrimmedStringPtr(verifierRecord.GetString("name"))
			verifierLogoURL = firstNonEmptyStringPtr(
				verifierRecord.GetString("avatar"),
				verifierRecord.GetString("logo"),
			)
		}
	}
	return buildEnrichedEntityMetadata(
		record.Id,
		record.GetString("name"),
		path,
		firstNonEmptyStringPtr(record.GetString("avatar"), record.GetString("logo")),
		verifierName,
		verifierLogoURL,
	), nil
}
```

- [ ] **Step 5: Add `AvatarURL` to the generic fallback**

In `interopEntityFromRecord`, update the final return statement to include `AvatarURL`:

```go
// Before:
return InteropMatrixEntity{
    ID:   record.Id,
    Name: record.GetString("name"),
    Path: path,
}, nil

// After:
return InteropMatrixEntity{
    ID:        record.Id,
    Name:      record.GetString("name"),
    Path:      path,
    AvatarURL: firstNonEmptyStringPtr(record.GetString("avatar"), record.GetString("logo")),
}, nil
```

- [ ] **Step 6: Run all interop tests**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run 'TestBuildEnrichedEntityMetadata|TestInteropEntityFromRecord_VerifierAvatarURL|TestInteropModeValidation|TestInteropModeConfigRelations|TestBuildInteropMatrix|TestInteropStatus' -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add pkg/internal/apis/handlers/scoreboard_interop.go pkg/internal/apis/handlers/scoreboard_interop_test.go
git commit -m "feat(scoreboard): add use_cases_verifications metadata enrichment and verifier avatar fallback"
```

---

### Task 3: Frontend types, mode selector, and i18n

**Files:**
- Modify: `webapp/src/lib/scoreboard/interop/types.ts`
- Modify: `webapp/src/routes/(public)/scoreboard/interop/+page.ts`
- Modify: `webapp/messages/en.json`

- [ ] **Step 1: Extend `InteropMode` type and `SUPPORTED_MODES` array**

In `webapp/src/lib/scoreboard/interop/types.ts`:

```ts
export type InteropMode =
	| 'wallets_credentials'
	| 'wallets_issuers'
	| 'wallets_verifiers'
	| 'wallets_use_case_verifications';
```

In `webapp/src/routes/(public)/scoreboard/interop/+page.ts`, extend the `SUPPORTED_MODES` array if it exists, or add it:

```ts
const SUPPORTED_MODES: InteropMode[] = [
	'wallets_credentials',
	'wallets_issuers',
	'wallets_verifiers',
	'wallets_use_case_verifications'
];
```

- [ ] **Step 2: Verify typecheck fails before i18n keys exist**

Run: `cd webapp && bun run check`
(If mode selector references the i18n keys already, this may fail. Proceed to Step 3 regardless.)

- [ ] **Step 3: Add i18n keys**

In `webapp/messages/en.json`, add:

```json
"interop_mode_wallets_verifiers": "Wallet × Verifier",
"interop_mode_wallets_use_case_verifications": "Wallet × Use case verification"
```

Run paraglide generation if needed: `cd webapp && bun run paraglide` (or whatever `package.json` documents).

- [ ] **Step 4: Re-run webapp check and lint**

Run: `cd webapp && bun run check && bun run lint`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/scoreboard/interop/types.ts webapp/src/routes/(public)/scoreboard/interop/+page.ts webapp/messages/en.json
git commit -m "feat(webapp): add interop mode labels for verifiers and use case verifications"
```

---

### Task 4: Verification and handler integration test

**Files:**
- Modify: `pkg/internal/apis/handlers/scoreboard_interop_test.go`
- Modify: `docs/src/content/docs/software-architecture/scoreboard.md`

- [ ] **Step 1: Extend handler test for new modes**

Extend `TestHandleInteropMatrix_InvalidMode` (or add a new test) to verify the error message enumerates all 4 modes:

```go
func TestHandleInteropMatrix_AllSupportedModes(t *testing.T) {
	app := setupScoreboardInteropApp(t)
	defer app.Cleanup()

	for _, mode := range []string{
		"wallets_credentials",
		"wallets_issuers",
		"wallets_verifiers",
		"wallets_use_case_verifications",
	} {
		t.Run(mode, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/scoreboard/interop?mode="+mode, nil)
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code)
		})
	}
}
```

- [ ] **Step 2: Run to verify all modes return 200**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run TestHandleInteropMatrix_AllSupportedModes -v`
Expected: PASS (all 4 sub-tests).

- [ ] **Step 3: Run full interop test suite**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run Interop -v`
Expected: PASS on all interop tests.

- [ ] **Step 4: Run full webapp verification**

Run: `cd webapp && bun run check && bun run lint`
Expected: PASS.

- [ ] **Step 5: Update architecture docs**

In `docs/src/content/docs/software-architecture/scoreboard.md`, update the mode listing to include all 4 modes:

```markdown
- `/scoreboard/interop` supports `mode=wallets_credentials|wallets_issuers|wallets_verifiers|wallets_use_case_verifications`
- Default UI mode is `wallets_credentials`
```

- [ ] **Step 6: Commit**

```bash
git add pkg/internal/apis/handlers/scoreboard_interop_test.go docs/src/content/docs/software-architecture/scoreboard.md
git commit -m "test(scoreboard): verify all 4 interop modes return 200 and docs update"
```

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| `wallets_verifiers` mode config entry | Task 1 |
| `wallets_use_case_verifications` mode config entry | Task 1 |
| Verifier `avatar_url` from logo, no subtitle | Task 2 (generic fallback + test) |
| Use case `subtitle` = verifier name | Task 2 |
| Use case `avatar_url`: use_case logo → verifier logo → nil | Task 2 |
| Rename `buildCredentialEntityMetadata` → `buildEnrichedEntityMetadata` | Task 2 |
| Credential metadata fallback still works after rename | Task 2 (`TestBuildEnrichedEntityMetadata_CredentialAvatarFallback`) |
| Default mode stays `wallets_credentials` | Task 3 (unchanged) |
| Mode selector shows all 4 options | Task 3 |
| i18n keys for new modes | Task 3 |
| All 4 modes return 200 | Task 4 |
| Invalid mode → 400 | Task 1 (already covered, unchanged) |
| Docs update | Task 4 |

---

## Self-review

1. **Spec coverage:** Every approved spec requirement is mapped above. No missing feature slice.
2. **Placeholder scan:** No TODO/TBD markers. All test code, implementation code, and commands are concrete and complete.
3. **Type consistency:** Mode names (`wallets_verifiers`, `wallets_use_case_verifications`) are consistent across backend config, frontend types, i18n keys. Renamed function `buildEnrichedEntityMetadata` is used in both the `use_cases_verifications` branch and the existing `credentials` branch (call site updated). Test function names and property assertions match the implementation.

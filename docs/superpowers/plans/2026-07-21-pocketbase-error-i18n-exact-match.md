# PocketBase Error i18n Exact Match Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the hardcoded PocketBase validation-code mapper with exact-key lookup against Paraglide `m`, using backend `validation_*` codes as message keys.

**Architecture:** Add `localizePocketBaseErrorCode` in `webapp/src/modules/utils/errors.ts`. It looks up `code` on an optional message catalog (default `m`); if the value is a function that returns a string when called with no args, use that; otherwise return the backend fallback. Rename the wallet i18n key to match the Go code. Wire collection forms to the shared helper.

**Tech Stack:** TypeScript, Paraglide (`m`), Vitest, PocketBase `ClientResponseError` field `{ code, message }`.

**Design spec:** `docs/superpowers/specs/2026-07-21-pocketbase-error-i18n-exact-match-design.md`

## Global Constraints

- Exact string match only — no strip/capitalize transforms.
- Backend `validation_*` codes are canonical; rename i18n, not Go.
- Missing key → backend `message` fallback.
- Do not add i18n for other custom codes in this pass.
- Do not commit unless the user explicitly asks.

---

## File map

| File | Responsibility |
|------|----------------|
| `webapp/src/modules/utils/errors.ts` | `localizePocketBaseErrorCode` |
| `webapp/src/modules/utils/errors.test.ts` | Unit tests for the helper |
| `webapp/messages/{en,it,de,fr,da,pt-BR,es-es}.json` | Rename wallet validation message key |
| `webapp/src/modules/collections-components/form/collectionFormSetup.ts` | Use shared helper; remove hardcoded branch |

---

### Task 1: Helper + unit tests (TDD)

**Files:**
- Modify: `webapp/src/modules/utils/errors.ts`
- Modify: `webapp/src/modules/utils/errors.test.ts`

**Interfaces:**
- Produces: `localizePocketBaseErrorCode(code: string | undefined, fallback: string, catalog?: Record<string, unknown>): string`
- `catalog` defaults to `m as unknown as Record<string, unknown>` so tests can inject stubs without mocking the whole i18n module.

- [x] **Step 1: Write failing tests**

Append to `errors.test.ts`:

```ts
import { localizePocketBaseErrorCode } from './errors';

describe('localizePocketBaseErrorCode', () => {
	const catalog = {
		validation_wallet_action_market_link_requires_install_app: () => 'Localized wallet error',
		validation_not_a_function: 'oops',
		validation_throws: () => {
			throw new Error('needs inputs');
		},
		validation_non_string: () => 42
	} as Record<string, unknown>;

	it('returns localized string when code matches a message function', () => {
		expect(
			localizePocketBaseErrorCode(
				'validation_wallet_action_market_link_requires_install_app',
				'fallback',
				catalog
			)
		).toBe('Localized wallet error');
	});

	it('returns fallback for unknown codes', () => {
		expect(localizePocketBaseErrorCode('validation_unknown', 'fallback', catalog)).toBe(
			'fallback'
		);
	});

	it('returns fallback when code is undefined', () => {
		expect(localizePocketBaseErrorCode(undefined, 'fallback', catalog)).toBe('fallback');
	});

	it('returns fallback when catalog value is not a function', () => {
		expect(
			localizePocketBaseErrorCode('validation_not_a_function', 'fallback', catalog)
		).toBe('fallback');
	});

	it('returns fallback when message function throws', () => {
		expect(localizePocketBaseErrorCode('validation_throws', 'fallback', catalog)).toBe(
			'fallback'
		);
	});

	it('returns fallback when message function returns a non-string', () => {
		expect(localizePocketBaseErrorCode('validation_non_string', 'fallback', catalog)).toBe(
			'fallback'
		);
	});
});
```

- [x] **Step 2: Run tests — expect fail**

```sh
cd webapp && bun run test:unit -- --run src/modules/utils/errors.test.ts
```

Expected: FAIL — `localizePocketBaseErrorCode` is not exported.

- [x] **Step 3: Implement helper**

In `errors.ts`:

```ts
import { m } from '@/i18n';

export function localizePocketBaseErrorCode(
	code: string | undefined,
	fallback: string,
	catalog: Record<string, unknown> = m as unknown as Record<string, unknown>
): string {
	if (!code) return fallback;

	const candidate = catalog[code];
	if (typeof candidate !== 'function') return fallback;

	try {
		const localized = (candidate as () => unknown)();
		return typeof localized === 'string' ? localized : fallback;
	} catch {
		return fallback;
	}
}
```

Keep existing `getExceptionMessage` / `exceptionToError` unchanged.

- [x] **Step 4: Run tests — expect pass**

```sh
cd webapp && bun run test:unit -- --run src/modules/utils/errors.test.ts
```

Expected: PASS.

---

### Task 2: Rename i18n key in all locale files

**Files:**
- Modify: `webapp/messages/en.json`
- Modify: `webapp/messages/it.json`
- Modify: `webapp/messages/de.json`
- Modify: `webapp/messages/fr.json`
- Modify: `webapp/messages/da.json`
- Modify: `webapp/messages/pt-BR.json`
- Modify: `webapp/messages/es-es.json`

**Interfaces:**
- Produces: message key `validation_wallet_action_market_link_requires_install_app` (exact backend code)

- [x] **Step 1: Rename key in every locale**

Replace the JSON key only (keep each locale’s string value):

- from: `"Wallet_action_market_link_requires_install_app"`
- to: `"validation_wallet_action_market_link_requires_install_app"`

- [x] **Step 2: Verify no old key remains**

```sh
rg 'Wallet_action_market_link_requires_install_app' webapp/messages webapp/src
```

Expected: no matches (except possibly this plan/spec docs).

---

### Task 3: Wire collection form to shared helper

**Files:**
- Modify: `webapp/src/modules/collections-components/form/collectionFormSetup.ts`

**Interfaces:**
- Consumes: `localizePocketBaseErrorCode` from `@/utils/errors`

- [x] **Step 1: Update imports**

- Remove `m` import if it becomes unused after this change (it is still used for success toasts — keep it).
- Add: `import { getExceptionMessage, localizePocketBaseErrorCode } from '@/utils/errors';`
- Remove the separate `getExceptionMessage` import path if duplicated; keep a single import from `@/utils/errors`.

Current imports use `getExceptionMessage` from `@/utils/errors` — extend that import.

- [x] **Step 2: Replace local helper**

Delete:

```ts
function localizePocketBaseError(code: string | undefined, fallback: string): string {
	if (code === 'validation_wallet_action_market_link_requires_install_app') {
		return m.Wallet_action_market_link_requires_install_app();
	}

	return fallback;
}
```

In the `ClientResponseError` catch block, replace both call sites:

- `localizePocketBaseError(...)` → `localizePocketBaseErrorCode(...)`

- [x] **Step 3: Typecheck / unit smoke**

```sh
cd webapp && bun run check
cd webapp && bun run test:unit -- --run src/modules/utils/errors.test.ts
```

Expected: check succeeds (or only pre-existing unrelated errors); tests PASS.

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| Exact-match lookup helper | Task 1 |
| Backend message fallback | Task 1 |
| Rename i18n key to backend code | Task 2 |
| Wire collection form; remove if-branch | Task 3 |
| Unit tests hit/miss/undefined/non-callable | Task 1 |
| Out of scope: other validation codes | — (not implemented) |
| Out of scope: Go renames | — (not implemented) |

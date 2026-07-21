# PocketBase Validation Error i18n — Exact Match Design Spec

**Date:** 2026-07-21  
**Status:** Approved (design interview)  
**Scope:** Replace the hardcoded PocketBase validation-code → `m` mapping in collection forms with exact-key lookup against Paraglide messages, aligned to backend `validation_*` codes.

---

## Summary

The last PR added `localizePocketBaseError` with a single hardcoded branch:

- backend code: `validation_wallet_action_market_link_requires_install_app`
- i18n key: `Wallet_action_market_link_requires_install_app`

That mapping does not scale and invents a second naming scheme.

**Change:**

1. **Canonical key** — backend PocketBase/ozzo validation `code` is the i18n message key (exact string match).
2. **Rename** — change the existing wallet message key in all `webapp/messages/*.json` to `validation_wallet_action_market_link_requires_install_app`.
3. **Lookup** — shared helper: if `code` exists on `m` and is a zero-arg (no required inputs) message function, call it; otherwise return the backend `message` fallback.
4. **Wire** — collection form error handling uses the helper instead of the if-branch.

**Out of scope:** Localizing other custom codes (`validation_pipeline_published_locked`, `validation_organization_public_entities`); renaming Go constants; transforming or stripping `validation_`; changing PocketBase built-in error UX beyond opt-in localization when a matching key exists.

---

## Problem

| Aspect | Current | Desired |
|--------|---------|---------|
| Key relationship | Hand-mapped, names differ | Exact match |
| Adding a new localized validation | Edit helper + messages | Add matching message key only |
| Missing translation | N/A (hardcoded) | Show backend `message` |
| Collision with UI labels (e.g. `Required`) | Transform would collide | Exact `validation_*` keys avoid UI collisions |

---

## Decisions

| Decision | Choice |
|----------|--------|
| Match strategy | Exact string equality between PB `code` and `m` key |
| Canonical owner of the name | Backend `validation_*` code |
| Rename direction | Rename i18n key to match backend (no Go rename) |
| Missing key fallback | Backend `message` / `e.message` as today |
| Implementation shape | Shared helper + unit tests (not an inline-only change) |
| Other custom codes this pass | Do not add i18n keys yet |

---

## Contract

### Backend → frontend

PocketBase `ClientResponseError` field errors expose `{ code, message }`.

- `code`: stable machine id (e.g. `validation_wallet_action_market_link_requires_install_app`)
- `message`: human or code-echo fallback from the server

### Localization rule

```text
if code is defined
  and code is a key of m
  and m[code] is a function callable with no required inputs
then return m[code]()
else return fallback (backend message)
```

### Adding a new localized validation error (future)

1. Choose a stable `validation_…` code in Go (`validation.NewError(code, …)` / bad-request payload).
2. Add the **same** string as a key in `webapp/messages/*.json`.
3. No frontend mapper update required.

---

## Design

### Helper

Add `localizePocketBaseErrorCode` to `webapp/src/modules/utils/errors.ts` (alongside `getExceptionMessage`, which collection forms already use). Extend `errors.test.ts`.

Responsibilities:

- Accept `(code: string | undefined, fallback: string): string`
- Perform exact-key lookup on `m`
- Do not strip prefixes, capitalize, or otherwise transform `code`
- Do not throw on unknown codes; always return a string

Safety:

- Only invoke message functions that need no required inputs (current custom validation messages are parameterless).
- If a key exists but is not a safe zero-arg message function, return `fallback`.

### Collection form

`collectionFormSetup.ts` keeps calling the helper for:

- per-field `data.code` / `data.message`
- top-level form error using first field `code` and `e.message`

Remove the wallet-specific if-branch.

### Messages

Rename in every locale file under `webapp/messages/`:

- from: `Wallet_action_market_link_requires_install_app`
- to: `validation_wallet_action_market_link_requires_install_app`

Copy stays unchanged per locale. Paraglide regenerates `m` via the existing Vite plugin on next build/dev.

### Tests

Unit-test the helper:

- known code present on a stub/`m` → localized string
- unknown code → fallback
- `undefined` code → fallback
- key present but not a callable message → fallback

No requirement for an E2E form test in this pass.

---

## Non-goals

- Do not localize `validation_pipeline_published_locked` or `validation_organization_public_entities` unless a follow-up explicitly adds matching message keys.
- Do not change Go error codes or English server messages for this feature.
- Do not introduce a transform layer (`strip validation_`, PascalCase, etc.).
- Do not attempt to localize arbitrary PocketBase built-in codes unless matching keys are deliberately added to `messages/*.json`.

---

## Risks

| Risk | Mitigation |
|------|------------|
| Stale Paraglide output until vite regenerates | Rename source JSON; regeneration is existing project workflow |
| Accidental collision if someone adds a UI string equal to a `validation_*` code | Unlikely; `validation_` prefix is the namespace; exact-match is intentional |
| Parameterized Paraglide messages later | Helper only calls zero-required-input functions; otherwise fallback |

---

## Acceptance

- Hardcoded wallet if-branch is gone.
- Wallet validation code localizes when the matching `m` key exists.
- Unknown codes still show the backend message.
- All locale files use the renamed key.
- Helper has unit coverage for hit / miss / undefined / non-callable.

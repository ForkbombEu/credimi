# OpenID4VCI Wallet Action Picker Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** When adding an `openid4vci_wallet*` conformance check and the execution-target wallet has 2+ `get-credential-generic` actions, show a funnel step to pick which action binds to `action_id`; skip the step when exactly one action exists.

**Architecture:** Extend `ConformanceCheckStepForm` with `select-wallet-action` state between test selection and `ready`. Reuse the existing `walletActions` resource (already filtered by category). Extract a small pure helper for action-count branching so selection logic is unit-testable. Template adds breadcrumb + action list using the same card markup as `wallet-action-step-form.svelte`.

**Tech Stack:** Svelte 5 runes, Runed `resource()`, Vitest, Paraglide (`m.*`), PocketBase `wallet_actions`.

**Design spec:** `docs/superpowers/specs/2026-06-06-openid4vci-wallet-action-picker-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `webapp/src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.svelte.ts` | State machine, helpers, `selectTest` / `selectWalletAction` / discard methods, derived lists |
| `webapp/src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.svelte` | Breadcrumb row, `select-wallet-action` picker UI |
| `webapp/src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.test.ts` | Unit tests for pure selection helper |
| `webapp/src/lib/pipeline-form/steps/conformance-check/index.ts` | **No changes** — `serialize()` safety net unchanged |

---

### Task 1: Pure selection helper + unit tests

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.svelte.ts`
- Create: `webapp/src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.test.ts`

- [ ] **Step 1: Write the failing test file**

Create `conformance-check-step-form.test.ts`:

```ts
// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest';

vi.mock('./conformance-check-step-form.svelte', () => ({ default: class {} }));

import {
	resolveWalletActionSelection,
	type WalletActionSelection
} from './conformance-check-step-form.svelte.js';

const action = (id: string) =>
	({ id, name: id, category: 'get-credential-generic' }) as never;

describe('resolveWalletActionSelection', () => {
	it('returns blocked when no actions', () => {
		expect(resolveWalletActionSelection([])).toEqual({ kind: 'blocked' });
	});

	it('returns auto with single action', () => {
		const a = action('a1');
		expect(resolveWalletActionSelection([a])).toEqual({
			kind: 'auto',
			action: a
		} satisfies WalletActionSelection);
	});

	it('returns picker when multiple actions', () => {
		expect(resolveWalletActionSelection([action('a1'), action('a2')])).toEqual({
			kind: 'picker'
		});
	});
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.test.ts --run`

Expected: FAIL — `resolveWalletActionSelection` is not exported

- [ ] **Step 3: Add exported helper at bottom of `conformance-check-step-form.svelte.ts`**

Add after `getWalletTestBlockReason` (keep existing function; update block check to use actions length):

```ts
export type WalletActionSelection =
	| { kind: 'blocked' }
	| { kind: 'auto'; action: WalletActionsResponse }
	| { kind: 'picker' };

export function isOpenId4VciWalletTest(test: string) {
	return test.startsWith('openid4vci_wallet');
}

export function resolveWalletActionSelection(
	actions: WalletActionsResponse[]
): WalletActionSelection {
	if (actions.length === 0) return { kind: 'blocked' };
	if (actions.length === 1) return { kind: 'auto', action: actions[0]! };
	return { kind: 'picker' };
}
```

Update `getWalletTestBlockReason` — replace `.find(...)` missing-action check with length check:

```ts
	const actions = walletActions.current ?? [];

	if (actions.length === 0) {
		return m.Pipeline_form_wallet_missing_action_category({
			wallet: wallet.name,
			category: OPENID4VCI_WALLET_ACTION_CATEGORY
		});
	}

	return null;
```

Remove the old single `.find` before the missing-action branch (the resource query already filters by category).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.test.ts --run`

Expected: PASS (3 tests)

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.svelte.ts \
  webapp/src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.test.ts
git commit -m "test(conformance-check): add wallet action selection helper"
```

---

### Task 2: Form state machine and selection methods

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.svelte.ts`

- [ ] **Step 1: Extend types**

Update `FormState`:

```ts
export type FormState =
	| 'select-standard'
	| 'select-version'
	| 'select-suite'
	| 'select-test'
	| 'select-wallet-action'
	| 'ready'
	| 'loading'
	| 'error';
```

Remove `action_id` from `TestOption`:

```ts
export type TestOption = {
	test: Test;
	testName: string;
	enabled: boolean;
};
```

- [ ] **Step 2: Add derived helpers on the class**

After `hasWalletTests`:

```ts
	genericCredentialActions = $derived(this.walletActions.current ?? []);

	selectedWalletAction = $derived.by(() => {
		if (!this.data.action_id) return undefined;
		return this.genericCredentialActions.find(
			(action) => getPath(action) === this.data.action_id
		);
	});
```

- [ ] **Step 3: Replace `state` derived**

```ts
	state: FormState = $derived.by(() => {
		const { standard, version, suite, test, action_id } = this.data;
		if (this.standardsWithTestSuites.loading) {
			return 'loading';
		} else if (this.standardsWithTestSuites.error) {
			return 'error';
		} else if (!standard) {
			return 'select-standard';
		} else if (standard && !version) {
			return 'select-version';
		} else if (standard && version && !suite) {
			return 'select-suite';
		} else if (standard && version && suite && !test) {
			return 'select-test';
		} else if (
			standard &&
			version &&
			suite &&
			test &&
			isOpenId4VciWalletTest(test) &&
			resolveWalletActionSelection(this.genericCredentialActions).kind === 'picker' &&
			!action_id
		) {
			return 'select-wallet-action';
		} else if (standard && version && suite && test) {
			return 'ready';
		} else {
			throw new Error(m.Pipeline_form_invalid_state());
		}
	});
```

- [ ] **Step 4: Simplify `testOptions` — remove silent `action_id` bind**

```ts
	testOptions: TestOption[] = $derived.by(() => {
		const wallet = ExecutionTarget.state.current?.wallet;
		const walletTestsBlocked =
			this.hasWalletTests &&
			(!wallet || this.walletActions.loading || getWalletTestBlockReason(wallet, this.walletActions));

		return this.availableTests.map((test) => {
			const testName = test.split('/').at(-1) ?? test;

			if (!isOpenId4VciWalletTest(test)) {
				return { test, testName, enabled: true };
			}

			return { test, testName, enabled: !walletTestsBlocked };
		});
	});
```

- [ ] **Step 5: Refactor `selectTest`, add `selectWalletAction`, update discard methods**

Replace `selectTest` and add methods:

```ts
	selectTest(option: TestOption) {
		if (!option.enabled) return;

		this.data.test = option.test;

		if (!isOpenId4VciWalletTest(option.test)) {
			this.data.action_id = undefined;
			if (this.intent === 'add') {
				this.commit({ ...this.data, test: option.test } as FormData);
			}
			return;
		}

		const selection = resolveWalletActionSelection(this.genericCredentialActions);

		if (selection.kind === 'auto') {
			this.data.action_id = getPath(selection.action);
			if (this.intent === 'add') {
				this.commit({
					...this.data,
					test: option.test,
					action_id: this.data.action_id
				} as FormData);
			}
			return;
		}

		if (selection.kind === 'picker') {
			this.data.action_id = undefined;
			return;
		}
	}

	selectWalletAction(action: WalletActionsResponse) {
		this.data.action_id = getPath(action);
		if (this.intent === 'add') {
			this.commit({
				...this.data,
				action_id: this.data.action_id
			} as FormData);
		}
	}

	discardTest() {
		this.data.test = undefined;
		this.data.action_id = undefined;
	}

	discardWalletAction() {
		this.data.action_id = undefined;
	}
```

- [ ] **Step 6: Run unit tests**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.test.ts --run`

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.svelte.ts
git commit -m "feat(conformance-check): add wallet action picker funnel state"
```

---

### Task 3: Template — breadcrumb and action picker UI

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.svelte`

- [ ] **Step 1: Add imports**

After existing imports, add:

```ts
	import WalletActionTags from '$lib/components/wallet-action-tags.svelte';
	import { Wallet } from '$lib/wallet';

	import { Badge } from '@/components/ui/badge';
```

- [ ] **Step 2: Extend `selectLabel`**

```ts
	const selectLabel = $derived.by(() => {
		if (form.state === 'select-standard') {
			return m.Standard();
		} else if (form.state === 'select-version') {
			return m.Version();
		} else if (form.state === 'select-suite') {
			return m.Suite();
		} else if (form.state === 'select-test') {
			return m.Test();
		} else if (form.state === 'select-wallet-action') {
			return m.Wallet_action();
		} else {
			return '';
		}
	});
```

- [ ] **Step 3: Add wallet-action breadcrumb row**

Inside the `{#if hasSelection}` block, after the test row:

```svelte
				{#if form.data.action_id && form.selectedWalletAction}
					<WithLabel label={m.Wallet_action()}>
						<ItemCard
							title={form.selectedWalletAction.name}
							onDiscard={() => form.discardWalletAction()}
						/>
					</WithLabel>
				{/if}
```

- [ ] **Step 4: Add `select-wallet-action` picker block**

In the picker `{#if form.state !== 'ready'}` chain, after the `select-test` branch:

```svelte
					{:else if form.state === 'select-wallet-action'}
						{#each form.genericCredentialActions as action (action.id)}
							<ItemCard
								title={action.name}
								onClick={() => form.selectWalletAction(action)}
							>
								{#snippet beforeContent()}
									{@const category = Wallet.Action.getCategoryLabel(action)}
									{#if category}
										<T class="text-xs text-muted-foreground">{category}</T>
									{/if}
								{/snippet}
								{#snippet afterContent()}
									<WalletActionTags
										action={action}
										variant="secondary"
										containerClass="pt-2"
									>
										{#if !action.published}
											<Badge variant="outline">
												{m.private()}
											</Badge>
										{/if}
									</WalletActionTags>
								{/snippet}
							</ItemCard>
						{/each}
```

- [ ] **Step 5: Run Svelte check and lint**

Run: `cd webapp && bun run check`

Expected: no errors in conformance-check files

Run: `cd webapp && bun run lint`

Expected: pass (or only pre-existing issues elsewhere)

- [ ] **Step 6: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.svelte
git commit -m "feat(conformance-check): wallet action picker UI in funnel"
```

---

### Task 4: Edit-intent behavior test (optional guard)

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.test.ts`

- [ ] **Step 1: Add edit-intent test**

Append to test file:

```ts
import { ConformanceCheckStepForm } from './conformance-check-step-form.svelte.js';

describe('ConformanceCheckStepForm edit intent', () => {
	it('selectWalletAction does not commit until commit()', () => {
		const onSubmit = vi.fn();
		const form = new ConformanceCheckStepForm({
			intent: 'edit',
			initial: {
				standard: { uid: 's', name: 'S', versions: [] } as never,
				version: { uid: 'v', name: 'V', suites: [] } as never,
				suite: { uid: 'su', name: 'Su', paths: [] } as never,
				test: 'openid4vci_wallet/foo',
				action_id: 'owners/w/actions/old'
			}
		});
		form.onSubmit(onSubmit);

		const newAction = {
			id: 'a2',
			name: 'New',
			category: 'get-credential-generic',
			wallet: 'w1'
		} as never;

		form.walletActions.current = [newAction];
		form.data.test = 'openid4vci_wallet/foo';
		form.selectWalletAction(newAction);

		expect(onSubmit).not.toHaveBeenCalled();
		expect(form.data.action_id).toBeTruthy();
	});
});
```

Note: if `walletActions.current` is read-only on the resource object, set selection via `Object.assign(form.walletActions, { current: [newAction] })` or mock at construction time — adjust to whatever works at runtime; the assertion that matters is `onSubmit` not called on edit intent.

- [ ] **Step 2: Run tests**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.test.ts --run`

Expected: PASS

- [ ] **Step 3: Commit (if test added and passing)**

```bash
git add webapp/src/lib/pipeline-form/steps/conformance-check/conformance-check-step-form.test.ts
git commit -m "test(conformance-check): edit intent does not auto-commit action pick"
```

Skip this task if mocking `walletActions` on the class instance is impractical without refactoring — the helper tests in Task 1 are sufficient for v1.

---

### Task 5: Manual UAT

**Files:** none

- [ ] **Step 1: Start dev stack**

Run: `make dev` (or ensure API + webapp running)

- [ ] **Step 2: Walk UAT checklist from design spec**

1. Wallet with **1** `get-credential-generic` action → one click completes; YAML has correct `action_id`
2. Wallet with **2+** actions → test click → action picker → chosen `action_id` in YAML
3. **Discard** action → action picker; discard test → test picker
4. Single-test suite + **multiple actions** → auto-selects test only, then action picker
5. **Edit** existing step → breadcrumb shows test + action; save preserves `action_id`
6. Non-wallet tests → no action step
7. Disabled states (no wallet / no action / loading) → unchanged

- [ ] **Step 3: Record result**

Note pass/fail in PR description or commit message if fixing issues found during UAT.

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| Funnel step `select-wallet-action` when 2+ actions | Task 2 Step 3, Task 3 Step 4 |
| Skip action step when 1 action | Task 1 helper, Task 2 Step 5 |
| Wallet-action card styling | Task 3 Step 4 |
| Breadcrumb + discard action | Task 3 Step 3, Task 2 Step 5 |
| Reuse `walletActions` resource | Task 2 (no new fetch) |
| `serialize()` unchanged | File map — no edit |
| Auto-select single test + multi action → test only | Task 2 `selectSuite` unchanged (calls `selectTest`) |
| Edit/deserialize with `action_id` → ready | Task 2 state derived |
| Disabled-state behavior unchanged | Task 1 `getWalletTestBlockReason` length check |

## Self-review

- No TBD/TODO placeholders in tasks
- All file paths are explicit
- Code blocks are complete for each change
- Type names consistent: `WalletActionSelection`, `resolveWalletActionSelection`, `genericCredentialActions`, `selectWalletAction`, `discardWalletAction`
- Task 4 marked skippable if resource mocking is awkward — helper tests satisfy v1 spec

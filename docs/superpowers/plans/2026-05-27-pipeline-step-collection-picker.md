# Pipeline Step Collection Picker Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace hand-rolled search/list logic in PB-backed pipeline funnel sub-steps with a shared `StepCollectionPicker` wrapper around `CollectionManager`, giving pagination and search while keeping `ItemCard` visuals and slim step form classes.

**Architecture:** Add `step-collection-picker.svelte` in `steps/_partials/` configuring `CollectionManager` for single-select funnel use (search + paginated `ItemCard` list, optional `prepend`). Migrate `HubItemStepForm` and `WalletActionStepForm` PB sub-steps; step `*.svelte.ts` files keep selection/state machine only. Delete `search-hub.ts` when unused.

**Tech Stack:** Svelte 5 runes, existing `CollectionManager` / `PocketbaseQueryAgent`, Paraglide (`m.*`), Vitest.

**Design spec:** `docs/superpowers/specs/2026-05-27-pipeline-step-collection-picker-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `webapp/src/lib/pipeline-form/steps/_partials/step-collection-picker.svelte` | **Create** — funnel picker wrapper around `CollectionManager` |
| `webapp/src/lib/pipeline-form/steps/hub-item/hub-item-step-form.svelte.ts` | Remove search/fetch; keep selection handlers |
| `webapp/src/lib/pipeline-form/steps/hub-item/hub-item-step-form.svelte` | Wire `StepCollectionPicker` |
| `webapp/src/lib/pipeline-form/steps/hub-item/hub-item-step-form.test.ts` | **Create** — selection/commit unit tests |
| `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte.ts` | Remove search/fetch for PB sub-steps; simplify `selectWallet` |
| `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte` | Wire pickers for wallet/version/action sub-steps |
| `webapp/src/lib/pipeline-form/steps/_partials/search-hub.ts` | **Delete** after migration |

**Unchanged:** runner sub-step, conformance check, `search.svelte.ts`, `search-input.svelte`, `with-empty-state.svelte`.

---

### Task 1: `StepCollectionPicker` component

**Files:**
- Create: `webapp/src/lib/pipeline-form/steps/_partials/step-collection-picker.svelte`

- [ ] **Step 1: Create the wrapper**

```svelte
<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script
	lang="ts"
	generics="C extends CollectionName, E extends PocketbaseQueryExpandOption<C> = never"
>
	import type { Snippet } from 'svelte';
	import type { ClassValue } from 'svelte/elements';

	import type { CollectionName } from '@/pocketbase/collections-models';
	import type {
		PocketbaseQueryAgentOptions,
		PocketbaseQueryExpandOption,
		PocketbaseQueryOptions,
		PocketbaseQueryResponse
	} from '@/pocketbase/query';
	import type { CollectionResponses } from '@/pocketbase/types';

	import { CollectionManager } from '@/collections-components';
	import { ScrollArea } from '@/components/ui/scroll-area';
	import { m } from '@/i18n';

	import ItemCard from './item-card.svelte';
	import WithLabel from './with-label.svelte';

	type Props = {
		collection: C;
		queryOptions: PocketbaseQueryOptions<C, E>;
		queryAgentOptions?: PocketbaseQueryAgentOptions;
		onSelect: (record: CollectionResponses[C]) => void;
		label?: string;
		class?: ClassValue;
		emptyText?: string;
		prepend?: Snippet;
		item?: Snippet<
			[
				{
					record: PocketbaseQueryResponse<C, E>;
					onSelect: (record: CollectionResponses[C]) => void;
				}
			]
		>;
	};

	let {
		collection,
		queryOptions,
		queryAgentOptions = {},
		onSelect,
		label,
		class: className,
		emptyText,
		prepend,
		item: itemSnippet
	}: Props = $props();

	function handleSelect(record: PocketbaseQueryResponse<C, E>) {
		onSelect(record as CollectionResponses[C]);
	}
</script>

<div class={['flex min-h-0 flex-col', className]}>
	<CollectionManager
		{collection}
		queryOptions={{ perPage: 10, ...queryOptions }}
		{queryAgentOptions}
		hide={['empty_state']}
	>
		{#snippet top({ Search })}
			{@render prepend?.()}
			{#if label}
				<WithLabel {label} class="p-4">
					<Search />
				</WithLabel>
			{:else}
				<div class="p-4">
					<Search />
				</div>
			{/if}
		{/snippet}

		{#snippet records({ records })}
			<ScrollArea class="grow [&>div>div]:space-y-2 [&>div>div]:p-4">
				{#each records as record (record.id)}
					{#if itemSnippet}
						{@render itemSnippet({ record, onSelect: handleSelect })}
					{:else}
						<ItemCard
							title={'name' in record && record.name ? String(record.name) : record.id}
							onClick={() => handleSelect(record)}
						/>
					{/if}
				{/each}
			</ScrollArea>
		{/snippet}

		{#snippet emptyState({ EmptyState })}
			<EmptyState title={emptyText ?? m.No_items_here()} />
		{/snippet}
	</CollectionManager>
</div>
```

- [ ] **Step 2: Typecheck the webapp**

Run: `cd webapp && bun run check`
Expected: PASS (no errors in new file)

- [ ] **Step 3: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/_partials/step-collection-picker.svelte
git commit -m "feat(pipeline-form): add StepCollectionPicker wrapper for funnel sub-steps"
```

---

### Task 2: Migrate `HubItemStepForm`

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/hub-item/hub-item-step-form.svelte.ts`
- Modify: `webapp/src/lib/pipeline-form/steps/hub-item/hub-item-step-form.svelte`
- Create: `webapp/src/lib/pipeline-form/steps/hub-item/hub-item-step-form.test.ts`

- [ ] **Step 1: Write hub-item step form tests**

Create `hub-item-step-form.test.ts`:

```ts
// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it, vi } from 'vitest';

vi.mock('./hub-item-step-form.svelte', () => ({ default: class {} }));

import { HubItemStepForm } from './hub-item-step-form.svelte.js';

const hubItem = {
	id: 'h1',
	name: 'Test Credential',
	organization_name: 'Org',
	type: 'credentials'
} as never;

describe('HubItemStepForm', () => {
	it('selectItem commits on add intent', () => {
		const onSubmit = vi.fn();
		const form = new HubItemStepForm(
			{ collection: 'credentials', entityData: { labels: { singular: 'Credential' } } as never },
			{ intent: 'add' }
		);
		form.onSubmit(onSubmit);
		form.selectItem(hubItem);
		expect(onSubmit).toHaveBeenCalledOnce();
		expect(onSubmit.mock.calls[0][0]).toBe(hubItem);
	});

	it('selectItem does not commit on edit intent', () => {
		const onSubmit = vi.fn();
		const form = new HubItemStepForm(
			{ collection: 'credentials', entityData: { labels: { singular: 'Credential' } } as never },
			{ intent: 'edit', initial: hubItem }
		);
		form.onSubmit(onSubmit);
		form.selectItem({ ...hubItem, id: 'h2', name: 'Other' } as never);
		expect(onSubmit).not.toHaveBeenCalled();
		expect(form.selectedItem?.id).toBe('h2');
	});

	it('discardSelection clears selectedItem', () => {
		const form = new HubItemStepForm(
			{ collection: 'credentials', entityData: { labels: { singular: 'Credential' } } as never },
			{ intent: 'edit', initial: hubItem }
		);
		form.discardSelection();
		expect(form.selectedItem).toBeUndefined();
	});
});
```

- [ ] **Step 2: Run tests to verify they pass with current class (or fail only after slim-down)**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/hub-item/hub-item-step-form.test.ts --run`
Expected: PASS (handlers unchanged)

- [ ] **Step 3: Slim down `hub-item-step-form.svelte.ts`**

Replace file contents with:

```ts
// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { EntityData } from '$lib/global';
import type { HubItem, HubItemType } from '$lib/hub';

import { BaseForm, type InitFormOptions } from '../types';
import Component from './hub-item-step-form.svelte';

const collections = [
	'credentials',
	'use_cases_verifications',
	'custom_checks'
] as const satisfies HubItemType[];

type HubStepCollection = (typeof collections)[number];

type Props = {
	collection: HubStepCollection;
	entityData: EntityData;
};

export class HubItemStepForm extends BaseForm<HubItem, HubItemStepForm> {
	readonly Component = Component;

	selectedItem = $state<HubItem | undefined>(undefined);

	constructor(
		private props: Props,
		opts?: InitFormOptions<HubItem>
	) {
		super(opts);
		if (opts?.initial) {
			this.selectedItem = opts.initial;
		}
	}

	canSave() {
		return this.selectedItem !== undefined;
	}

	getSubmitData() {
		return this.selectedItem;
	}

	selectItem(item: HubItem) {
		this.selectedItem = item;
		if (this.intent === 'add') {
			this.commit(item);
		}
	}

	discardSelection() {
		this.selectedItem = undefined;
	}

	get collection() {
		return this.props.collection;
	}

	get entityData() {
		return this.props.entityData;
	}
}
```

- [ ] **Step 4: Update `hub-item-step-form.svelte`**

Replace search/list block with picker. Full script + template changes:

```svelte
<script lang="ts">
	import type { SelfProp } from '$lib/renderable';

	import { getHubItemLogo, getHubItemTypeFilter } from '$lib/hub/utils.js';

	import type { HubItemStepForm } from './hub-item-step-form.svelte.js';

	import ItemCard from '../_partials/item-card.svelte';
	import StepCollectionPicker from '../_partials/step-collection-picker.svelte';
	import WithLabel from '../_partials/with-label.svelte';

	let { self: form }: SelfProp<HubItemStepForm> = $props();

	const { labels } = $derived(form.entityData);
</script>

{#if form.selectedItem}
	<div class="border-b p-4">
		<WithLabel label={labels.singular}>
			<ItemCard
				avatar={getHubItemLogo(form.selectedItem)}
				title={form.selectedItem.name}
				subtitle={form.selectedItem.organization_name}
				onDiscard={() => form.discardSelection()}
			/>
		</WithLabel>
	</div>
{/if}

<StepCollectionPicker
	collection="hub_items"
	label={labels.singular}
	queryOptions={{
		filter: getHubItemTypeFilter(form.collection),
		searchFields: ['name']
	}}
	onSelect={(record) => form.selectItem(record as import('$lib/hub').HubItem)}
>
	{#snippet item({ record, onSelect })}
		{@const item = record as import('$lib/hub').HubItem}
		<ItemCard
			avatar={getHubItemLogo(item)}
			title={item.name}
			subtitle={item.organization_name}
			onClick={() => onSelect(item)}
		/>
	{/snippet}
</StepCollectionPicker>
```

Prefer a top-level `import type { HubItem } from '$lib/hub'` instead of inline `import('$lib/hub').HubItem` if lint complains.

- [ ] **Step 5: Run hub-item tests**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/hub-item/hub-item-step-form.test.ts --run`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/hub-item/
git commit -m "refactor(pipeline-form): migrate HubItemStepForm to StepCollectionPicker"
```

---

### Task 3: Migrate `WalletActionStepForm` (class)

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte.ts`

- [ ] **Step 1: Run existing wallet-action tests (baseline)**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.test.ts --run`
Expected: PASS

- [ ] **Step 2: Remove fetch/search state from class**

In `wallet-action-step-form.svelte.ts`:

1. Remove imports: `searchHub`, `Search` (keep `pb` only if still used — it won't be after this task).
2. Delete: `foundWallets`, `foundVersions`, `foundActions`, `walletSearch`, `actionSearch`, `searchWallet`, `searchAction`.
3. Replace `selectWallet`:

```ts
selectWallet(wallet: HubItem) {
	this.data.wallet = wallet;
	if (ExecutionTarget.hasGlobalRunner() || ExecutionTarget.hasUndefinedRunner()) {
		this.data.runner = 'global';
	}
}
```

4. Update `removeWallet` — remove lines clearing `foundVersions` / `foundActions`:

```ts
removeWallet() {
	this.data.wallet = undefined;
	this.data.version = undefined;
	this.data.runner = undefined;
	this.data.action = undefined;
}
```

5. Keep `runnerSearch`, `selectVersion`, `selectExternalVersion`, `selectRunner`, `selectAction`, `removeAction`, `removeVersion`, `removeRunner` unchanged.

- [ ] **Step 3: Re-run wallet-action tests**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.test.ts --run`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte.ts
git commit -m "refactor(pipeline-form): slim WalletActionStepForm class for collection picker"
```

---

### Task 4: Migrate `WalletActionStepForm` (view)

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte`

- [ ] **Step 1: Add imports**

Add:
```ts
import StepCollectionPicker from '../_partials/step-collection-picker.svelte';
import { getHubItemTypeFilter } from '$lib/hub/utils.js';
```

Remove unused `SearchInput` and `WithEmptyState` imports if no longer referenced (runner step still uses `SearchInput`).

- [ ] **Step 2: Replace `select-wallet` block**

```svelte
{:else if form.state === 'select-wallet'}
	<StepCollectionPicker
		collection="hub_items"
		label={m.Wallet()}
		queryOptions={{
			filter: getHubItemTypeFilter('wallets'),
			searchFields: ['name']
		}}
		onSelect={(record) => form.selectWallet(record as import('$lib/hub').HubItem)}
	>
		{#snippet item({ record, onSelect })}
			{@const item = record as import('$lib/hub').HubItem}
			{@const data = getHubItemData(item)}
			<ItemCard
				avatar={data.logo}
				title={item.name}
				subtitle={item.organization_name}
				onClick={() => onSelect(item)}
			/>
		{/snippet}
	</StepCollectionPicker>
```

- [ ] **Step 3: Replace `select-version` block**

```svelte
{:else if form.state === 'select-version'}
	<StepCollectionPicker
		collection="wallet_versions"
		label={m.Version()}
		queryOptions={{
			filter: `wallet = '${form.data.wallet!.id}'`,
			sort: ['tag', 'DESC']
		}}
		emptyText={m.No_wallet_versions_found()}
		onSelect={(record) => form.selectVersion(record)}
	>
		{#snippet prepend()}
			<div class="px-4">
				<ItemCard
					title={m.Install_from_external_source()}
					onClick={() => form.selectExternalVersion()}
				>
					{#snippet titleRight()}
						<span class="ml-0.5 inline-flex translate-0.5 gap-1 text-gray-300">
							<ExternalLinkIcon size={16} class="stroke-2" />
						</span>
					{/snippet}
				</ItemCard>
			</div>
		{/snippet}
		{#snippet item({ record, onSelect })}
			<ItemCard title={record.tag} onClick={() => onSelect(record)}>
				{#snippet titleRight()}
					<span class="ml-0.5 inline-flex translate-0.5 gap-1 text-gray-300">
						{#if record.android_installer}
							<AndroidLogo size={16} />
						{/if}
						{#if record.ios_installer}
							<AppleLogo size={16} />
						{/if}
					</span>
				{/snippet}
			</ItemCard>
		{/snippet}
	</StepCollectionPicker>
```

Remove the standalone `<WithLabel label={m.Version()} class="p-4" />` that preceded the external-source card (label is on the picker now).

- [ ] **Step 4: Replace `select-action` block**

```svelte
{:else if form.state === 'select-action'}
	<StepCollectionPicker
		collection="wallet_actions"
		label={m.Wallet_action()}
		queryOptions={{
			filter: `wallet = '${form.data.wallet!.id}'`,
			searchFields: ['name', 'canonified_name'],
			sort: ['category', 'ASC']
		}}
		emptyText={m.No_actions_available()}
		onSelect={(record) => form.selectAction(record)}
	>
		{#snippet item({ record, onSelect })}
			<ItemCard title={record.name} onClick={() => onSelect(record)}>
				{#snippet beforeContent()}
					{@const category = Wallet.Action.getCategoryLabel(record)}
					{#if category}
						<T class="text-xs text-muted-foreground">{category}</T>
					{/if}
				{/snippet}
				{#snippet afterContent()}
					<WalletActionTags action={record} variant="secondary" containerClass="pt-2">
						{#if !record.published}
							<Badge variant="outline">
								{m.private()}
							</Badge>
						{/if}
					</WalletActionTags>
				{/snippet}
			</ItemCard>
		{/snippet}
	</StepCollectionPicker>
```

- [ ] **Step 5: Leave `select-runner` block unchanged**

- [ ] **Step 6: Typecheck**

Run: `cd webapp && bun run check`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte
git commit -m "refactor(pipeline-form): migrate WalletActionStepForm views to StepCollectionPicker"
```

---

### Task 5: Delete `search-hub.ts` and verify

**Files:**
- Delete: `webapp/src/lib/pipeline-form/steps/_partials/search-hub.ts`

- [ ] **Step 1: Confirm no remaining imports**

Run: `rg search-hub webapp/src`
Expected: no matches

- [ ] **Step 2: Delete the file**

```bash
git rm webapp/src/lib/pipeline-form/steps/_partials/search-hub.ts
```

- [ ] **Step 3: Run unit tests for touched step forms**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline-form/steps/hub-item/hub-item-step-form.test.ts src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.test.ts --run`
Expected: PASS

- [ ] **Step 4: Lint webapp**

Run: `cd webapp && bun run lint`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git commit -m "chore(pipeline-form): remove search-hub helper after picker migration"
```

---

### Task 6: Manual smoke test

- [ ] **Step 1: Start dev stack** (if not running)

Run: `make dev` (or backend + `cd webapp && bun dev`)

- [ ] **Step 2: Hub-item step**

Open pipeline composer → add credential-offer (or use-case-verification / custom-check) step:
- Search by name returns results
- Pagination appears when >10 items
- Select item advances funnel / commits on add

- [ ] **Step 3: Wallet-action step**

Add mobile-automation step:
- Wallet picker: search + paginate + select
- Version picker: external source prepend works; version list filtered by wallet
- Runner step: unchanged
- Action picker: search + select; tags render

- [ ] **Step 4: Edit wallet-action step**

Edit existing step → change action → Save does not auto-submit on select; explicit Save works.

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| `StepCollectionPicker` wrapper | Task 1 |
| Single-click, no CRUD/checkboxes | Task 1 (no RecordSelect/RecordCreate in picker) |
| `ItemCard` visuals | Tasks 2, 4 (custom `item` snippets) |
| `prepend` for external version | Task 4 |
| Hub search on `name` | Task 2 |
| `perPage: 10` | Task 1 default merge |
| HubItemStepForm slim-down | Task 2 |
| WalletActionStepForm slim-down | Tasks 3–4 |
| Runner/conformance unchanged | Tasks 4–6 verify |
| Delete `search-hub.ts` | Task 5 |
| Hub-item unit tests | Task 2 |
| Wallet-action tests preserved | Tasks 3, 5 |

## Non-goals (do not implement)

- Conformance check picker migration
- Runner catalog picker migration
- Loading skeleton
- `CollectionQuery` extraction
- `mode="picker"` on `CollectionManager`

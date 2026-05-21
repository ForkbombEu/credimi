# Pipeline Runner Catalog Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor mobile-runner selection to use a single coalesced catalog snapshot (`Pipeline.Runner.Catalog`), remove `StatusCoordinator`, and rename list API JSON fields (`path`, `is_owned`, `is_online`, …) with a hard break.

**Architecture:** Backend `MobileRunnerListItem` gets accurate json tags. Webapp adds `runners/query.ts` (Task + zod), `runners/search.ts` (pure filter), `runners/catalog.svelte.ts` (state + coalesced refresh + `isReady`), and `runner/index.ts` exports `Binding`, `Catalog`, `Record`, and Svelte components. Delete `Pipeline.Runners` and the status subsystem.

**Tech Stack:** Go 1.x (`pkg/internal/apis/handlers`), OpenAPI YAML, Svelte 5 runes, Paraglide i18n, Vitest, `true-myth/task`, PocketBase JS SDK, `pb.send`.

**Design spec:** `docs/superpowers/specs/2026-05-20-pipeline-runner-catalog-design.md`

**Recommended worktree:** dedicated branch/worktree (see superpowers:using-git-worktrees).

---

## File map

| File | Action | Responsibility |
|------|--------|----------------|
| `pkg/internal/apis/handlers/mobile_runners_handlers.go` | Modify | Rename struct fields + json tags |
| `pkg/internal/apis/handlers/mobile_runners_handlers_test.go` | Modify | Assert new JSON keys |
| `docs/public/API/openapi.yml` | Modify | `HandlersMobileRunnerListItem` schema |
| `webapp/src/lib/pipeline/runners/types.ts` | Create | `RunnerRecord` type |
| `webapp/src/lib/pipeline/runners/query.ts` | Create | `listSelector()`, zod wire → `RunnerRecord` |
| `webapp/src/lib/pipeline/runners/search.ts` | Create | Pure `filterRunners()` |
| `webapp/src/lib/pipeline/runners/catalog.svelte.ts` | Create | Store, `refresh`, `isReady`, live refresh |
| `webapp/src/lib/pipeline/runner/index.ts` | Modify | `Binding`, `Catalog`, `Record` barrel |
| `webapp/src/lib/pipeline/index.ts` | Modify | Remove `export * as Runners` |
| `webapp/src/lib/pipeline/runner/binding.ts` | Modify | Accept `RunnerRecord`, store `path` |
| `webapp/src/lib/pipeline/runner/binding.test.ts` | Modify | `RunnerRecord` + `path` |
| `webapp/src/lib/pipeline/runners/query.test.ts` | Create | Zod parse fixture |
| `webapp/src/lib/pipeline/runners/search.test.ts` | Create | Filter tests |
| `webapp/src/lib/pipeline/runners/catalog.test.ts` | Create | `isReady` + mocked `listSelector` |
| `webapp/src/lib/pipeline/runner/run-now-button.svelte` | Modify | Catalog offline/checking |
| `webapp/src/lib/pipeline/runner/runner-select-input.svelte` | Modify | Derived search, `isOnline` |
| `webapp/src/lib/pipeline/runner/runner-select-modal.svelte` | Modify | Modal refresh, `findByPath` |
| `webapp/src/lib/pipeline-form/steps/wallet-action/index.ts` | Modify | `listSelector` deserialize |
| `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte.ts` | Modify | Remove async search state |
| `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte` | Modify | `$derived` catalog search |
| `webapp/src/routes/my/pipelines/+layout.svelte` | Modify | `Catalog.startLiveRefresh()` |
| `webapp/src/routes/my/pipelines/_partials/schedule-pipeline-form.svelte` | Modify | `Record` + `path` |
| `webapp/src/hooks.client.ts` | Modify | `Runner.Catalog.init/dispose` |
| `webapp/messages/*.json` | Modify | `Runner_status_checking` (7 locales) |
| `AGENTS.md` | Modify | Note list API field names |
| `webapp/src/lib/pipeline/runners/utils.ts` | Delete | Replaced by query + catalog |
| `webapp/src/lib/pipeline/runners/store.svelte.ts` | Delete | Replaced by catalog |
| `webapp/src/lib/pipeline/runners/index.ts` | Delete | Barrel moved to `runner/index.ts` |
| `webapp/src/lib/pipeline/runners/status*.ts` | Delete | Status subsystem removed |

---

### Task 1: Backend list item JSON rename

**Files:**
- Modify: `pkg/internal/apis/handlers/mobile_runners_handlers.go`
- Modify: `pkg/internal/apis/handlers/mobile_runners_handlers_test.go`
- Modify: `docs/public/API/openapi.yml`

- [ ] **Step 1: Update `MobileRunnerListItem` struct**

In `mobile_runners_handlers.go`, replace the struct (lines ~48–59):

```go
type MobileRunnerListItem struct {
	Name        string                     `json:"name"`
	Path        string                     `json:"path"`
	URL         string                     `json:"url,omitempty"`
	Description string                     `json:"description,omitempty"`
	Type        string                     `json:"type,omitempty"`
	IsPublished bool                       `json:"is_published"`
	IsOwned     bool                       `json:"is_owned"`
	IsOnline    bool                       `json:"is_online"`
	Devices     []MobileRunnerHealthDevice `json:"devices,omitempty"`
	QueueLength *int                       `json:"queue_length,omitempty"`
}
```

- [ ] **Step 2: Update `mobileRunnerListItem` assignment and sort**

In the same file, set fields on `item`:

```go
item := MobileRunnerListItem{
	Name:        record.GetString("name"),
	Path:        runnerID,
	Description: record.GetString("description"),
	IsPublished: record.GetBool("published"),
	IsOwned:     callerOrgID != "" && record.GetString("owner") == callerOrgID,
	IsOnline:    online,
}
```

In `includeDetails` block use `item.URL`, `item.Type`, `item.Devices`, `item.QueueLength`.

Update `sort.SliceStable` to compare `IsOwned`, `IsOnline`, `Path` instead of `Mine`, `Online`, `RunnerID`.

- [ ] **Step 3: Update handler tests**

In `mobile_runners_handlers_test.go`, replace assertions:

- `response.Runners[0].RunnerID` → `response.Runners[0].Path`
- `.Mine` → `.IsOwned`
- `.Online` → `.IsOnline`
- `.Published` → `.IsPublished` (if asserted)
- `.QueueLen` → `.QueueLength`
- `.RunnerURL` → `.URL`

In selector subtest raw JSON checks, replace:

```go
require.NotContains(t, raw["runners"][0], "queue_len")
require.NotContains(t, raw["runners"][0], "runner_url")
require.NotContains(t, raw["runners"][0], "runner_id")
require.Contains(t, raw["runners"][0], "path")
require.Contains(t, raw["runners"][0], "is_online")
```

- [ ] **Step 4: Update OpenAPI schema**

In `docs/public/API/openapi.yml`, under `HandlersMobileRunnerListItem.properties`, replace keys:

- `runner_id` → `path` (string)
- `runner_url` → `url`
- `mine` → `is_owned`
- `published` → `is_published`
- `online` → `is_online`
- `queue_len` → `queue_length`

Remove old property names.

- [ ] **Step 5: Run Go tests**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run TestListMobileRunners -v`

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/internal/apis/handlers/mobile_runners_handlers.go \
  pkg/internal/apis/handlers/mobile_runners_handlers_test.go \
  docs/public/API/openapi.yml
git commit -m "feat(api): rename mobile runner list item JSON fields"
```

---

### Task 2: Webapp query layer + zod tests

**Files:**
- Create: `webapp/src/lib/pipeline/runners/types.ts`
- Create: `webapp/src/lib/pipeline/runners/query.ts`
- Create: `webapp/src/lib/pipeline/runners/query.test.ts`

- [ ] **Step 1: Write failing zod test**

Create `query.test.ts`:

```ts
// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { parseSelectorResponse } from './query';

describe('parseSelectorResponse', () => {
	it('maps snake_case API body to RunnerRecord', () => {
		const records = parseSelectorResponse({
			runners: [
				{
					name: 'Online owned',
					path: 'usera-s-organization/owned-online',
					description: 'desc',
					is_owned: true,
					is_published: false,
					is_online: true
				}
			]
		});

		expect(records).toEqual([
			{
				name: 'Online owned',
				path: 'usera-s-organization/owned-online',
				description: 'desc',
				isOwned: true,
				isPublished: false,
				isOnline: true
			}
		]);
	});
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline/runners/query.test.ts`

Expected: FAIL — `parseSelectorResponse` not exported / not defined

- [ ] **Step 3: Implement `types.ts` and `query.ts`**

`types.ts`:

```ts
// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export type RunnerRecord = {
	name: string;
	path: string;
	description?: string;
	isOwned: boolean;
	isPublished: boolean;
	isOnline: boolean;
	url?: string;
	type?: string;
	queueLength?: number;
};
```

`query.ts`:

```ts
// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { ClientResponseError } from 'pocketbase';
import * as Task from 'true-myth/task';
import { z, ZodError } from 'zod';

import { pb } from '@/pocketbase';

import type { RunnerRecord } from './types';

const runnerWireSchema = z.object({
	name: z.string(),
	path: z.string(),
	description: z.string().optional(),
	is_owned: z.boolean(),
	is_published: z.boolean(),
	is_online: z.boolean(),
	url: z.string().optional(),
	type: z.string().optional(),
	queue_length: z.number().optional()
});

const listResponseSchema = z.object({
	runners: z.array(runnerWireSchema)
});

function mapWireToRecord(wire: z.infer<typeof runnerWireSchema>): RunnerRecord {
	return {
		name: wire.name,
		path: wire.path,
		description: wire.description,
		isOwned: wire.is_owned,
		isPublished: wire.is_published,
		isOnline: wire.is_online,
		url: wire.url,
		type: wire.type,
		queueLength: wire.queue_length
	};
}

export function parseSelectorResponse(body: unknown): RunnerRecord[] {
	const parsed = listResponseSchema.parse(body);
	return parsed.runners.map(mapWireToRecord);
}

export function listSelector(
	options: { fetch?: typeof fetch } = {}
): Task.Task<RunnerRecord[], ClientResponseError | ZodError> {
	const { fetch: fetchFn = fetch } = options;

	return Task.tryOrElse(
		(err) => err as ClientResponseError,
		() =>
			pb.send('/api/mobile-runners?view=selector', {
				method: 'GET',
				fetch: fetchFn,
				requestKey: null
			})
	).andThen((response) => {
		try {
			return Task.resolve(parseSelectorResponse(response));
		} catch (error) {
			return Task.reject(error as ZodError);
		}
	});
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline/runners/query.test.ts`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/pipeline/runners/types.ts \
  webapp/src/lib/pipeline/runners/query.ts \
  webapp/src/lib/pipeline/runners/query.test.ts
git commit -m "feat(webapp): add mobile runner selector query layer"
```

---

### Task 3: Pure search filter + tests

**Files:**
- Create: `webapp/src/lib/pipeline/runners/search.ts`
- Create: `webapp/src/lib/pipeline/runners/search.test.ts`

- [ ] **Step 1: Write failing search tests**

Create `search.test.ts`:

```ts
// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { RunnerRecord } from './types';

import { filterRunners } from './search';

const RUNNERS: RunnerRecord[] = [
	{
		name: 'Alpha Runner',
		path: 'org-a/alpha',
		isOwned: true,
		isPublished: true,
		isOnline: true
	},
	{
		name: 'Beta',
		path: 'org-b/beta-runner',
		isOwned: false,
		isPublished: true,
		isOnline: false
	}
];

describe('filterRunners', () => {
	it('returns all runners when search is empty', () => {
		expect(filterRunners(RUNNERS, '')).toHaveLength(2);
	});

	it('filters by name case-insensitively', () => {
		expect(filterRunners(RUNNERS, 'alpha')).toEqual([RUNNERS[0]]);
	});

	it('filters by path case-insensitively', () => {
		expect(filterRunners(RUNNERS, 'beta-runner')).toEqual([RUNNERS[1]]);
	});
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline/runners/search.test.ts`

Expected: FAIL

- [ ] **Step 3: Implement `search.ts`**

```ts
// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { RunnerRecord } from './types';

export function filterRunners(runners: readonly RunnerRecord[], text: string): RunnerRecord[] {
	const search = text.trim().toLowerCase();
	if (!search) return [...runners];

	return runners.filter(
		(runner) =>
			runner.name.toLowerCase().includes(search) ||
			runner.path.toLowerCase().includes(search)
	);
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline/runners/search.test.ts`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/pipeline/runners/search.ts webapp/src/lib/pipeline/runners/search.test.ts
git commit -m "feat(webapp): add catalog runner search filter"
```

---

### Task 4: Catalog store + readiness tests

**Files:**
- Create: `webapp/src/lib/pipeline/runners/catalog.svelte.ts`
- Create: `webapp/src/lib/pipeline/runners/catalog.test.ts`

- [ ] **Step 1: Write failing catalog readiness test**

Create `catalog.test.ts` (tests exported helpers without Svelte UI):

```ts
// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { RunnerRecord } from './types';

import * as query from './query';

import {
	__test__,
	applyRefreshFailure,
	applyRefreshSuccess,
	createCatalogState
} from './catalog.svelte';

const SAMPLE: RunnerRecord[] = [
	{
		name: 'R1',
		path: 'org/r1',
		isOwned: true,
		isPublished: true,
		isOnline: true
	}
];

describe('catalog readiness', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
	});

	it('isReady is false until first success', () => {
		const state = createCatalogState();
		expect(state.isReady()).toBe(false);
		applyRefreshSuccess(state, SAMPLE);
		expect(state.isReady()).toBe(true);
	});

	it('keeps snapshot and stays ready after later failure', () => {
		const state = createCatalogState();
		applyRefreshSuccess(state, SAMPLE);
		applyRefreshFailure(state);
		expect(state.isReady()).toBe(true);
		expect(state.read()).toEqual(SAMPLE);
	});
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline/runners/catalog.test.ts`

Expected: FAIL — test exports not found

- [ ] **Step 3: Implement `catalog.svelte.ts`**

Implement full module with:

- `LIVE_REFRESH_MS = 30_000`
- Internal `CatalogState` class: `#runners`, `#ready`, `#generation`, `#inFlight`, PB subscribe via `$effect.root` in `init()`
- Coalesced `refresh()`: return shared `#inFlight` promise; on success set `#runners` + `#ready = true`; on failure before ready clear; on failure after ready keep snapshot
- Exported functions: `init`, `dispose`, `refresh`, `read`, `search`, `findByPath`, `isReady`, `startLiveRefresh`
- Test exports: `createCatalogState`, `applyRefreshSuccess`, `applyRefreshFailure`, `__test__` optional

`search(text)` implementation:

```ts
export function search(text: string): RunnerRecord[] {
	return filterRunners(read(), text);
}
```

`startLiveRefresh(ms = LIVE_REFRESH_MS)`:

```ts
export function startLiveRefresh(intervalMs = LIVE_REFRESH_MS): () => void {
	void refresh();
	const timer = setInterval(() => {
		void refresh();
	}, intervalMs);
	return () => clearInterval(timer);
}
```

Wire `init()` to `userOrganization` from `$lib/app-state` (same as old store).

Use `listSelector().match({ Resolved, Rejected })` inside private `refreshForGeneration(generation)`.

- [ ] **Step 4: Run catalog tests**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline/runners/catalog.test.ts`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/pipeline/runners/catalog.svelte.ts \
  webapp/src/lib/pipeline/runners/catalog.test.ts
git commit -m "feat(webapp): add Pipeline runner catalog store"
```

---

### Task 5: Public barrel + remove old runners module

**Files:**
- Modify: `webapp/src/lib/pipeline/runner/index.ts`
- Modify: `webapp/src/lib/pipeline/index.ts`
- Delete: `webapp/src/lib/pipeline/runners/utils.ts`
- Delete: `webapp/src/lib/pipeline/runners/store.svelte.ts`
- Delete: `webapp/src/lib/pipeline/runners/index.ts`
- Delete: `webapp/src/lib/pipeline/runners/status-coordinator.ts`
- Delete: `webapp/src/lib/pipeline/runners/status.svelte.ts`
- Delete: `webapp/src/lib/pipeline/runners/status.test.ts`

- [ ] **Step 1: Replace `runner/index.ts` barrel**

```ts
// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import RunNowButton from './run-now-button.svelte';
import SelectInput from './runner-select-input.svelte';
import SelectModal from './runner-select-modal.svelte';

import * as binding from './binding';
import * as catalog from '../runners/catalog.svelte.js';

export type { RunnerRecord as Record } from '../runners/types.js';

export const Binding = {
	get: binding.get,
	getExecutionRunnerPath: binding.getExecutionRunnerPath,
	getType: binding.getType,
	isRequired: binding.isRequired,
	set: binding.set
};

export const Catalog = {
	dispose: catalog.dispose,
	findByPath: catalog.findByPath,
	init: catalog.init,
	isReady: catalog.isReady,
	read: catalog.read,
	refresh: catalog.refresh,
	search: catalog.search,
	startLiveRefresh: catalog.startLiveRefresh
};

export { RunNowButton, SelectInput, SelectModal };
```

- [ ] **Step 2: Remove `Runners` from pipeline index**

In `webapp/src/lib/pipeline/index.ts`, delete line:

```ts
export * as Runners from './runners/index.js';
```

- [ ] **Step 3: Delete obsolete runner files**

Delete the six files listed in task header (`utils.ts`, `store.svelte.ts`, `runners/index.ts`, `status-coordinator.ts`, `status.svelte.ts`, `status.test.ts`).

- [ ] **Step 4: Commit**

```bash
git add webapp/src/lib/pipeline/runner/index.ts webapp/src/lib/pipeline/index.ts
git add -u webapp/src/lib/pipeline/runners/
git commit -m "refactor(webapp): export Runner.Binding and Runner.Catalog barrel"
```

---

### Task 6: Binding uses `RunnerRecord.path`

**Files:**
- Modify: `webapp/src/lib/pipeline/runner/binding.ts`
- Modify: `webapp/src/lib/pipeline/runner/binding.test.ts`

- [ ] **Step 1: Update failing binding test fixtures**

In `binding.test.ts`, change import:

```ts
import type { Record } from '../runners/types';
```

Replace `runnerRecord`:

```ts
function runnerRecord(path: string): Record {
	return {
		name: path.split('/').at(-1) ?? 'runner',
		path,
		isOwned: true,
		isPublished: true,
		isOnline: true
	};
}
```

Change assertion:

```ts
expect(Runner.Binding.getExecutionRunnerPath(p)).toBe(r.runner.path);
```

Update imports to use `Binding` namespace:

```ts
import { Binding } from './index';
// or import * as Runner from './binding' and export Binding from index only in tests use:
import { Binding } from './index';
```

Simplest: test imports `{ Binding }` from `./index` after barrel exists.

- [ ] **Step 2: Run binding tests (expect fail on set signature)**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline/runner/binding.test.ts`

- [ ] **Step 3: Update `binding.ts`**

```ts
import type { Record } from '../runners/types';

export function set(pipeline: PipelinesResponse, runner: Pick<Record, 'path'>): void {
	try {
		pipelinesRunnersConfig[pipeline.id] = runner.path;
	} catch (error) {
		console.error('Failed to set pipeline runner:', error);
	}
}
```

Remove `MobileRunnerReference` import from deleted `utils.ts`.

- [ ] **Step 4: Run binding tests**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline/runner/binding.test.ts`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add webapp/src/lib/pipeline/runner/binding.ts webapp/src/lib/pipeline/runner/binding.test.ts
git commit -m "refactor(webapp): store runner path on Pipeline.Runner.Binding"
```

---

### Task 7: i18n `Runner_status_checking`

**Files:**
- Modify: `webapp/messages/en.json`
- Modify: `webapp/messages/da.json`, `de.json`, `es-es.json`, `fr.json`, `it.json`, `pt-BR.json`

- [ ] **Step 1: Add English string**

In `webapp/messages/en.json` next to `Runner_offline_run_disabled`:

```json
"Runner_status_checking": "Checking runner status…",
```

- [ ] **Step 2: Add same key to other locale files**

Use the English sentence as placeholder in `da.json`, `de.json`, `es-es.json`, `fr.json`, `it.json`, `pt-BR.json` (project pattern for new keys until translated).

- [ ] **Step 3: Regenerate Paraglide types if required**

Run: `cd webapp && bun run check` (or project i18n compile command from `package.json` scripts).

Expected: no missing-message errors for `Runner_status_checking`

- [ ] **Step 4: Commit**

```bash
git add webapp/messages/
git commit -m "i18n: add Runner_status_checking message"
```

---

### Task 8: App lifecycle + pipelines layout

**Files:**
- Modify: `webapp/src/hooks.client.ts`
- Modify: `webapp/src/routes/my/pipelines/+layout.svelte`

- [ ] **Step 1: Update hooks**

Replace:

```ts
Pipeline.Runners.store.init();
```

with:

```ts
Pipeline.Runner.Catalog.init();
```

Replace dispose:

```ts
Pipeline.Runner.Catalog.dispose();
```

- [ ] **Step 2: Update pipelines layout**

Replace `+layout.svelte` script:

```svelte
<script lang="ts">
	import { Pipeline } from '$lib';
	import { onMount } from 'svelte';

	let { children } = $props();

	onMount(() => Pipeline.Runner.Catalog.startLiveRefresh());
</script>

{@render children()}
```

- [ ] **Step 3: Commit**

```bash
git add webapp/src/hooks.client.ts webapp/src/routes/my/pipelines/+layout.svelte
git commit -m "feat(webapp): wire runner catalog init and live refresh"
```

---

### Task 9: `runner-select-input.svelte`

**Files:**
- Modify: `webapp/src/lib/pipeline/runner/runner-select-input.svelte`

- [ ] **Step 1: Rewrite script block**

Key changes:

```svelte
<script lang="ts">
	import { Pipeline } from '$lib';
	// ...existing partial imports...

	import type { Record } from '$lib/pipeline/runner';

	type Props = {
		onSelect?: (runner: Record) => void;
		selectedRunner?: string;
		// ...
	};

	const foundRunners = $derived.by(() => {
		Pipeline.Runner.Catalog.read();
		return Pipeline.Runner.Catalog.search(runnerSearch.text);
	});

	function searchRunner(_text: string) {
		// noop: Search still debounces text; list is derived
	}

	// Remove status import and both $effect blocks for probe/resync
</script>
```

Template updates:

- `{#each foundRunners as item (item.path)}`
- `selectedRunner === item.path`
- `const online = !Pipeline.Runner.Catalog.isReady() ? undefined : item.isOnline`
- `!item.isPublished` instead of `!item.published`
- `title={online === undefined ? m.Runner_status_checking() : ...}`

- [ ] **Step 2: Run Svelte check**

Run: `cd webapp && bun run check`

Expected: no errors in `runner-select-input.svelte`

- [ ] **Step 3: Commit**

```bash
git add webapp/src/lib/pipeline/runner/runner-select-input.svelte
git commit -m "refactor(webapp): derive runner select list from Catalog"
```

---

### Task 10: `runner-select-modal.svelte`

**Files:**
- Modify: `webapp/src/lib/pipeline/runner/runner-select-modal.svelte`

- [ ] **Step 1: Update imports and handlers**

```svelte
import { Pipeline } from '$lib';
import type { Record } from '$lib/pipeline/runner';

function handleSelect(runner: Record) {
	Pipeline.Runner.Binding.set(pipeline, runner);
	currentRunnerPath = runner.path;
	currentRunner = runner;
	// ...
}

$effect(() => {
	if (!currentRunnerPath) return;
	Pipeline.Runner.Catalog.read();
	currentRunner = Pipeline.Runner.Catalog.findByPath(currentRunnerPath);
});

$effect(() => {
	if (!open) return;
	void Pipeline.Runner.Catalog.refresh();
});
```

Remove `Runners.status.probe` effect and `../runners/utils` type import.

- [ ] **Step 2: Commit**

```bash
git add webapp/src/lib/pipeline/runner/runner-select-modal.svelte
git commit -m "refactor(webapp): refresh catalog when runner modal opens"
```

---

### Task 11: `run-now-button.svelte` offline + checking

**Files:**
- Modify: `webapp/src/lib/pipeline/runner/run-now-button.svelte`

- [ ] **Step 1: Replace status logic**

```svelte
import { Pipeline } from '$lib';
import * as Binding from './binding';
// or Pipeline.Runner.Binding via Pipeline

const executionPath = $derived(Binding.getExecutionRunnerPath(pipeline));
const runnerRequired = $derived(Binding.isRequired(pipeline));

const isChecking = $derived(
	runnerRequired &&
		!!executionPath &&
		!Pipeline.Runner.Catalog.isReady()
);

const isRunnerOffline = $derived(
	runnerRequired &&
		Pipeline.Runner.Catalog.isReady() &&
		executionPath !== undefined &&
		Pipeline.Runner.Catalog.findByPath(executionPath)?.isOnline === false
);

const runDisabled = $derived(isChecking || isRunnerOffline);
```

Remove `$effect` status probe entirely.

Update `handleRunNow` guard: `if (runDisabled) return;`

Button: `disabled={runDisabled}`

Tooltip wrapper: branch `{#if isChecking}` → `m.Runner_status_checking()`; `{:else if isRunnerOffline}` → existing offline message.

- [ ] **Step 2: Commit**

```bash
git add webapp/src/lib/pipeline/runner/run-now-button.svelte
git commit -m "feat(webapp): disable run now while catalog checking or offline"
```

---

### Task 12: Wallet action step

**Files:**
- Modify: `webapp/src/lib/pipeline-form/steps/wallet-action/index.ts`
- Modify: `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte.ts`
- Modify: `webapp/src/lib/pipeline-form/steps/wallet-action/wallet-action-step-form.svelte`

- [ ] **Step 1: Update serialize + deserialize in `index.ts`**

```ts
import { listSelector } from '$lib/pipeline/runners/query';
import type { Record } from '$lib/pipeline/runner';
```

Serialize:

```ts
if (runner !== GLOBAL_RUNNER) {
	_with.runner_id = runner.path;
}
```

Deserialize:

```ts
import { getLastPathSegment } from '../_partials/misc';

function fallbackRunner(path: string): Record {
	return {
		name: getLastPathSegment(path),
		path,
		isOwned: false,
		isPublished: false,
		isOnline: false
	};
}

// inside deserialize:
let runner: SelectedRunner = GLOBAL_RUNNER;
if (data.runner_id !== GLOBAL_RUNNER && data.runner_id) {
	const result = await listSelector().match({
		Resolved: (runners) =>
			runners.find((item) => item.path === data.runner_id) ??
			fallbackRunner(data.runner_id),
		Rejected: () => fallbackRunner(data.runner_id)
	});
	runner = result;
}
```

Update `SelectedRunner` type import in `wallet-action-step-form.svelte.ts`:

```ts
import type { Record } from '$lib/pipeline/runner';
export type SelectedRunner = Record | typeof GLOBAL_RUNNER;
```

Remove `foundRunners` state and `searchRunner` async method; keep `runnerSearch = new Search({ onSearch: () => {} })`.

- [ ] **Step 2: Derive list in `wallet-action-step-form.svelte`**

```svelte
<script lang="ts">
	import { Pipeline } from '$lib';
	// ...

	const foundRunners = $derived.by(() => {
		Pipeline.Runner.Catalog.read();
		return Pipeline.Runner.Catalog.search(form.runnerSearch.text);
	});
</script>
```

Update runner list snippet: `!item.isPublished` instead of `!item.published`.

- [ ] **Step 3: Commit**

```bash
git add webapp/src/lib/pipeline-form/steps/wallet-action/
git commit -m "refactor(webapp): wallet action step uses runner catalog"
```

---

### Task 13: Schedule form + AGENTS.md

**Files:**
- Modify: `webapp/src/routes/my/pipelines/_partials/schedule-pipeline-form.svelte`
- Modify: `AGENTS.md`

- [ ] **Step 1: Update schedule form**

```ts
import type { Record } from '$lib/pipeline/runner';

function onRunnerSelect(runner: Record) {
	formData.update((v) => ({
		...v,
		global_runner_id: runner.path
	}));
}
```

- [ ] **Step 2: Document API field names in AGENTS.md**

Under mobile runners section, add bullet:

```markdown
- `GET /api/mobile-runners` list items use `path`, `is_owned`, `is_published`, `is_online` (selector view omits `url`, `type`, `devices`, `queue_length`). Pipeline YAML still uses `runner_id` / `global_runner_id`.
```

- [ ] **Step 3: Commit**

```bash
git add webapp/src/routes/my/pipelines/_partials/schedule-pipeline-form.svelte AGENTS.md
git commit -m "docs: align schedule form and AGENTS with runner catalog API"
```

---

### Task 14: Final verification

**Files:** (none)

- [ ] **Step 1: Run Go handler tests**

Run: `go test -tags=unit ./pkg/internal/apis/handlers/ -run TestListMobileRunners -v`

Expected: PASS

- [ ] **Step 2: Run webapp unit tests for pipeline runner**

Run: `cd webapp && bun run test:unit -- src/lib/pipeline/runner/ src/lib/pipeline/runners/`

Expected: PASS

- [ ] **Step 3: Run webapp check/lint**

Run: `cd webapp && bun run check && bun run lint`

Expected: PASS (fix any import path or Paraglide issues)

- [ ] **Step 4: Grep for stale symbols**

Run: `rg "Pipeline\.Runners|runners/utils|status\.svelte|MobileRunnerListItem|runner_id:" webapp/src`

Expected: no hits except YAML `runner_id` in test YAML strings and serialize keys

- [ ] **Step 5: Manual smoke (optional but recommended)**

1. `make dev` or run API + webapp
2. Open `/my/pipelines` — runner dots update without status coordinator
3. Open runner modal — list refreshes
4. Offline runner not selectable; Run now disabled when offline
5. Before first load, Run now shows checking tooltip

---

## Self-review (spec coverage)

| Spec requirement | Task |
|------------------|------|
| Hard-break API rename | Task 1 |
| `query.ts` + zod | Task 2 |
| Sync `Catalog.search` | Task 3–4 |
| Coalesced `refresh` | Task 4 |
| `isReady()` semantics | Task 4 tests |
| Delete status subsystem | Task 5 |
| `Binding` + `Catalog` barrel | Task 5–6 |
| `Record` type barrel-only | Task 5 |
| 30s `startLiveRefresh` | Task 4, 8 |
| Modal `refresh` | Task 10 |
| No pre-run refresh | (no task — omit from run-now) |
| Offline gating | Task 9, 11 |
| Checking gating + i18n | Task 7, 11 |
| Wallet deserialize `listSelector` | Task 12 |
| Derived `foundRunners` | Task 9, 12 |
| Remove `Pipeline.Runners` | Task 5, 8 |
| Moderate tests | Tasks 2–4, 6 |
| One PR | All tasks one branch |
| AGENTS.md | Task 13 |

No placeholders remain; types consistently use `path` / `RunnerRecord` / exported `Record`.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-20-pipeline-runner-catalog.md`. Two execution options:

**1. Subagent-Driven (recommended)** — Fresh subagent per task, review between tasks, fast iteration.

**2. Inline Execution** — Run tasks in this session using superpowers:executing-plans with batch checkpoints.

Which approach?

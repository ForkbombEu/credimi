<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { CardDetailsComponentProps } from '$pipeline-form/steps';

	import { ChevronRightIcon } from '@lucide/svelte';

	import { m } from '@/i18n';

	import {
		getFCAFValidationTestIDs,
		type FCAFValidationFormData
	} from './fcaf-validation-step-form.svelte.js';
	import { groupSelectedTests } from './grouping.js';

	//

	let { data }: CardDetailsComponentProps<FCAFValidationFormData> = $props();

	const groups = $derived(groupSelectedTests(getFCAFValidationTestIDs(data.yaml)));
	let openGroups = $state<Record<string, boolean>>({});

	function isOpen(key: string): boolean {
		return openGroups[key] ?? false;
	}

	function toggleOpen(key: string) {
		openGroups[key] = !isOpen(key);
	}
</script>

{#if groups.length > 0}
	<div class="space-y-1.5">
		<p class="text-xs font-medium text-muted-foreground">{m.Tests()}:</p>
		<!-- svelte-ignore a11y_no_noninteractive_tabindex -->
		<div
			aria-label={m.Tests()}
			class="max-h-60 overflow-y-auto rounded-md border bg-muted/30"
			role="region"
			tabindex="0"
		>
			{#each groups as group (group.key)}
				<div>
					<button
						type="button"
						class="flex w-full items-center gap-1.5 px-2 py-1.5 text-left hover:bg-muted/50"
						onclick={() => toggleOpen(group.key)}
					>
						<ChevronRightIcon
							class="size-4 shrink-0 text-muted-foreground transition-transform {isOpen(
								group.key
							)
								? 'rotate-90'
								: ''}"
						/>
						<span class="text-xs font-medium">{group.label}</span>
						<span class="text-[10px] text-muted-foreground">{group.category}</span>
						<span class="ml-auto text-[10px] text-muted-foreground">
							{group.tests.length}
						</span>
					</button>
					{#if isOpen(group.key)}
						<ul class="space-y-0.5 pb-1 pl-8">
							{#each group.tests as test (test.id)}
								<li class="truncate pr-2 font-mono text-xs" title={test.id}>
									{test.id}
								</li>
							{/each}
						</ul>
					{/if}
				</div>
			{/each}
		</div>
	</div>
{/if}

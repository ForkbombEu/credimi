<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import {
		displayStandardName,
		SuiteBrowse,
		type SuiteFacetKey
	} from '$lib/conformance';
	import { entities, type EntityData } from '$lib/global/entities';

	import { m } from '@/i18n';

	//

	type Props = {
		browse: SuiteBrowse;
	};

	let { browse }: Props = $props();

	const facetFields: { key: SuiteFacetKey; label: string }[] = [
		{ key: 'standard', label: m.Standard() },
		{ key: 'component', label: m.Component() },
		{ key: 'provider', label: m.Provider() }
	];

	const selectClass =
		'border-input bg-background flex h-9 min-w-[8rem] rounded-md border px-3 py-1 text-sm shadow-xs outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px]';

	function entityForComponent(component: string): EntityData | undefined {
		switch (component.trim()) {
			case 'wallet':
				return entities.wallets;
			case 'issuer':
				return entities.credential_issuers;
			case 'verifier':
				return entities.verifiers;
			default:
				return undefined;
		}
	}

	function facetOptionLabel(key: SuiteFacetKey, value: string): string {
		switch (key) {
			case 'standard':
				return displayStandardName(value);
			case 'component': {
				const entity = entityForComponent(value);
				return entity?.labels.singular ?? value;
			}
			case 'provider':
				return browse.providerLabels[value] ?? value;
		}
	}
</script>

<div class="flex flex-wrap items-end gap-3">
	{#each facetFields as { key, label } (key)}
		<div class="flex flex-col gap-1">
			<label class="text-muted-foreground text-xs" for={`facet-${key}`}>{label}</label>
			<select id={`facet-${key}`} class={selectClass} bind:value={browse.filters[key]}>
				<option value="">{m.All()}</option>
				{#each browse.facetOptions[key] as value (value)}
					<option {value}>{facetOptionLabel(key, value)}</option>
				{/each}
			</select>
		</div>
	{/each}

	{#if browse.hasActiveFilters}
		<button
			type="button"
			class="text-primary text-sm underline-offset-4 hover:underline"
			onclick={browse.clearFilters}
		>
			{m.Clear_filters()}
		</button>
	{/if}
</div>

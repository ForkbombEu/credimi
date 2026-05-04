<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { entities, type EntitySlug } from '$lib/global/entities.js';

	import Icon from '@/components/ui-custom/icon.svelte';
	import T from '@/components/ui-custom/t.svelte';

	type Props = {
		entity: EntitySlug;
		description?: string;
	};

	let { entity, description }: Props = $props();

	const entityData = $derived(entities[entity]);
</script>

<div
	class="mx-auto mb-8 flex max-w-7xl flex-col items-center justify-center gap-4 px-4 pt-10 text-primary md:px-8"
>
	<div class="flex items-center gap-2">
		<Icon src={entityData.icon} class="size-10" />
		<T tag="h1">
			{entityData.labels.singular}
		</T>
	</div>
	<p class="max-w-[80ch] text-center text-balance">{description}</p>
</div>

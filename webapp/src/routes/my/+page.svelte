<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { baseSections, entities } from '$lib/global';

	import Icon from '@/components/ui-custom/icon.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { currentUser } from '@/pocketbase';

	import { setDashboardNavbar } from './+layout@.svelte';

	//

	setDashboardNavbar({
		title: m.Dashboard()
	});

	const sections = [...baseSections, entities.test_runs];
</script>

<div class="flex grow flex-col items-center justify-center gap-8">
	<div class="space-y-2 text-center">
		<T tag="h1">{m.Welcome_user({ username: $currentUser?.name ?? '' })}</T>
		<T tag="p">{m.Welcome_dashboard_sentence()}</T>
	</div>

	<div class="grid grid-cols-1 gap-4 sm:grid-cols-[1fr_1fr]">
		{#each sections as section (section)}
			<a
				href={`/my/${section.slug}`}
				class={[
					'bg-secondary block space-y-2 rounded-lg p-4',
					section.classes.text,
					'transition-transform hover:-translate-y-2 hover:ring-2'
				]}
			>
				<Icon size={24} src={section.icon} />
				<T tag="h3">{section.labels.plural}</T>
			</a>
		{/each}
	</div>
</div>

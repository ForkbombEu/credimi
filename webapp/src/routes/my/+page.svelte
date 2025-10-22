<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { appSections, testRunsSection } from '$lib/marketplace/sections';

	import type { IconComponent } from '@/components/types';

	import Icon from '@/components/ui-custom/icon.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import { m } from '@/i18n';
	import { currentUser } from '@/pocketbase';

	import { setDashboardNavbar } from './+layout@.svelte';

	//

	setDashboardNavbar({
		title: m.Dashboard()
	});
</script>

<div class="flex grow flex-col items-center justify-center gap-8">
	<div class="space-y-2 text-center">
		<T tag="h1">{m.Welcome_user({ username: $currentUser?.name ?? '' })}</T>
		<T tag="p">{m.Welcome_dashboard_sentence()}</T>
	</div>

	<div class="grid grid-cols-1 gap-4 sm:grid-cols-[1fr_1fr]">
		{#each Object.values(appSections) as section (section)}
			{@render cardLink(`/my/${section.id}`, section.icon, section.label, section.textClass)}
		{/each}
		{@render cardLink(
			testRunsSection.id,
			testRunsSection.icon,
			testRunsSection.label,
			testRunsSection.textClass,
			'sm:col-span-2'
		)}
	</div>
</div>

{#snippet cardLink(
	href: string,
	icon: IconComponent,
	label: string,
	textClass: string,
	className?: string
)}
	<a
		{href}
		class={[
			'bg-secondary block space-y-2 rounded-lg p-4',
			textClass,
			'transition-transform hover:-translate-y-2 hover:ring-2',
			className
		]}
	>
		<Icon size={24} src={icon} />
		<T tag="h3">{label}</T>
	</a>
{/snippet}

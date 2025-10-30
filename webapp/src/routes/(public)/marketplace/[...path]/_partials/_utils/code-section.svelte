<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { BundledLanguage } from 'shiki';
	import type { ComponentProps } from 'svelte';

	import CodeDisplay from '$lib/layout/codeDisplay.svelte';
	import { ChevronDown, ChevronUp } from 'lucide-svelte';

	import Button from '@/components/ui-custom/button.svelte';
	import { m } from '@/i18n';

	import PageSection from './page-section.svelte';

	//

	type Props = ComponentProps<typeof PageSection> & {
		code: string | undefined | null;
		language: BundledLanguage;
	};

	let { code, language, ...rest }: Props = $props();

	// 16 depends on max-h-[350px] & text-sm
	const canBeExpanded = $derived(code && code.split('\n').length > 16);

	let isExpanded = $state(false);
</script>

<PageSection {...rest} empty={!code}>
	<CodeDisplay
		content={code ?? ''}
		{language}
		theme="catppuccin-frappe"
		contentClass={[
			'text-sm transition-all',
			{
				'max-h-[350px]': !isExpanded
			}
		]}
	/>
	{#snippet right()}
		{#if canBeExpanded}
			<Button
				variant="ghost"
				size="sm"
				class="hover:bg-primary/10 text-primary"
				onclick={() => (isExpanded = !isExpanded)}
			>
				{#if isExpanded}
					<span>{m.Collapse()}</span>
					<ChevronUp class="h-4 w-4" />
				{:else}
					<span>{m.Expand()}</span>
					<ChevronDown class="h-4 w-4" />
				{/if}
			</Button>
		{/if}
	{/snippet}
</PageSection>

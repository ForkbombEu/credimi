<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { Snippet } from 'svelte';

	import type { GenericRecord } from '@/utils/types';

	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import * as Popover from '@/components/ui/popover';
	import { cn } from '@/components/ui/utils';

	import type { LanguageSelectTriggerSnippetProps } from './baseLanguageSelect.svelte';

	import BaseLanguageSelect from './baseLanguageSelect.svelte';
	import { setLocale } from './paraglide/runtime';

	type Props = {
		contentClass?: string;
		trigger?: Snippet<
			[LanguageSelectTriggerSnippetProps & { triggerAttributes: GenericRecord }]
		>;
		flagsOnly?: boolean;
	};

	const { contentClass = '', trigger: triggerSnippet, flagsOnly }: Props = $props();
</script>

<Popover.Root>
	<BaseLanguageSelect>
		{#snippet trigger(data)}
			<Popover.Trigger>
				{#snippet child({ props: triggerAttributes })}
					{#if triggerSnippet}
						{@render triggerSnippet({ ...data, triggerAttributes })}
					{:else}
						{@const { icon: LanguageIcon, text } = data}
						<Button variant="secondary" {...triggerAttributes}>
							<Icon src={LanguageIcon} />
							{text}
						</Button>
					{/if}
				{/snippet}
			</Popover.Trigger>
		{/snippet}

		{#snippet languages({ languages })}
			<Popover.Content
				class={cn('w-[--bits-popover-anchor-width] space-y-0.5 p-1', contentClass)}
			>
				{#each languages as { name, flag, isCurrent, tag }}
					<Button
						onclick={() => setLocale(tag)}
						variant={isCurrent ? 'secondary' : 'ghost'}
						class="flex w-full items-center justify-start gap-2"
						size="sm"
					>
						<span class="text-2xl">
							{flag}
						</span>
						{#if !flagsOnly}
							<span>
								{name}
							</span>
						{/if}
					</Button>
				{/each}
			</Popover.Content>
		{/snippet}
	</BaseLanguageSelect>
</Popover.Root>

<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { TestConfigFieldsFormComponent } from '$lib/start-checks-form/test-config-fields-form';
	import { TestConfigFormComponent } from '$lib/start-checks-form/test-config-form';
	import Button from '@/components/ui-custom/button.svelte';
	import Footer from '../_utils/footer.svelte';
	import SectionCard from '../_utils/section-card.svelte';
	import { ChecksConfigForm, type ChecksConfigFormProps } from './checks-configs-form.svelte.js';
	import { m } from '@/i18n';
	import * as Popover from '@/components/ui/popover';
	import { ArrowUp, Eye } from 'lucide-svelte';
	import type { IconComponent } from '@/components/types';
	import type { GenericRecord } from '@/utils/types';

	//

	const props: ChecksConfigFormProps = $props();
	const form = new ChecksConfigForm(props);

	const SHARED_FIELDS_ID = 'shared-fields';
</script>

<div class="space-y-4">
	{#if form.hasSharedFields}
		<SectionCard id={SHARED_FIELDS_ID} title={m.Shared_fields()}>
			<TestConfigFieldsFormComponent form={form.sharedFieldsForm} />
		</SectionCard>
	{/if}

	{#each Object.entries(form.checksForms) as [id, checkForm]}
		<SectionCard {id} title={id.replace('.json', '')}>
			<TestConfigFormComponent form={checkForm} />
		</SectionCard>
	{/each}
</div>

<Footer>
	{#snippet left()}
		{@const status = form.getCompletionStatus()}

		<div class="flex items-center gap-2">
			{#if form.hasSharedFields}
				<div class="flex items-center gap-2">
					<p>
						<span>{m.Shared_fields()}:</span>
						{#if status.sharedFields}
							<span class="font-bold text-green-600">
								{m.Completed()}
							</span>
						{:else}
							<span class="font-bold text-red-600">
								{m.count_missing({ count: status.missingSharedFieldsCount })}
							</span>
						{/if}
					</p>
					{#if !status.sharedFields}
						{@render SmallButton({
							href: `#${SHARED_FIELDS_ID}`,
							Icon: ArrowUp,
							text: m.Scroll()
						})}
					{/if}
				</div>
				<p>
					{' | '}
				</p>
			{/if}

			<div class="flex items-center gap-2">
				<p>
					<span>{m.Configs()}:</span>
					<span class="font-bold text-green-600">
						{m.count_valid({ count: status.validFormsCount })}
					</span>
					<span>{' / '}</span>
					<span class="font-bold text-red-600">
						{m.count_invalid({ count: status.invalidFormsCount })}
					</span>
				</p>
				{@render InvalidFormsPopover(status.invalidFormIds)}
			</div>
		</div>
	{/snippet}

	{#snippet right()}
		<Button>ao</Button>
	{/snippet}
</Footer>

{#snippet InvalidFormsPopover(ids: string[])}
	{#if ids.length}
		<Popover.Root>
			<Popover.Trigger>
				{#snippet child({ props })}
					{@render SmallButton({ Icon: Eye, text: m.View(), restProps: props })}
				{/snippet}
			</Popover.Trigger>
			<Popover.Content class="dark w-fit">
				<ul class="space-y-1 text-sm">
					{#each ids as testId}
						<li>
							<a class="underline hover:no-underline" href={`#${testId}`}>
								{testId}
							</a>
						</li>
					{/each}
				</ul>
			</Popover.Content>
		</Popover.Root>
	{/if}
{/snippet}

{#snippet SmallButton(props: {
	href?: string;
	Icon: IconComponent;
	text: string;
	restProps?: GenericRecord;
})}
	{@const { href, Icon, text, restProps = {} } = props}
	<Button {href} variant="outline" class="h-8 px-2 text-sm" {...restProps}>
		<Icon size={10} class="" />
		{text}
	</Button>
{/snippet}

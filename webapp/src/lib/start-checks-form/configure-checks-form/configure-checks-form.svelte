<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { CheckConfigFormEditorComponent } from './check-config-form-editor';
	import { CheckConfigEditorComponent } from './check-config-editor';
	import Button from '@/components/ui-custom/button.svelte';
	import Footer from '../_utils/footer.svelte';
	import SectionCard from '../_utils/section-card.svelte';
	import {
		ConfigureChecksForm,
		type ConfigureChecksFormProps
	} from './configure-checks-form.svelte.js';
	import { m } from '@/i18n';
	import * as Popover from '@/components/ui/popover';
	import { ArrowUp, Eye } from 'lucide-svelte';
	import type { IconComponent } from '@/components/types';
	import type { GenericRecord } from '@/utils/types';
	import { Separator } from '@/components/ui/separator';
	import { CustomCheckConfigEditorComponent } from './custom-check-config-editor';
	import LoadingDialog from '@/components/ui-custom/loadingDialog.svelte';
	import SmallErrorDisplay from '../_utils/small-error-display.svelte';
	import CopyButton from '@/components/ui-custom/copyButton.svelte';
	import { pb } from '@/pocketbase/index.js';
	import { PUBLIC_POCKETBASE_URL } from '$env/static/public';

	//

	const props: ConfigureChecksFormProps = $props();
	const form = new ConfigureChecksForm(props);

	const SHARED_FIELDS_ID = 'shared-fields';

	//

	function getCurlCommand() {
		const url = new URL(
			`api/compliance/${form.props.standardAndVersionPath}/save-variables-and-start`,
			PUBLIC_POCKETBASE_URL
		);
		return `curl '${url.toString()}' -X POST -H 'Authorization: ${pb.authStore.token}' --data-raw '${JSON.stringify(form.getFormData())}'`;
	}
</script>

<div class="space-y-4">
	{#if form.hasSharedFields}
		<SectionCard id={SHARED_FIELDS_ID} title={m.Shared_fields()}>
			<CheckConfigFormEditorComponent editor={form.sharedFieldsEditor} />
		</SectionCard>

		{@render SectionDivider(m.Configs())}
	{/if}

	{#each Object.entries(form.checkConfigEditors) as [id, checkConfigEditor]}
		<SectionCard {id} title={id.replace('.json', '')}>
			<CheckConfigEditorComponent editor={checkConfigEditor} />
		</SectionCard>
	{/each}

	{#if form.customCheckConfigEditors.length}
		{@render SectionDivider(m.Custom_checks())}
		{#each form.customCheckConfigEditors as customCheckConfigEditor}
			<SectionCard
				id={customCheckConfigEditor.props.customCheck.id}
				title={customCheckConfigEditor.props.customCheck.name}
			>
				<CustomCheckConfigEditorComponent editor={customCheckConfigEditor} />
			</SectionCard>
		{/each}
	{/if}

	{@render SectionDivider(m.Submit())}
</div>

<Footer class="!mt-4">
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
					{#if !form.isValid}
						<span class="font-bold text-red-600">
							{m.count_invalid({ count: status.invalidFormsCount })}
						</span>
					{:else}
						<span class="font-bold text-green-600">
							{status.validFormsCount}
						</span>
					{/if}
				</p>
				{@render InvalidFormsPopover(status.invalidFormsEntries)}
			</div>
		</div>
	{/snippet}

	{#snippet right()}
		{#if form.loadingError}
			<SmallErrorDisplay error={form.loadingError} />
		{/if}
		<CopyButton textToCopy={getCurlCommand()}>
			{m.Copy_as_curl()}
		</CopyButton>
		<Button disabled={!form.isValid} onclick={() => form.submit()}>{m.Start_checks()}</Button>
	{/snippet}
</Footer>

{#if form.isLoading}
	<LoadingDialog />
{/if}

{#snippet InvalidFormsPopover(entries: { id: string; text: string }[])}
	{#if entries.length}
		<Popover.Root>
			<Popover.Trigger>
				{#snippet child({ props })}
					{@render SmallButton({ Icon: Eye, text: m.View(), restProps: props })}
				{/snippet}
			</Popover.Trigger>
			<Popover.Content class="dark w-fit">
				<ul class="space-y-1 text-sm">
					{#each entries as { id, text }}
						<li>
							<a class="underline hover:no-underline" href="#{id}">
								{text}
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

{#snippet SectionDivider(text: string)}
	<div class="flex items-center gap-3 py-1">
		<Separator class="!w-auto grow" />
		<p class="text-muted-foreground text-sm">{text}</p>
		<Separator class="!w-auto grow" />
	</div>
{/snippet}

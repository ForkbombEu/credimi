<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import type { Snippet } from 'svelte';
	import type { SuperForm } from 'sveltekit-superforms/client';

	import type { GenericRecord } from '@/utils/types';

	import { setupDerivedContext } from '@/utils/svelte-context';

	//

	export type FormContext<Data extends GenericRecord = GenericRecord> = {
		form: SuperForm<Data>;
		hideRequiredIndicator: boolean;
	};

	const { getDerivedContext: getFormContext, setDerivedContext: setFormContext } =
		setupDerivedContext<FormContext>('form');

	export { getFormContext };

	//

	export type FormSnippets = {
		submitButtonContent?: Snippet;
		submitButton?: Snippet<[{ SubmitButton: typeof SubmitButton }]>;
		loadingStateContent?: Snippet;
		loadingState?: Snippet<[{ isLoading: boolean }]>;
	};
</script>

<script lang="ts" generics="T extends GenericRecord">
	import type { HTMLFormAttributes } from 'svelte/elements';

	import LoadingDialog from '@/components/ui-custom/loadingDialog.svelte';
	import { m } from '@/i18n';

	import FormError from './components/formError.svelte';
	import SubmitButton from './components/submitButton.svelte';

	//

	type Props = {
		form: SuperForm<T, any>;
		hide?: ('error' | 'submit_button' | 'loading_state')[];
		hideRequiredIndicator?: boolean;
		children?: Snippet<
			[{ SubmitButton: typeof SubmitButton; FormError: typeof FormError; isLoading: boolean }]
		>;
	} & FormSnippets &
		HTMLFormAttributes;

	const {
		form,
		hide = [],
		class: className = 'space-y-8',
		hideRequiredIndicator = false,
		children,
		submitButton,
		submitButtonContent,
		loadingState,
		loadingStateContent,
		...rest
	}: Props = $props();

	//

	type Enctype = HTMLFormAttributes['enctype'];
	const enctype: Enctype = $derived(
		form.options.dataType == 'form' ? 'multipart/form-data' : undefined
	);

	const { enhance, delayed } = $derived(form);

	//

	const context = $derived<FormContext<T>>({
		form,
		hideRequiredIndicator
	});

	setFormContext(() => context);
</script>

<form {...rest} class={className} method="post" use:enhance {enctype}>
	{@render children?.({ FormError, SubmitButton, isLoading: $delayed })}

	{#if !hide.includes('error')}
		<FormError></FormError>
	{/if}

	{#if submitButton}
		{@render submitButton({ SubmitButton })}
	{:else if !hide.includes('submit_button')}
		<div class="flex justify-end">
			<SubmitButton>
				{#if submitButtonContent}
					{@render submitButtonContent()}
				{:else}
					{m.Submit()}
				{/if}
			</SubmitButton>
		</div>
	{/if}
</form>

{#if loadingState}
	{@render loadingState({ isLoading: $delayed })}
{:else if !hide.includes('loading_state')}
	<LoadingDialog loading={$delayed}>
		{#if loadingStateContent}
			{@render loadingStateContent()}
		{:else}
			{m.Please_wait()}
		{/if}
	</LoadingDialog>
{/if}

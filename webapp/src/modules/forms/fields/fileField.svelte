<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" generics="Data extends GenericRecord">
	import type { GenericRecord } from '@/utils/types';
	import type { SuperForm, FormPath } from 'sveltekit-superforms';
	import { fieldProxy } from 'sveltekit-superforms';
	import * as Form from '@/components/ui/form';
	import FieldWrapper from './parts/fieldWrapper.svelte';
	import FileManager from '@/components/ui-custom/fileManager.svelte';
	import Input from '@/components/ui/input/input.svelte';
	import type { FieldOptions } from './types';
	import type { Writable } from 'svelte/store';
	import type { ComponentProps, Snippet } from 'svelte';
	import { createFilesValidator } from './fileField';
	import { Button } from '@/components/ui/button';

	//

	type Props = {
		form: SuperForm<Data>;
		name: FormPath<Data>;
		variant?: ComponentProps<typeof Button>['variant'];
		class?: string;
		options?: Partial<
			FieldOptions &
				Omit<ComponentProps<typeof Input>, 'type' | 'value'> & { showFilesList?: boolean }
		>;
		children?: Snippet;
	};

	const {
		form,
		name,
		class: className,
		variant = 'default',
		options = {},
		children
	}: Props = $props();

	const multiple = $derived(options.multiple ?? false);
	const valueProxy = $derived(fieldProxy(form, name) as Writable<File | File[]>);

	const validator = $derived(
		createFilesValidator(form as SuperForm<GenericRecord>, name, multiple)
	);

	let fileInput: HTMLInputElement;
</script>

<Form.Field {form} {name}>
	<FieldWrapper field={name} {options}>
		{#snippet children({ props })}
			<FileManager
				bind:data={$valueProxy}
				{validator}
				{multiple}
				showFilesList={options.showFilesList}
			>
				{#snippet children({ addFiles })}
					<Button
						type="button"
						{variant}
						onclick={() => fileInput.click()}
						class={['w-full', className]}
					>
						{#if children}
							{@render children({ addFiles })}
						{:else}
							{options.placeholder}
						{/if}
					</Button>
					<input
						bind:this={fileInput}
						{...props}
						type="file"
						{multiple}
						accept={options.accept}
						class="hidden"
						onchange={(e) => {
							const fileList = e.currentTarget.files;
							if (fileList) addFiles([...fileList]);
							e.currentTarget.value = '';
						}}
					/>
					<!-- e.currentTarget.value = '' is needed to clear the file input -->
				{/snippet}
			</FileManager>
		{/snippet}
	</FieldWrapper>
</Form.Field>

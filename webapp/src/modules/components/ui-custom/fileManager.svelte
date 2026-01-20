<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	type MaybePromise<T> = T | Promise<T>;

	export type RejectedFile = { file: File; reasons: string[] };
	export type FileManagerValidator = (
		newFiles: File[]
	) => MaybePromise<{ acceptedFiles: File[]; rejectedFiles: RejectedFile[] }>;
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';

	import { X } from '@lucide/svelte';
	import _ from 'lodash';

	import Button from '@/components/ui-custom/button.svelte';
	import Badge from '@/components/ui/badge/badge.svelte';
	import { m } from '@/i18n';

	import ArrayOrItemManager from './arrayOrItemManager.svelte';
	import List from './list.svelte';
	import ListHeader from './listHeader.svelte';
	import ListItem from './listItem.svelte';
	import T from './t.svelte';

	//

	type Props = {
		data?: File | File[];
		validator?: FileManagerValidator;
		multiple?: boolean;
		showFilesList?: boolean;
		children?: Snippet<[{ addFiles: (newFiles: File[]) => void }]>;
	};

	let {
		data = $bindable(),
		validator,
		multiple = false,
		showFilesList = true,
		children: child
	}: Props = $props();

	/* File upload handling */

	let rejectedFiles: RejectedFile[] = $state([]);

	const addFiles = $derived(async function (newFiles: File[]) {
		if (validator) {
			const result = await validator(newFiles);
			newFiles = result.acceptedFiles;
			rejectedFiles = result.rejectedFiles;
		}
		updateData(newFiles);
	});

	const updateData = $derived(function (newFiles: File[]) {
		if (multiple) {
			if (Array.isArray(data)) data = [...data, ...newFiles];
			else if (data === undefined) data = [...newFiles];
		} else {
			data = newFiles[0];
		}
	});

	function clearRejectedFiles() {
		rejectedFiles = [];
	}

	/* Checking if a file has been added after the component has been initalized */

	const oldFiles = data;

	function isNewFile(file: File): boolean {
		if (Array.isArray(oldFiles)) {
			const search = oldFiles.find((f) => _.isEqual(f, file));
			return !search;
		} else {
			return !_.isEqual(oldFiles, file);
		}
	}

	// 

	const hasFiles= $derived(((Array.isArray(data) && data.length > 0) || (!multiple && Boolean(data))))
</script>

<div class={{"space-y-2": (showFilesList && hasFiles) || rejectedFiles.length > 0}}>
	{@render child?.({ addFiles })}

	{#if showFilesList && hasFiles}
		<List>
			<ListHeader label={m.Files()} />

			<ArrayOrItemManager bind:data>
				{#snippet children({ item, removeItem })}
					{@const isNew = isNewFile(item)}
					<ListItem onclick={removeItem}>
						<div class="flex items-center gap-2">
							<T tag="p">
								{item.name}
							</T>
							{#if isNew}
								<Badge variant="secondary">{m.New()}</Badge>
							{/if}
						</div>
					</ListItem>
				{/snippet}
			</ArrayOrItemManager>
		</List>
	{/if}

	{#if rejectedFiles.length > 0}
		<List>
			<ListHeader class="flex items-center justify-between">
				<T tag="small">{m.Rejected_files()}</T>
				<Button variant="link" class="h-6 text-xs" onclick={clearRejectedFiles}>
					<X size={14} class="mr-1" />
					{m.Clear()}
				</Button>
			</ListHeader>
			{#each rejectedFiles as rejection (rejection.file)}
				<div class="px-3 py-1 leading-tight text-red-600">
					<T tag="small" class="font-normal!">{rejection.file.name}</T>
					<ul>
						{#each rejection.reasons as reason (reason)}
							<li>
								<T tag="small">{reason}</T>
							</li>
						{/each}
					</ul>
				</div>
			{/each}
		</List>
	{/if}
</div>

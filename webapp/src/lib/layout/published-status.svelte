<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	type Props = {
		item: { published: boolean } | { public: boolean };
		class?: string;
		size?: 'sm' | 'md';
	};

	let { item, class: className, size = 'md' }: Props = $props();

	console.log(item);
	const published = $derived(
		('published' in item && item.published) || ('public' in item && item.public)
	);
</script>

<span
	class={[
		'inline-block shrink-0 rounded-full',
		{
			'border-emerald-500 bg-emerald-400': published,
			'bg-inherit/50 border': !published,
			'size-1.5': size === 'sm',
			'size-2': size === 'md'
		},
		className
	]}
></span>

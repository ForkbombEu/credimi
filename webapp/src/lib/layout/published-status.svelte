<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	type Props = {
		item: { published: boolean } | { public: boolean };
		class?: string;
		size?: 'sm' | 'md';
		privateDisplay?: 'muted' | 'warning';
	};

	let { item, class: className, size = 'md', privateDisplay = 'muted' }: Props = $props();

	const published = $derived(
		('published' in item && item.published) || ('public' in item && item.public)
	);
</script>

<span
	class={[
		'inline-block shrink-0 rounded-full border',
		{
			'border-emerald-500 bg-emerald-400': published,
			'border-border bg-border/20': !published && privateDisplay === 'muted',
			'border-yellow-500 bg-yellow-400': !published && privateDisplay === 'warning',
			'size-1.5': size === 'sm',
			'size-2': size === 'md'
		},
		className
	]}
></span>

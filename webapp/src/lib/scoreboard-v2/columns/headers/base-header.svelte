<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { HeaderAlign } from './alignment';

	type Props = {
		header: string;
		align?: HeaderAlign;
	};

	let { header, align = 'left' }: Props = $props();
</script>

<div
	class={[
		'flex items-center gap-2 text-[1rem] text-primary',
		{
			'justify-start': align === 'left',
			'justify-center': align === 'center',
			'justify-end': align === 'right'
		}
	]}
>
	<span>{header}</span>
</div>

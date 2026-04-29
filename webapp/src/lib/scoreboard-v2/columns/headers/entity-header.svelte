<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ComponentProps } from 'svelte';

	import { EntityTag } from '$lib/global';

	import type { HeaderAlign } from './alignment';

	//

	type Props = ComponentProps<typeof EntityTag> & {
		align?: HeaderAlign;
	};

	let { align = 'left', ...props }: Props = $props();
</script>

<div
	class={[
		'flex items-center',
		{
			'justify-start': align === 'left',
			'justify-center': align === 'center',
			'justify-end': align === 'right'
		}
	]}
>
	<EntityTag {...props} />
</div>

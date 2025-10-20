<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { IconComponent } from '@/components/types';

	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';

	//

	type Props = {
		text: string;
		icon: IconComponent;
	};

	let { text, icon, ...rest }: Props = $props();
</script>

<Button
	variant="link"
	size="sm"
	class="h-8 gap-1 px-2 text-blue-600 hover:cursor-pointer hover:bg-blue-50 hover:no-underline"
	{...rest}
>
	<Icon src={icon} />
	{text}
</Button>

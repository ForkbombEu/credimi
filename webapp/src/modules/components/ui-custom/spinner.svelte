<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ComponentProps } from 'svelte';

	import { Moon } from 'svelte-loading-spinners';

	type Props = ComponentProps<Moon>;

	const { size = 40, color = 'black', ...rest }: Props = $props();
</script>

<Moon {size} {color} {...rest} />

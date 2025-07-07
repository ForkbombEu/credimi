<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { Button, type ButtonProps } from '@/components/ui/button';
	import { localizeHref } from '@/i18n';
	import { cn } from '@/components/ui/utils.js';

	let { class: className, href, children, ...restProps }: ButtonProps = $props();
</script>

<Button
	href={href ? localizeHref(href) : undefined}
	class={cn(className, 'cursor-pointer')}
	{...restProps}
>
	{@render children?.()}
</Button>

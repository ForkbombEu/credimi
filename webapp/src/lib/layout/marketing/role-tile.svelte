<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import type { EntityData } from '$lib/global/entities';

	/** Icon and color classes of a Hub entity, or of a marketing-only role styled the same way. */
	export type RoleStyle = Pick<EntityData, 'icon' | 'classes'>;
</script>

<script lang="ts">
	const { role, size = 'lg' }: { role: RoleStyle; size?: 'sm' | 'lg' } = $props();

	const Icon = $derived(role.icon);
</script>

<span
	class={[
		'grid shrink-0 place-items-center rounded-xl',
		role.classes.bg,
		role.classes.text,
		size === 'lg' ? 'size-12' : 'size-10'
	]}
>
	<Icon class={size === 'lg' ? 'size-6' : 'size-5'} aria-hidden="true" />
</span>

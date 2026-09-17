<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts" module>
	import { renderComponent } from '@/components/ui/data-table';
	import { m } from '@/i18n';

	import * as Column from '../column';
	import EntityHeader from './headers/entity-header.svelte';

	//

	export const column = Column.define({
		fn: (row) => row.expanded_data?.mobile_devices ?? [],
		id: 'devices',
		header: renderComponent(EntityHeader, {
			label: m.Devices()
		}),
		sortField: 'mobile_devices.name'
	});
</script>

<script lang="ts">
	import { AppleIcon, SmartphoneIcon } from '@lucide/svelte';
	import { SvelteMap } from 'svelte/reactivity';

	import type { IconComponent } from '@/components/types';

	import Tooltip from '@/components/ui-custom/tooltip.svelte';

	import * as EntityDisplay from '../entity-display';

	let { value }: Column.Props<typeof column> = $props();

	type Device = (typeof value)[number];
	type Platform = { key: string; label: string; icon: IconComponent; devices: Device[] };

	const platforms = $derived.by(() => {
		const groups = new SvelteMap<string, Platform>();
		for (const device of value) {
			const isIos = Boolean(device.type?.startsWith('ios'));
			const key = isIos ? 'ios' : 'android';
			const platform: Platform = groups.get(key) ?? {
				key,
				label: isIos ? 'iOS' : 'Android',
				icon: isIos ? AppleIcon : SmartphoneIcon,
				devices: []
			};
			platform.devices.push(device);
			groups.set(key, platform);
		}
		return [...groups.values()];
	});
</script>

{#if platforms.length === 0}
	<EntityDisplay.Na />
{:else}
	<div class="flex items-center gap-1.5">
		{#each platforms as platform (platform.key)}
			<Tooltip>
				<span
					class="inline-flex min-w-5 items-center justify-center gap-0.5 rounded-full bg-secondary px-1.5 py-0.5 text-[10px] leading-none font-semibold text-secondary-foreground tabular-nums"
				>
					<platform.icon class="size-3" />
					{platform.devices.length}
				</span>
				{#snippet content()}
					<ul class="flex flex-col gap-1">
						{#each platform.devices as device (device)}
							<li class="text-xs">{device.name.trim()}</li>
						{/each}
					</ul>
				{/snippet}
			</Tooltip>
		{/each}
	</div>
{/if}

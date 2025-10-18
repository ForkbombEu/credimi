<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ComponentProps, Snippet } from 'svelte';

	import { page } from '$app/state';
	import UserNav from '$lib/layout/userNav.svelte';

	import { AppLogo } from '@/brand';
	import Icon from '@/components/ui-custom/icon.svelte';
	import * as Sidebar from '@/components/ui/sidebar/index.js';

	import type { SidebarGroup } from './sidebar';

	//

	type Props = ComponentProps<typeof Sidebar.Root> & {
		data: SidebarGroup[];
	};

	let { ref = $bindable(null), data, class: className, ...restProps }: Props = $props();
</script>

<Sidebar.Root {...restProps} class={[className]} bind:ref>
	<Sidebar.Header
		class="border-b-primary/50 flex flex-row items-center justify-between border-b px-4 pb-[7px]"
	>
		<div class="flex size-8 items-center justify-center overflow-hidden">
			<AppLogo />
		</div>
		<UserNav />
	</Sidebar.Header>
	<Sidebar.Content class="gap-0">
		{#each data as group (group.title)}
			<Sidebar.Group>
				{#if group.title}
					<Sidebar.GroupLabel>{group.title}</Sidebar.GroupLabel>
				{/if}
				<Sidebar.GroupContent>
					<Sidebar.Menu>
						{#each group.items as item (item)}
							{#if typeof item === 'function'}
								{@render (item as Snippet)()}
							{:else}
								{@const isActive = page.url.pathname.endsWith(item.url)}
								<Sidebar.MenuItem>
									<Sidebar.MenuButton {isActive}>
										{#snippet child({ props })}
											<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
											<a href={item.url} {...props}>
												{#if item.icon}
													<Icon src={item.icon} />
												{/if}
												<span class="text-nowrap">
													{item.title}
												</span>
											</a>
										{/snippet}
									</Sidebar.MenuButton>
								</Sidebar.MenuItem>
							{/if}
						{/each}
					</Sidebar.Menu>
				</Sidebar.GroupContent>
			</Sidebar.Group>
		{/each}
	</Sidebar.Content>
	<Sidebar.Rail />
</Sidebar.Root>

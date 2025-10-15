<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import type { ComponentProps } from 'svelte';

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
	<Sidebar.Header>
		<!-- <VersionSwitcher versions={data.versions} defaultVersion={data.versions[0]} /> -->
		<!-- <SearchForm /> -->
	</Sidebar.Header>
	<Sidebar.Content>
		<!-- We create a Sidebar.Group for each parent. -->
		{#each data as group (group.title)}
			<Sidebar.Group>
				<Sidebar.GroupLabel>{group.title}</Sidebar.GroupLabel>
				<Sidebar.GroupContent>
					<Sidebar.Menu>
						{#each group.items as item (item.title)}
							<Sidebar.MenuItem>
								<Sidebar.MenuButton isActive={item.isActive}>
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
						{/each}
					</Sidebar.Menu>
				</Sidebar.GroupContent>
			</Sidebar.Group>
		{/each}
	</Sidebar.Content>
	<Sidebar.Rail />
</Sidebar.Root>

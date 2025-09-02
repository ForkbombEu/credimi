<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import { version } from '$app/environment';
	import { page } from '$app/state';
	import {
		Flame,
		Home,
		Inbox,
		CircleHelp,
		SquareStack,
		ChevronUp,
		LogOut,
		File,
		User,
		SquareArrowOutUpRight
	} from 'lucide-svelte';

	import { AppLogo } from '@/brand';
	import { appName } from '@/brand';
	import SidebarGroup from '@/components/layout/sidebar/sidebarGroup.svelte';
	import SidebarItemIcon from '@/components/layout/sidebar/sidebarItemIcon.svelte';
	import SidebarLink from '@/components/layout/sidebar/sidebarLink.svelte';
	import Button from '@/components/ui-custom/button.svelte';
	import Icon from '@/components/ui-custom/icon.svelte';
	import T from '@/components/ui-custom/t.svelte';
	import UserAvatar from '@/components/ui-custom/userAvatar.svelte';
	import * as Popover from '@/components/ui/popover';
	import * as Sidebar from '@/components/ui/sidebar/index.js';
	import { getUserDidUrl } from '@/did';
	import { featureFlags } from '@/features';
	import { m } from '@/i18n';
	import LanguageSelect from '@/i18n/languageSelect.svelte';
	import { createOrganizationLinks, type OrgRole } from '@/organizations';
	import { OrganizationAvatar } from '@/organizations/components';
	import { currentUser, pb } from '@/pocketbase';
	import { PocketbaseQueryAgent } from '@/pocketbase/query';
	import { getUserDisplayName } from '@/pocketbase/utils';

	//

	const authorizationsQuery = new PocketbaseQueryAgent({
		collection: 'orgAuthorizations',
		filter: `user = "${pb.authStore.record!.id}"`,
		expand: ['organization', 'role']
	});

	const sidebarState = Sidebar.useSidebar();

	function closeMobile() {
		sidebarState.setOpenMobile(false);
	}
</script>

<Sidebar.Root>
	<Sidebar.Header class="flex flex-row items-center justify-between border-b">
		<AppLogo />
		{#if sidebarState.isMobile}
			<Sidebar.Trigger variant="outline" class="size-9" />
		{/if}
	</Sidebar.Header>

	<Sidebar.Content class="gap-0">
		<Sidebar.Group>
			<Sidebar.GroupContent>
				<Sidebar.Menu>
					<SidebarLink href="/my" title={m.Home()} icon={Home} />
					<SidebarLink
						href="/my/notifications"
						title={m.notifications()}
						icon={Inbox}
						disabled
					/>
				</Sidebar.Menu>
			</Sidebar.GroupContent>
		</Sidebar.Group>

		{#if $featureFlags.ORGANIZATIONS}
			<Sidebar.Group>
				<Sidebar.GroupLabel>{m.organizations()}</Sidebar.GroupLabel>
				<Sidebar.GroupContent class="space-y-1">
					<Sidebar.Menu>
						<SidebarLink
							href="/my/organizations"
							title={m.My_organizations()}
							icon={SquareStack}
						/>
					</Sidebar.Menu>

					{#await authorizationsQuery.getFullList() then authorizations}
						{#each authorizations as authorization}
							{@const organization = authorization.expand?.organization!}
							{@const role = authorization.expand?.role!}
							{@const active = page.url.pathname.includes(
								`/my/organizations/${organization.id}`
							)}
							<SidebarGroup {active}>
								{#snippet trigger()}
									<SidebarItemIcon>
										<OrganizationAvatar {organization} class="size-5 border" />
									</SidebarItemIcon>
									<span>{organization.name}</span>
								{/snippet}

								{#snippet content()}
									{@const links = createOrganizationLinks(
										organization.id,
										role.name as OrgRole
									)}

									{#each links as link}
										<SidebarLink {...link} sub />
									{/each}
								{/snippet}
							</SidebarGroup>
						{/each}
					{/await}
				</Sidebar.GroupContent>
			</Sidebar.Group>
		{/if}
	</Sidebar.Content>

	<Sidebar.Footer class="gap-0 border-t p-0">
		<Sidebar.Menu class="p-2">
			<SidebarLink href="" title={m.Help()} icon={CircleHelp} disabled />

			<Sidebar.MenuItem>
				<LanguageSelect>
					{#snippet trigger({ triggerAttributes, text, icon: LanguagesIcon })}
						<Sidebar.MenuButton
							{...triggerAttributes}
							class="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
						>
							<Icon src={LanguagesIcon} />
							<span>{text}</span>
							<ChevronUp class="ml-auto" />
						</Sidebar.MenuButton>
					{/snippet}
				</LanguageSelect>
			</Sidebar.MenuItem>

			{#if $currentUser}
				<Sidebar.MenuItem>
					<Popover.Root>
						<Popover.Trigger>
							{#snippet child({ props })}
								<Sidebar.MenuButton
									{...props}
									class="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
								>
									<SidebarItemIcon>
										<UserAvatar user={$currentUser} class="size-6" />
									</SidebarItemIcon>
									<span>{getUserDisplayName($currentUser!)}</span>

									<ChevronUp class="ml-auto" />
								</Sidebar.MenuButton>
							{/snippet}
						</Popover.Trigger>

						<Popover.Content
							side="top"
							class="w-[--bits-popover-anchor-width] space-y-0.5 p-1"
						>
							<Button
								variant="ghost"
								size="sm"
								data-sveltekit-preload-data="off"
								class="flex w-full items-center justify-start gap-2"
								href="/my/profile"
								onclick={closeMobile}
							>
								<Icon src={User} />
								{m.My_profile()}
							</Button>

							{#if $featureFlags.DID}
								{#await getUserDidUrl() then href}
									{#if href}
										<Button
											variant="ghost"
											size="sm"
											{href}
											class="flex w-full items-center justify-start gap-2"
											target="_blank"
										>
											<Icon src={File} />
											{m.my_DID()}
											<Icon src={SquareArrowOutUpRight} />
										</Button>
									{/if}
								{/await}
							{/if}

							<Button
								variant="ghost"
								size="sm"
								data-sveltekit-preload-data="off"
								class="flex w-full items-center justify-start gap-2"
								disabled
							>
								<Icon src={Flame} />
								{m.Go_Pro()}
							</Button>

							<Button
								variant="ghost"
								size="sm"
								href="/logout"
								data-sveltekit-preload-data="off"
								class="flex w-full items-center justify-start gap-2"
								disabled
							>
								<Icon src={LogOut} />
								{m.Sign_out()}
							</Button>
						</Popover.Content>
					</Popover.Root>
				</Sidebar.MenuItem>
			{/if}
		</Sidebar.Menu>

		<div class="flex w-full overflow-hidden border-t p-2">
			<T
				tag="small"
				class="text-wrap font-mono text-xs leading-normal text-secondary-foreground/50"
			>
				{appName} – Version {version}
			</T>
		</div>
	</Sidebar.Footer>
</Sidebar.Root>
<!-- 
{#snippet DropdownMenuLink({
	title,
	icon,
	href,
	options,
	...rest
}: {
	title: string;
	href: string;
	icon?: IconComponent;
	options?: DropdownMenuItemProps;
} & HTMLAnchorAttributes)}
	<DropdownMenu.Item {...options} class="{options?.class} hover:cursor-pointer">
		{#snippet child({ props })}
			<a {href} {...props} {...rest}>
				{#if icon}
					<Icon src={icon} />
				{/if}
				<span>{title}</span>
			</a>
		{/snippet}
	</DropdownMenu.Item>
{/snippet} -->

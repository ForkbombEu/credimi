<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import Check from 'lucide-svelte/icons/check';
	import ChevronsUpDown from 'lucide-svelte/icons/chevrons-up-down';
	import { tick } from 'svelte';

	import type { OrganizationsResponse } from '@/pocketbase/types';

	import * as Avatar from '@/components/ui/avatar/index.js';
	import { buttonVariants } from '@/components/ui/button/index.js';
	import * as Command from '@/components/ui/command/index.js';
	import * as Dialog from '@/components/ui/dialog/index.js';
	import * as Popover from '@/components/ui/popover/index.js';
	import { cn } from '@/components/ui/utils.js';
	import { m } from '@/i18n';
	import { goto } from '@/i18n';
	import OrganizationAvatar from '@/organizations/components/organizationAvatar.svelte';

	//

	type Props = {
		class?: string;
		organizations: OrganizationsResponse[];
	};

	let { class: className = '', organizations = [] }: Props = $props();

	//

	const groups = [
		{
			label: 'Personal Account',
			teams: [
				{
					label: 'Alicia Koch',
					value: 'personal'
				}
			]
		},
		{
			label: 'Teams',
			teams: [
				{
					label: 'Acme Inc.',
					value: 'acme-inc'
				},
				{
					label: 'Monsters Inc.',
					value: 'monsters'
				}
			]
		}
	];

	type Team = (typeof groups)[number]['teams'][number];

	let open = $state(false);
	let showTeamDialog = $state(false);
	let selectedTeam: Team = $state(groups[0].teams[0]);
	let triggerId = 'team-switcher-trigger';

	function closeAndRefocusTrigger(triggerId: string) {
		open = false;
		tick().then(() => document.getElementById(triggerId)?.focus());
	}
</script>

<Dialog.Root bind:open={showTeamDialog}>
	<Popover.Root bind:open>
		<!--  -->

		<Popover.Trigger
			id={triggerId}
			role="combobox"
			aria-expanded={open}
			aria-label="Select a team"
			class={cn(
				buttonVariants({ variant: 'outline', class: 'w-[200px] justify-between' }),
				className
			)}
		>
			<Avatar.Root class="mr-2 size-5">
				<Avatar.Image
					src="https://avatar.vercel.sh/${selectedTeam.value}.png"
					alt={selectedTeam.label}
					class="grayscale"
				/>
				<Avatar.Fallback>SC</Avatar.Fallback>
			</Avatar.Root>
			{selectedTeam.label}
			<ChevronsUpDown class="ml-auto size-4 shrink-0 opacity-50" />
		</Popover.Trigger>

		<Popover.Content class="w-[200px] p-0">
			<Command.Root>
				<Command.Input placeholder="Search team..." />

				<Command.List>
					<Command.Empty>No team found.</Command.Empty>

					<Command.Group heading={m.organizations()}>
						{#each organizations as organization}
							<Command.Item
								onSelect={() => {
									goto(`/my/organizations/${organization.id}`);
									closeAndRefocusTrigger(triggerId);
								}}
								value={organization.name}
								class="text-sm"
							>
								<OrganizationAvatar {organization} />
								{organization.name}
								<Check
									class={cn(
										'ml-auto size-4',
										selectedTeam.value !== organization.id && 'text-transparent'
									)}
								/>
							</Command.Item>
						{/each}
					</Command.Group>
					<!-- {#each groups as group}
						<Command.Group heading={group.label}>
							{#each group.teams as team}
								<Command.Item
									onSelect={() => {
										selectedTeam = team;
										closeAndRefocusTrigger(triggerId);
									}}
									value={team.label}
									class="text-sm"
								>
									<Avatar.Root class="mr-2 size-5">
										<Avatar.Image
											src="https://avatar.vercel.sh/${team.value}.png"
											alt={team.label}
											class="grayscale"
										/>
										<Avatar.Fallback>SC</Avatar.Fallback>
									</Avatar.Root>
									{team.label}
									<Check
										class={cn(
											'ml-auto size-4',
											selectedTeam.value !== team.value && 'text-transparent'
										)}
									/>
								</Command.Item>
							{/each}
						</Command.Group>
					{/each} -->
				</Command.List>
			</Command.Root>
		</Popover.Content>
	</Popover.Root>
</Dialog.Root>

<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import DashboardCard from '$lib/layout/dashboard-card.svelte';
	import { runWithLoading } from '$lib/utils';
	import { CogIcon, Pencil, PlayIcon, Plus } from 'lucide-svelte';

	import type { PipelinesResponse } from '@/pocketbase/types';

	import { CollectionManager } from '@/collections-components';
	import Button from '@/components/ui-custom/button.svelte';
	import IconButton from '@/components/ui-custom/iconButton.svelte';
	import { m } from '@/i18n';
	import { pb } from '@/pocketbase';

	import { setDashboardNavbar } from '../+layout@.svelte';

	let { data } = $props();
	let { organization } = $derived(data);

	setDashboardNavbar({ title: 'Pipelines', right: navbarRight });

	//

	function runPipeline(pipeline: PipelinesResponse) {
		runWithLoading({
			fn: async () => {
				await pb.send('/api/pipeline/start', {
					method: 'POST',
					body: {
						yaml: pipeline.yaml
					}
				});
			}
		});
	}
</script>

<CollectionManager collection="pipelines">
	{#snippet records({ records })}
		<div class="space-y-4">
			{#each records as pipeline (pipeline.id)}
				<DashboardCard
					record={pipeline}
					avatar={() => pb.files.getURL(organization, organization.logo)}
					path={[organization.canonified_name, pipeline.canonified_name]}
				>
					{#snippet editAction()}
						<Button onclick={() => runPipeline(pipeline)}>
							<PlayIcon />{m.Run_now()}
						</Button>
						<Button
							href="/my/pipelines/settings-{pipeline.id}"
							variant="outline"
							size="icon"
						>
							<CogIcon />
						</Button>
						<IconButton href="/my/pipelines/edit-{pipeline.id}" icon={Pencil} />
					{/snippet}
				</DashboardCard>
			{/each}
		</div>
	{/snippet}
</CollectionManager>

{#snippet navbarRight()}
	<Button href="/my/pipelines/new">
		<Plus />
		{m.New()}
	</Button>
{/snippet}

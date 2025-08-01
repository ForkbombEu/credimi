<!--
SPDX-FileCopyrightText: 2025 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

<script lang="ts">
	import InfoBox from '$lib/layout/infoBox.svelte';
	import PageHeader from '$lib/layout/pageHeader.svelte';
	import type { IndexItem } from '$lib/layout/pageIndex.svelte';
	import { Building2, Layers3 } from 'lucide-svelte';
	import { String } from 'effect';
	import { z } from 'zod';
	import Card from '@/components/ui-custom/card.svelte';
	import { Badge } from '@/components/ui/badge';
	import { ConformanceCheckSchema } from '$services-and-products/_wallets/wallet-form-checks-table.svelte';
	import MarketplacePageLayout from '$lib/layout/marketplace-page-layout.svelte';

	//

	let { data } = $props();
	const { wallet } = $derived(data);

	//

	const sections = {
		general_info: {
			icon: Building2,
			anchor: 'general_info',
			label: 'General info'
		},
		conformance_checks: {
			icon: Layers3,
			anchor: 'conformance_checks',
			label: 'Conformance checks'
		}
	} satisfies Record<string, IndexItem>;

	const checks = $derived(z.array(ConformanceCheckSchema).safeParse(wallet.conformance_checks));

	//

	const statuses: Record<string, string> = {
		Running: 'bg-blue-300',
		TimedOut: 'bg-orange-200',
		Completed: 'bg-green-200',
		Failed: 'bg-red-200',
		ContinuedAsNew: 'bg-purple-200',
		Canceled: 'bg-slate-100',
		Terminated: 'bg-yellow-200',
		Paused: 'bg-yellow-200',
		Unspecified: 'bg-slate-100',
		Scheduled: 'bg-blue-300',
		Started: 'bg-blue-300',
		Open: 'bg-green-200',
		New: 'bg-blue-300',
		Initiated: 'bg-blue-300',
		Fired: 'bg-pink-200',
		CancelRequested: 'bg-yellow-200',
		Signaled: 'bg-pink-200',
		Pending: 'bg-purple-200',
		Retrying: 'bg-red-200'
	};
</script>

<MarketplacePageLayout tableOfContents={sections}>
	<div class="flex items-start gap-6">
		<div class="grow space-y-6">
			<PageHeader title={sections.general_info.label} id={sections.general_info.anchor} />

			<div>
				<InfoBox label="Description" value={wallet.description} />
			</div>

			<div class="grid grid-cols-2 gap-6">
				<InfoBox label="Homepage URL" value={wallet.home_url} />
				<InfoBox label="Repository URL" value={wallet.repository} />

				{#if String.isNonEmpty(wallet.appstore_url)}
					{@render AppStore(wallet.appstore_url)}
				{/if}

				{#if String.isNonEmpty(wallet.playstore_url)}
					{@render PlayStore(wallet.playstore_url)}
				{/if}
			</div>
		</div>
	</div>

	{#if checks.success}
		<div>
			<PageHeader
				title={sections.conformance_checks.label}
				id={sections.conformance_checks.anchor}
			/>
			<div class="space-y-2">
				{#each checks.data as check}
					{@const badgeColor = statuses[check.status]}
					<Card contentClass="flex justify-between items-center p-4">
						<div>
							<p class="font-bold">{check.standard}</p>
							<p>{check.test}</p>
						</div>
						<Badge class="{badgeColor} text-black hover:{badgeColor}">
							{check.status}
						</Badge>
					</Card>
				{/each}
			</div>
		</div>
	{/if}
</MarketplacePageLayout>

{#snippet AppStore(url: string)}
	<a href={url} target="_blank" class="">
		<svg
			id="livetype"
			xmlns="http://www.w3.org/2000/svg"
			viewBox="0 0 119.66407 40"
			class="h-16"
		>
			<title> Download_on_the_App_Store_Badge_US-UK_RGB_blk_4SVG_092917 </title>
			<g>
				<g>
					<g>
						<path
							d="M110.13477,0H9.53468c-.3667,0-.729,0-1.09473.002-.30615.002-.60986.00781-.91895.0127A13.21476,13.21476,0,0,0,5.5171.19141a6.66509,6.66509,0,0,0-1.90088.627A6.43779,6.43779,0,0,0,1.99757,1.99707,6.25844,6.25844,0,0,0,.81935,3.61816a6.60119,6.60119,0,0,0-.625,1.90332,12.993,12.993,0,0,0-.1792,2.002C.00587,7.83008.00489,8.1377,0,8.44434V31.5586c.00489.3105.00587.6113.01515.9219a12.99232,12.99232,0,0,0,.1792,2.0019,6.58756,6.58756,0,0,0,.625,1.9043A6.20778,6.20778,0,0,0,1.99757,38.001a6.27445,6.27445,0,0,0,1.61865,1.1787,6.70082,6.70082,0,0,0,1.90088.6308,13.45514,13.45514,0,0,0,2.0039.1768c.30909.0068.6128.0107.91895.0107C8.80567,40,9.168,40,9.53468,40H110.13477c.3594,0,.7246,0,1.084-.002.3047,0,.6172-.0039.9219-.0107a13.279,13.279,0,0,0,2-.1768,6.80432,6.80432,0,0,0,1.9082-.6308,6.27742,6.27742,0,0,0,1.6172-1.1787,6.39482,6.39482,0,0,0,1.1816-1.6143,6.60413,6.60413,0,0,0,.6191-1.9043,13.50643,13.50643,0,0,0,.1856-2.0019c.0039-.3106.0039-.6114.0039-.9219.0078-.3633.0078-.7246.0078-1.0938V9.53613c0-.36621,0-.72949-.0078-1.09179,0-.30664,0-.61426-.0039-.9209a13.5071,13.5071,0,0,0-.1856-2.002,6.6177,6.6177,0,0,0-.6191-1.90332,6.46619,6.46619,0,0,0-2.7988-2.7998,6.76754,6.76754,0,0,0-1.9082-.627,13.04394,13.04394,0,0,0-2-.17676c-.3047-.00488-.6172-.01074-.9219-.01269-.3594-.002-.7246-.002-1.084-.002Z"
							style="fill: #a6a6a6"
						></path>
						<path
							d="M8.44483,39.125c-.30468,0-.602-.0039-.90429-.0107a12.68714,12.68714,0,0,1-1.86914-.1631,5.88381,5.88381,0,0,1-1.65674-.5479,5.40573,5.40573,0,0,1-1.397-1.0166,5.32082,5.32082,0,0,1-1.02051-1.3965,5.72186,5.72186,0,0,1-.543-1.6572,12.41351,12.41351,0,0,1-.1665-1.875c-.00634-.2109-.01464-.9131-.01464-.9131V8.44434S.88185,7.75293.8877,7.5498a12.37039,12.37039,0,0,1,.16553-1.87207,5.7555,5.7555,0,0,1,.54346-1.6621A5.37349,5.37349,0,0,1,2.61183,2.61768,5.56543,5.56543,0,0,1,4.01417,1.59521a5.82309,5.82309,0,0,1,1.65332-.54394A12.58589,12.58589,0,0,1,7.543.88721L8.44532.875H111.21387l.9131.0127a12.38493,12.38493,0,0,1,1.8584.16259,5.93833,5.93833,0,0,1,1.6709.54785,5.59374,5.59374,0,0,1,2.415,2.41993,5.76267,5.76267,0,0,1,.5352,1.64892,12.995,12.995,0,0,1,.1738,1.88721c.0029.2832.0029.5874.0029.89014.0079.375.0079.73193.0079,1.09179V30.4648c0,.3633,0,.7178-.0079,1.0752,0,.3252,0,.6231-.0039.9297a12.73126,12.73126,0,0,1-.1709,1.8535,5.739,5.739,0,0,1-.54,1.67,5.48029,5.48029,0,0,1-1.0156,1.3857,5.4129,5.4129,0,0,1-1.3994,1.0225,5.86168,5.86168,0,0,1-1.668.5498,12.54218,12.54218,0,0,1-1.8692.1631c-.2929.0068-.5996.0107-.8974.0107l-1.084.002Z"
						></path>
					</g>
					<g id="_Group_" data-name="<Group>">
						<g id="_Group_2" data-name="<Group>">
							<g id="_Group_3" data-name="<Group>">
								<path
									id="_Path_"
									data-name="<Path>"
									d="M24.76888,20.30068a4.94881,4.94881,0,0,1,2.35656-4.15206,5.06566,5.06566,0,0,0-3.99116-2.15768c-1.67924-.17626-3.30719,1.00483-4.1629,1.00483-.87227,0-2.18977-.98733-3.6085-.95814a5.31529,5.31529,0,0,0-4.47292,2.72787c-1.934,3.34842-.49141,8.26947,1.3612,10.97608.9269,1.32535,2.01018,2.8058,3.42763,2.7533,1.38706-.05753,1.9051-.88448,3.5794-.88448,1.65876,0,2.14479.88448,3.591.8511,1.48838-.02416,2.42613-1.33124,3.32051-2.66914a10.962,10.962,0,0,0,1.51842-3.09251A4.78205,4.78205,0,0,1,24.76888,20.30068Z"
									style="fill: #fff"
								></path>
								<path
									id="_Path_2"
									data-name="<Path>"
									d="M22.03725,12.21089a4.87248,4.87248,0,0,0,1.11452-3.49062,4.95746,4.95746,0,0,0-3.20758,1.65961,4.63634,4.63634,0,0,0-1.14371,3.36139A4.09905,4.09905,0,0,0,22.03725,12.21089Z"
									style="fill: #fff"
								></path>
							</g>
						</g>
						<g>
							<path
								d="M42.30227,27.13965h-4.7334l-1.13672,3.35645H34.42727l4.4834-12.418h2.083l4.4834,12.418H43.438ZM38.0591,25.59082h3.752l-1.84961-5.44727h-.05176Z"
								style="fill: #fff"
							></path>
							<path
								d="M55.15969,25.96973c0,2.81348-1.50586,4.62109-3.77832,4.62109a3.0693,3.0693,0,0,1-2.84863-1.584h-.043v4.48438h-1.8584V21.44238H48.4302v1.50586h.03418a3.21162,3.21162,0,0,1,2.88281-1.60059C53.645,21.34766,55.15969,23.16406,55.15969,25.96973Zm-1.91016,0c0-1.833-.94727-3.03809-2.39258-3.03809-1.41992,0-2.375,1.23047-2.375,3.03809,0,1.82422.95508,3.0459,2.375,3.0459C52.30227,29.01563,53.24953,27.81934,53.24953,25.96973Z"
								style="fill: #fff"
							></path>
							<path
								d="M65.12453,25.96973c0,2.81348-1.50586,4.62109-3.77832,4.62109a3.0693,3.0693,0,0,1-2.84863-1.584h-.043v4.48438h-1.8584V21.44238H58.395v1.50586h.03418A3.21162,3.21162,0,0,1,61.312,21.34766C63.60988,21.34766,65.12453,23.16406,65.12453,25.96973Zm-1.91016,0c0-1.833-.94727-3.03809-2.39258-3.03809-1.41992,0-2.375,1.23047-2.375,3.03809,0,1.82422.95508,3.0459,2.375,3.0459C62.26711,29.01563,63.21438,27.81934,63.21438,25.96973Z"
								style="fill: #fff"
							></path>
							<path
								d="M71.71047,27.03613c.1377,1.23145,1.334,2.04,2.96875,2.04,1.56641,0,2.69336-.80859,2.69336-1.91895,0-.96387-.67969-1.541-2.28906-1.93652l-1.60937-.3877c-2.28027-.55078-3.33887-1.61719-3.33887-3.34766,0-2.14258,1.86719-3.61426,4.51855-3.61426,2.624,0,4.42285,1.47168,4.4834,3.61426h-1.876c-.1123-1.23926-1.13672-1.9873-2.63379-1.9873s-2.52148.75684-2.52148,1.8584c0,.87793.6543,1.39453,2.25488,1.79l1.36816.33594c2.54785.60254,3.60645,1.626,3.60645,3.44238,0,2.32324-1.85059,3.77832-4.79395,3.77832-2.75391,0-4.61328-1.4209-4.7334-3.667Z"
								style="fill: #fff"
							></path>
							<path
								d="M83.34621,19.2998v2.14258h1.72168v1.47168H83.34621v4.99121c0,.77539.34473,1.13672,1.10156,1.13672a5.80752,5.80752,0,0,0,.61133-.043v1.46289a5.10351,5.10351,0,0,1-1.03223.08594c-1.833,0-2.54785-.68848-2.54785-2.44434V22.91406H80.16262V21.44238H81.479V19.2998Z"
								style="fill: #fff"
							></path>
							<path
								d="M86.065,25.96973c0-2.84863,1.67773-4.63867,4.29395-4.63867,2.625,0,4.29492,1.79,4.29492,4.63867,0,2.85645-1.66113,4.63867-4.29492,4.63867C87.72609,30.6084,86.065,28.82617,86.065,25.96973Zm6.69531,0c0-1.9541-.89551-3.10742-2.40137-3.10742s-2.40039,1.16211-2.40039,3.10742c0,1.96191.89453,3.10645,2.40039,3.10645S92.76027,27.93164,92.76027,25.96973Z"
								style="fill: #fff"
							></path>
							<path
								d="M96.18606,21.44238h1.77246v1.541h.043a2.1594,2.1594,0,0,1,2.17773-1.63574,2.86616,2.86616,0,0,1,.63672.06934v1.73828a2.59794,2.59794,0,0,0-.835-.1123,1.87264,1.87264,0,0,0-1.93652,2.083v5.37012h-1.8584Z"
								style="fill: #fff"
							></path>
							<path
								d="M109.3843,27.83691c-.25,1.64355-1.85059,2.77148-3.89844,2.77148-2.63379,0-4.26855-1.76465-4.26855-4.5957,0-2.83984,1.64355-4.68164,4.19043-4.68164,2.50488,0,4.08008,1.7207,4.08008,4.46582v.63672h-6.39453v.1123a2.358,2.358,0,0,0,2.43555,2.56445,2.04834,2.04834,0,0,0,2.09082-1.27344Zm-6.28223-2.70215h4.52637a2.1773,2.1773,0,0,0-2.2207-2.29785A2.292,2.292,0,0,0,103.10207,25.13477Z"
								style="fill: #fff"
							></path>
						</g>
					</g>
				</g>
				<g id="_Group_4" data-name="<Group>">
					<g>
						<path
							d="M37.82619,8.731a2.63964,2.63964,0,0,1,2.80762,2.96484c0,1.90625-1.03027,3.002-2.80762,3.002H35.67092V8.731Zm-1.22852,5.123h1.125a1.87588,1.87588,0,0,0,1.96777-2.146,1.881,1.881,0,0,0-1.96777-2.13379h-1.125Z"
							style="fill: #fff"
						></path>
						<path
							d="M41.68068,12.44434a2.13323,2.13323,0,1,1,4.24707,0,2.13358,2.13358,0,1,1-4.24707,0Zm3.333,0c0-.97607-.43848-1.54687-1.208-1.54687-.77246,0-1.207.5708-1.207,1.54688,0,.98389.43457,1.55029,1.207,1.55029C44.57522,13.99463,45.01369,13.42432,45.01369,12.44434Z"
							style="fill: #fff"
						></path>
						<path
							d="M51.57326,14.69775h-.92187l-.93066-3.31641h-.07031l-.92676,3.31641h-.91309l-1.24121-4.50293h.90137l.80664,3.436h.06641l.92578-3.436h.85254l.92578,3.436h.07031l.80273-3.436h.88867Z"
							style="fill: #fff"
						></path>
						<path
							d="M53.85354,10.19482H54.709v.71533h.06641a1.348,1.348,0,0,1,1.34375-.80225,1.46456,1.46456,0,0,1,1.55859,1.6748v2.915h-.88867V12.00586c0-.72363-.31445-1.0835-.97168-1.0835a1.03294,1.03294,0,0,0-1.0752,1.14111v2.63428h-.88867Z"
							style="fill: #fff"
						></path>
						<path d="M59.09377,8.437h.88867v6.26074h-.88867Z" style="fill: #fff"></path>
						<path
							d="M61.21779,12.44434a2.13346,2.13346,0,1,1,4.24756,0,2.1338,2.1338,0,1,1-4.24756,0Zm3.333,0c0-.97607-.43848-1.54687-1.208-1.54687-.77246,0-1.207.5708-1.207,1.54688,0,.98389.43457,1.55029,1.207,1.55029C64.11232,13.99463,64.5508,13.42432,64.5508,12.44434Z"
							style="fill: #fff"
						></path>
						<path
							d="M66.4009,13.42432c0-.81055.60352-1.27783,1.6748-1.34424l1.21973-.07031v-.38867c0-.47559-.31445-.74414-.92187-.74414-.49609,0-.83984.18213-.93848.50049h-.86035c.09082-.77344.81836-1.26953,1.83984-1.26953,1.12891,0,1.76563.562,1.76563,1.51318v3.07666h-.85547v-.63281h-.07031a1.515,1.515,0,0,1-1.35254.707A1.36026,1.36026,0,0,1,66.4009,13.42432Zm2.89453-.38477v-.37646l-1.09961.07031c-.62012.0415-.90137.25244-.90137.64941,0,.40527.35156.64111.835.64111A1.0615,1.0615,0,0,0,69.29543,13.03955Z"
							style="fill: #fff"
						></path>
						<path
							d="M71.34816,12.44434c0-1.42285.73145-2.32422,1.86914-2.32422a1.484,1.484,0,0,1,1.38086.79h.06641V8.437h.88867v6.26074h-.85156v-.71143h-.07031a1.56284,1.56284,0,0,1-1.41406.78564C72.0718,14.772,71.34816,13.87061,71.34816,12.44434Zm.918,0c0,.95508.4502,1.52979,1.20313,1.52979.749,0,1.21191-.583,1.21191-1.52588,0-.93848-.46777-1.52979-1.21191-1.52979C72.72121,10.91846,72.26613,11.49707,72.26613,12.44434Z"
							style="fill: #fff"
						></path>
						<path
							d="M79.23,12.44434a2.13323,2.13323,0,1,1,4.24707,0,2.13358,2.13358,0,1,1-4.24707,0Zm3.333,0c0-.97607-.43848-1.54687-1.208-1.54687-.77246,0-1.207.5708-1.207,1.54688,0,.98389.43457,1.55029,1.207,1.55029C82.12453,13.99463,82.563,13.42432,82.563,12.44434Z"
							style="fill: #fff"
						></path>
						<path
							d="M84.66945,10.19482h.85547v.71533h.06641a1.348,1.348,0,0,1,1.34375-.80225,1.46456,1.46456,0,0,1,1.55859,1.6748v2.915H87.605V12.00586c0-.72363-.31445-1.0835-.97168-1.0835a1.03294,1.03294,0,0,0-1.0752,1.14111v2.63428h-.88867Z"
							style="fill: #fff"
						></path>
						<path
							d="M93.51516,9.07373v1.1416h.97559v.74854h-.97559V13.2793c0,.47168.19434.67822.63672.67822a2.96657,2.96657,0,0,0,.33887-.02051v.74023a2.9155,2.9155,0,0,1-.4834.04541c-.98828,0-1.38184-.34766-1.38184-1.21582v-2.543h-.71484v-.74854h.71484V9.07373Z"
							style="fill: #fff"
						></path>
						<path
							d="M95.70461,8.437h.88086v2.48145h.07031a1.3856,1.3856,0,0,1,1.373-.80664,1.48339,1.48339,0,0,1,1.55078,1.67871v2.90723H98.69v-2.688c0-.71924-.335-1.0835-.96289-1.0835a1.05194,1.05194,0,0,0-1.13379,1.1416v2.62988h-.88867Z"
							style="fill: #fff"
						></path>
						<path
							d="M104.76125,13.48193a1.828,1.828,0,0,1-1.95117,1.30273A2.04531,2.04531,0,0,1,100.73,12.46045a2.07685,2.07685,0,0,1,2.07617-2.35254c1.25293,0,2.00879.856,2.00879,2.27V12.688h-3.17969v.0498a1.1902,1.1902,0,0,0,1.19922,1.29,1.07934,1.07934,0,0,0,1.07129-.5459Zm-3.126-1.45117h2.27441a1.08647,1.08647,0,0,0-1.1084-1.1665A1.15162,1.15162,0,0,0,101.63527,12.03076Z"
							style="fill: #fff"
						></path>
					</g>
				</g>
			</g>
		</svg>
	</a>
{/snippet}

{#snippet PlayStore(url: string)}
	<a href={url} target="_blank" class="shrink-0">
		<img
			class="h-16"
			alt="PlayStore url"
			src="data:image/png;base64,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"
		/>
	</a>
{/snippet}

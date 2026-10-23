// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import * as conformanceAutomation from '$lib/content/conformance-automation';
import * as whyCredimi from '$lib/content/why-credimi';
import { fcafCatalog, standards } from '$lib/layout/marketing/standards';

import type { PublicPage, PublicPageSection } from './public-pages';

import { absoluteUrl, ATLAS_URL, DOCS_URL, plainText, SITE_DESCRIPTION, SOURCE_URL } from './index';

// llms.txt format: https://llmstxt.org — H1, summary blockquote, then H2 sections of links.

const sectionTitles: Record<PublicPageSection, string> = {
	core: 'Start here',
	guides: 'Guides and comparisons',
	hub: 'Hub: live wallets, issuers, verifiers and credentials',
	'conformance-suites': 'Conformance check suites',
	'conformance-checks': 'Conformance checks',
	news: 'News',
	organizations: 'Organizations on the Hub'
};

const keyFacts = [
	'Credimi tests EUDI wallets, credential issuers and verifiers (relying parties), alone and together.',
	`It runs OpenID4VCI 1.0, OpenID4VP 1.0 (and Drafts 23, 24), the OpenID Foundation conformance suite, WE BUILD / EWC and the EUDI Functional Conformance Assessment Framework (FCAF: ${fcafCatalog.testCases} test cases in ${fcafCatalog.scenarios} scenarios as of ${fcafCatalog.asOf}).`,
	'Pipelines drive real wallet apps on Android devices, emulators and iOS simulators (Maestro) and call live issuers and verifiers (StepCI), then judge the run against FCAF and protocol test cases.',
	'Every run keeps evidence: a timestamped Temporal history of each step, protocol messages, video, screenshots, logs, and SHA-256 hashed FCAF evidence.',
	'CI integration per role: run-wallet-apk, run-issuer and run-verifier endpoints and a GitHub Action.',
	'Use it at credimi.io or self-host it: open source under AGPL-3.0, built by Forkbomb BV.'
];

function link(page: PublicPage, origin: string, maxDescription = 220): string {
	const description = plainText(page.description, maxDescription);
	const label = page.kind ? `${page.title} (${page.kind})` : page.title;
	return `- [${label}](${absoluteUrl(page.path, origin)})${description ? `: ${description}` : ''}`;
}

function header(origin: string): string[] {
	return [
		'# Credimi',
		'',
		`> ${SITE_DESCRIPTION}`,
		'',
		...keyFacts.map((fact) => `- ${fact}`),
		'',
		`Website: ${origin} · Documentation: ${DOCS_URL} · Source: ${SOURCE_URL}`
	];
}

/** /llms.txt: a curated, linked index of the site. */
export function renderLlmsIndex(pages: PublicPage[], origin: string): string {
	// Individual checks would drown the index; their suites link to them and llms-full.txt lists them.
	const sections = (Object.keys(sectionTitles) as PublicPageSection[]).filter(
		(section) => section !== 'conformance-checks'
	);

	const body = sections.flatMap((section) => {
		const sectionPages = pages.filter((page) => page.section === section);
		if (sectionPages.length === 0) return [];
		return [
			'',
			`## ${sectionTitles[section]}`,
			'',
			...sectionPages.map((page) => link(page, origin))
		];
	});

	return [
		...header(origin),
		...body,
		'',
		'## Optional',
		'',
		`- [Full text of this site for LLMs](${origin}/llms-full.txt): every page above with its complete description, the full competitor comparison and every conformance check`,
		`- [Credimi documentation](${DOCS_URL}/): manual, API and pipeline reference`,
		`- [EUDI Trust & Conformance Atlas](${ATLAS_URL}/): the EUDI Wallet rulebook mapped, with adoption status of every law, specification and standard`,
		`- [Source code](${SOURCE_URL}): AGPL-3.0`,
		''
	].join('\n');
}

const supportWords: Record<whyCredimi.Support, string> = {
	core: 'Yes',
	partial: 'Partial',
	none: 'No'
};

function whyCredimiSection(origin: string): string[] {
	const { columns, competitors, eudiGaps, matrices, credimiBestAt, chooseCredimi } = whyCredimi;
	const groups = (['deviceClouds', 'maestro', 'labs', 'suites'] as const).map((id) => [
		`### ${columns[id].category} (${columns[id].products.join(', ')})`,
		'',
		...competitors
			.filter((competitor) => competitor.column === id)
			.map((competitor) => `- ${competitor.name}, best at: ${competitor.bestAt}`),
		`- For EUDI wallet testing: ${eudiGaps[id]}`,
		''
	]);

	const tables = matrices.flatMap((matrix) => {
		const head = matrix.columns.map((id) => columns[id].products.join(' / '));
		return [
			`### ${matrix.title.replace(/:$/, '')}`,
			'',
			`| Capability | ${head.join(' | ')} |`,
			`|---|${head.map(() => '---').join('|')}|`,
			...matrix.rows.map(
				(row) =>
					`| ${row.label} | ${row.values.map((value) => supportWords[value]).join(' | ')} |`
			),
			...(matrix.footnote ? ['', matrix.footnote] : []),
			''
		];
	});

	return [
		`## Why Credimi: how it compares (${absoluteUrl('/pages/why-credimi', origin)})`,
		'',
		'Device clouds and test labs answer "does my app work on this device?". Credimi answers "does my wallet, issuer or verifier conform, and does it interoperate with the rest of the EUDI ecosystem?". Each tool below is good at its own job; none was built to test EUDI wallets, issuers and verifiers together.',
		'',
		...groups.flat(),
		'### Credimi',
		'',
		...credimiBestAt.map((item) => `- Best at: ${item}`),
		'',
		'Choose Credimi when:',
		...chooseCredimi.map((item) => `- ${item}`),
		'',
		'Capability matrices (Yes = built in, Partial = partial or needs custom work, No = not part of the product):',
		'',
		...tables
	];
}

function conformanceAutomationSection(origin: string): string[] {
	const { definition, manual, automated, steps, ciEndpoints, benefits, closing } =
		conformanceAutomation;
	return [
		`## What is conformance automation (${absoluteUrl('/pages/conformance-automation', origin)})`,
		'',
		...definition.flatMap((paragraph) => [paragraph, '']),
		'Manual review:',
		...manual.map((item) => `- ${item}`),
		'',
		'Automated conformance:',
		...automated.map((item) => `- ${item}`),
		'',
		'How Credimi automates it:',
		...steps.map((step, i) => `${i + 1}. ${step.title}: ${step.text}`),
		`CI endpoints per role: ${ciEndpoints.join(', ')}.`,
		'',
		...benefits.map((benefit) => `- ${benefit.title}: ${benefit.text}`),
		'',
		closing,
		''
	];
}

function standardsSection(): string[] {
	return [
		'## Standards and conformance coverage',
		'',
		...standards.map((standard) => `- ${standard.name}: ${standard.scope}`),
		`- EUDI FCAF, wallet solution and relying party roles: ${fcafCatalog.testCases} test cases in ${fcafCatalog.scenarios} scenarios (catalog as of ${fcafCatalog.asOf})`,
		''
	];
}

/** /llms-full.txt: the site's content as one markdown document. */
export function renderLlmsFull(pages: PublicPage[], origin: string): string {
	const listings = (Object.keys(sectionTitles) as PublicPageSection[])
		.filter((section) => section !== 'core' && section !== 'guides')
		.flatMap((section) => {
			const sectionPages = pages.filter((page) => page.section === section);
			if (sectionPages.length === 0) return [];
			return [
				`## ${sectionTitles[section]}`,
				'',
				...sectionPages.map((page) => link(page, origin, 2000)),
				''
			];
		});

	return [
		...header(origin),
		'',
		...whyCredimiSection(origin),
		...conformanceAutomationSection(origin),
		...standardsSection(),
		...listings
	].join('\n');
}

// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Copy of /pages/conformance-automation, shared by the page and llms-full.txt.

export const definition = [
	'Conformance automation uses tools to check that a product follows the technical standards, protocols and regulatory specifications it claims to support.',
	'In digital identity, that means testing a wallet, issuer or verifier against OpenID4VCI, OpenID4VP, the EUDI ARF and its conformance framework (FCAF), and running the same checks again whenever the product or the standard changes.'
];

export const manual = [
	'Runs once, before a release or an audit.',
	'A snapshot that ages with every commit.',
	'Steps and results live in documents and spreadsheets.',
	'Hard to repeat the same way twice.'
];

export const automated = [
	'Runs on every change, on a schedule, or from CI.',
	'The same suites, run the same way, every time.',
	'Every input, output and artifact is kept with the run.',
	'Failures surface before your users or an assessor find them.'
];

export type StepId = 'checks' | 'pipeline' | 'trigger' | 'evidence';

export const steps: { id: StepId; title: string; text: string }[] = [
	{
		id: 'checks',
		title: 'Choose the checks',
		text: 'Pick suites from the catalog: the OpenID Foundation conformance suite, WE BUILD / EWC, EUDIW verifier tests and EUDI FCAF.'
	},
	{
		id: 'pipeline',
		title: 'Build a pipeline',
		text: 'Chain checks with real flows: StepCI calls to issuers and verifiers, Maestro flows on wallets running on phones, emulators or iOS simulators.'
	},
	{
		id: 'trigger',
		title: 'Run it on every change',
		text: 'Trigger runs from a pull request with the GitHub Action, from the CI endpoints, or on a schedule.'
	},
	{
		id: 'evidence',
		title: 'Read the evidence',
		text: 'Each run keeps a timestamped Temporal history, protocol messages, video and logs. Published pipelines feed the public scoreboard.'
	}
];

export const ciEndpoints = ['run-wallet-apk', 'run-issuer', 'run-verifier'];

export type BenefitId = 'speed' | 'interop' | 'reports' | 'standards';

export const benefits: { id: BenefitId; title: string; text: string }[] = [
	{
		id: 'speed',
		title: 'Ship without waiting for a review',
		text: 'Check a new feature or integration in the same pipeline that checks everything else, from the first commit.'
	},
	{
		id: 'interop',
		title: 'See who you actually interoperate with',
		text: 'Run the same scenario against the wallets, issuers and verifiers published on the Hub, and see which ones complete the flow.'
	},
	{
		id: 'reports',
		title: 'Hand over reports, not screenshots',
		text: 'Every run keeps its evidence. FCAF reports hash each evidence item and point into the run history, so an assessor can check every claim.'
	},
	{
		id: 'standards',
		title: 'Keep up as the standards move',
		text: 'The catalog covers several versions side by side, such as OpenID4VP 1.0, Draft 23 and Draft 24. New suites plug into the pipelines you already run.'
	}
];

export const closing =
	'Conformance is not a certificate you earn once. It is a check you run on every release.';

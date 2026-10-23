// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export type Support = 'core' | 'partial' | 'none';

export const supportLabels: Record<Support, string> = {
	core: 'Built in',
	partial: 'Partial, or needs custom work',
	none: 'Not part of the product'
};

export type ColumnId = 'credimi' | 'deviceClouds' | 'maestro' | 'labs' | 'suites';

export type Column = { products: string[]; category: string };

export const columns: Record<ColumnId, Column> = {
	credimi: { products: ['Credimi'], category: 'EUDI conformance and interoperability' },
	deviceClouds: {
		products: ['BrowserStack', 'Sauce Labs', 'LambdaTest'],
		category: 'Device clouds'
	},
	maestro: { products: ['Maestro Cloud', 'DeviceCloud'], category: 'Maestro clouds' },
	labs: { products: ['Firebase Test Lab', 'AWS Device Farm'], category: 'Google and AWS labs' },
	suites: {
		products: ['OpenID Foundation', 'EC Interoperability Test Bed'],
		category: 'Conformance suites'
	}
};

export type MatrixRow = {
	label: string;
	/** One value per matrix column, in column order. */
	values: Support[];
	/** Short qualifier shown under the Credimi dot. */
	credimiNote?: string;
};

export type Matrix = {
	id: string;
	title: string;
	intro?: string;
	columns: ColumnId[];
	rows: MatrixRow[];
	footnote?: string;
};

const c: Support = 'core';
const p: Support = 'partial';
const n: Support = 'none';

export const matrices: Matrix[] = [
	{
		id: 'at-a-glance',
		title: 'At a glance:',
		columns: ['credimi', 'deviceClouds', 'maestro', 'labs', 'suites'],
		rows: [
			{ label: 'Built for EUDI and eIDAS 2', values: [c, n, n, n, p] },
			{ label: 'Tests issuer, wallet and verifier in one run', values: [c, n, n, n, n] },
			{ label: 'Mobile wallet UI automation (Maestro)', values: [c, c, c, p, n] },
			{
				label: 'Backend protocol conformance (OpenID4VCI, OpenID4VP)',
				values: [c, n, n, n, c]
			},
			{ label: 'EUDI FCAF test cases built in', values: [c, n, n, n, p] },
			{ label: 'Several conformance suites behind one interface', values: [c, n, n, n, n] },
			{ label: 'Public hub of live issuers, verifiers and wallets', values: [c, n, n, n, n] },
			{
				label: 'Audit-grade evidence (timeline, hashes, video, logs)',
				values: [c, p, p, p, p]
			},
			{ label: 'Runs on your own devices, emulators and network', values: [c, p, n, n, c] },
			{ label: 'Open source (AGPL-3.0)', values: [c, n, p, n, p] }
		]
	},
	{
		id: 'roles',
		title: 'Who you can test, and against whom:',
		columns: ['credimi', 'deviceClouds', 'maestro', 'suites'],
		rows: [
			{ label: 'Wallet (Android, iOS): UI flows', values: [c, c, c, n] },
			{ label: 'Wallet: protocol conformance', values: [c, n, n, c] },
			{ label: 'Credential issuer: protocol conformance', values: [c, n, n, c] },
			{ label: 'Verifier / relying party: protocol conformance', values: [c, n, n, c] },
			{ label: 'Wallet with third-party issuers', values: [c, p, p, n] },
			{ label: 'Wallet with third-party verifiers', values: [c, p, p, n] },
			{ label: 'Issuer to wallet to verifier, end to end', values: [c, n, n, n] },
			{ label: 'Same scenario against many wallets', values: [c, p, p, n] }
		],
		footnote:
			'Partial on device and Maestro clouds means you can script it yourself, but you also build and host the issuer and verifier side, the deeplink plumbing and the assertions.'
	},
	{
		id: 'evidence',
		title: 'Evidence kept for every run:',
		columns: ['credimi', 'deviceClouds', 'maestro'],
		rows: [
			{ label: 'Device video', values: [c, c, c] },
			{ label: 'Screenshots and last frame', values: [c, c, c] },
			{ label: 'Device logs (Logcat / console)', values: [c, c, p] },
			{
				label: 'Timestamped history of every step input and output',
				values: [c, n, n],
				credimiNote: 'Temporal'
			},
			{ label: 'Protocol messages (offers, requests, responses)', values: [c, n, n] },
			{ label: 'Hashed evidence index per test case', values: [c, n, n] },
			{ label: 'FCAF / conformance report attached to the run', values: [c, n, n] },
			{ label: 'Public scoreboard of pipeline results', values: [c, n, n] }
		]
	},
	{
		id: 'automation',
		title: 'Automation and CI:',
		columns: ['credimi', 'deviceClouds', 'maestro'],
		rows: [
			{ label: 'Run from a pull request (GitHub Action)', values: [c, c, c] },
			{ label: 'Upload a fresh wallet APK per commit', values: [c, c, c] },
			{
				label: 'Test a temporary issuer or verifier deployment per commit',
				values: [c, n, n]
			},
			{ label: 'Scheduled runs', values: [c, p, p] },
			{ label: 'Device queueing across teams', values: [c, c, c] },
			{ label: 'Live view of the device during a run', values: [c, c, p] },
			{ label: 'Results by API callback or email', values: [c, p, p] }
		],
		footnote:
			'CI endpoints exist for each role: run-wallet-apk, run-issuer and run-verifier. Wallet, issuer and verifier teams can each gate their own releases on the same shared scenarios.'
	}
];

export type Competitor = {
	name: string;
	column: Exclude<ColumnId, 'credimi'>;
	bestAt: string;
};

export const competitors: Competitor[] = [
	{
		name: 'BrowserStack',
		column: 'deviceClouds',
		bestAt: 'Breadth of real iOS and Android devices for manual and automated app testing (Appium, Espresso, XCUITest), with visual testing on top.'
	},
	{
		name: 'Sauce Labs',
		column: 'deviceClouds',
		bestAt: 'Enterprise-scale web and mobile test execution on real devices, emulators and simulators, with test analytics and error reporting.'
	},
	{
		name: 'LambdaTest',
		column: 'deviceClouds',
		bestAt: 'Fast, parallel cross-browser and mobile app runs, with HyperExecute orchestration for large suites.'
	},
	{
		name: 'Maestro Cloud',
		column: 'maestro',
		bestAt: 'Running Maestro flows in parallel on managed emulators and simulators. Flows are short, readable YAML; Credimi drives wallets with Maestro too.'
	},
	{
		name: 'Firebase Test Lab',
		column: 'labs',
		bestAt: 'Robo and instrumentation tests on Google-hosted devices, built into the Firebase and Android Studio toolchain.'
	},
	{
		name: 'AWS Device Farm',
		column: 'labs',
		bestAt: 'Real phones and tablets inside AWS, with automated test runs and remote access sessions in your AWS account.'
	},
	{
		name: 'OpenID Foundation conformance suite',
		column: 'suites',
		bestAt: 'The reference tests for OpenID4VCI and OpenID4VP, and the route to OpenID certification.'
	},
	{
		name: 'EC Interoperability Test Bed',
		column: 'suites',
		bestAt: 'Conformance testing of messages and data against EU specifications.'
	}
];

/** What each product group leaves to you when the system under test is an EUDI wallet, issuer or verifier. */
export const eudiGaps: Record<Exclude<ColumnId, 'credimi'>, string> = {
	deviceClouds:
		'Tests one app at a time. The issuer, the verifier, the deeplink plumbing and the protocol assertions are yours to build and host.',
	maestro:
		"Drives one app's UI. No issuer or verifier side, no OpenID4VCI or OpenID4VP conformance, no FCAF test cases.",
	labs: 'Device and test-framework coverage, not identity flows: no protocol conformance, no issuer to wallet to verifier journey, no assessment evidence.',
	suites: 'One component at a time, against a simulated counterpart. No real wallet on a real device and no end-to-end journey. Credimi runs these suites for you, side by side.'
};

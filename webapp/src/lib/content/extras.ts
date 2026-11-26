// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Copy of /pages/extras. Every claim is taken from the README of the tool's repository.

import type { extras } from '$lib/layout/topbar-links';

export type ExtraId = (typeof extras)[number]['id'];

export const tools: Record<
	ExtraId,
	{
		problem: string;
		does: string;
		limit: string;
		/** What the screenshot at /pages/extras/<id>.webp shows. */
		screen: string;
		input: string;
		output: string;
		/** Who it helps, as a sentence. */
		for: string;
	}
> = {
	'capture-wallet': {
		problem: 'What does my wallet actually send?',
		does: 'A test issuer and verifier for a PID in SD-JWT VC or mdoc format. It records the exchange while your wallet receives or presents the credential.',
		limit: 'A successful issuance says nothing about EUDI trust or interoperability.',
		screen: 'The start page of the wallet recorder: choose a flow, issue or present, choose a credential, and start a session.',
		input: 'A QR code scanned by your wallet',
		output: 'OpenID4VCI requests, proof keys, the OpenID4VP response',
		for: 'Wallet builders, and auditors collecting evidence.'
	},
	'capture-issuer-verifier': {
		problem: 'What is this issuer or verifier asking for?',
		does: 'Resolves a credential offer or a presentation request without a wallet or a device, so you can compare what a service advertises with what you expected.',
		limit: 'It reads what a service publishes. It does not complete the exchange.',
		screen: 'Two extractors side by side: one takes a credential offer and returns issuer and authorization metadata, the other takes a presentation request and returns its DCQL.',
		input: 'A credential offer or a presentation request',
		output: 'Issuer and authorization-server metadata, DCQL details',
		for: 'Wallet builders, issuers, verifiers and auditors.'
	},
	'trust-inspector': {
		problem: 'Is this trusted list well formed and signed?',
		does: 'Assesses a trusted list, or a List of Trusted Lists and every list it points to, against ETSI TS 119 602 and TS 119 612.',
		limit: 'Technical evidence, not a legal conformance decision.',
		screen: 'The trusted list inspector form: audit a List of Trusted Lists from a URL or a local file, or inspect one trusted list document.',
		input: 'A list URL or file: XML, JSON or compact JWS',
		output: 'Findings marked pass, warning, failure or not final',
		for: 'Member States, list operators, verifiers and auditors.'
	},
	lote: {
		problem: 'How does my product react to a broken trusted list?',
		does: 'Publishes versioned test lists: healthy ones, and ones with a known defect such as an expired next update. Providers can ask to be onboarded on a list.',
		limit: 'For testing only. Nothing it publishes has legal effect.',
		screen: 'The catalogue of published test trusted lists: each row shows the list family, sequence number, issue date, signature result and whether the list is deliberately broken.',
		input: 'A provider to onboard, or a list to download',
		output: 'Signed JSON LoTE and XML Trusted List artifacts',
		for: 'Wallet builders, issuers, verifiers and list operators.'
	},
	tsl: {
		problem: 'Does revoking a credential change the outcome?',
		does: 'Creates test credentials with a status entry, revokes them, and decodes the signed Token Status List that carries the result.',
		limit: 'Simplified test infrastructure. The explorer does not verify signatures.',
		screen: 'The status list server console: a table of status lists by country and document type, with the number of allocated entries, beside an output panel.',
		input: 'A test credential, or a status list token',
		output: 'The list URI and index, and the decoded JWT or CWT list',
		for: 'Wallet builders, issuers and verifiers.'
	},
	federation: {
		problem: 'Can my implementation build a trust chain?',
		does: 'An OpenID Federation 1.0 Trust Anchor for testing. It signs statements about its registered issuers and verifiers, pinning their approved public keys.',
		limit: 'A test federation. Membership confers no accreditation.',
		screen: 'The decoded Entity Configuration of the Trust Anchor: its JOSE header with algorithm and key ID, followed by its claims.',
		input: 'The entity ID of a registered member',
		output: 'Signed Entity Configuration and Subordinate Statements',
		for: 'Wallet builders, issuers and verifiers.'
	},
	attestbook: {
		problem: 'Does this credential match its rulebook?',
		does: 'A catalogue of attestation rulebooks with their version history, and a playground that validates an SD-JWT VC against one pinned version.',
		limit: 'Checks structure. Cryptography and issuer trust are reported as not tested.',
		screen: 'The rulebook catalogue: searchable rows with provider, version, publication status and attestation types.',
		input: 'A compact SD-JWT VC and a rulebook version',
		output: 'Violations with the JSON Pointer path of each one',
		for: 'Rulebook authors, standards bodies, issuers and auditors.'
	},
	atlas: {
		problem: 'Which law or standard does this requirement come from?',
		does: 'EU law, implementing acts, the Architecture and Reference Framework, standards, protocols and conformance evidence in one cross-linked, searchable reference.',
		limit: 'A map of the sources. Read the cited document for the binding text.',
		screen: 'The Atlas home page: a search box over a stack of layers, from law and implementing acts up to executable evidence, each with its document count.',
		input: 'A search term or a filter',
		output: 'Documents with their adoption status and cited clauses',
		for: 'Everyone, from implementers to Member States, standards bodies, CABs and NABs.'
	}
};

/** The suite, by what each tool does for you. */
export const layers: { id: string; title: string; text: string; tools: ExtraId[] }[] = [
	{
		id: 'inspect',
		title: 'Inspect your implementation',
		text: 'Give a tool your wallet, your service or your list, and see what is in it.',
		tools: ['capture-wallet', 'capture-issuer-verifier', 'trust-inspector']
	},
	{
		id: 'infrastructure',
		title: 'Run against known trust infrastructure',
		text: 'Real components with non-authoritative data, to point your product at.',
		tools: ['lote', 'tsl', 'federation']
	},
	{
		id: 'rules',
		title: 'Look up the rules',
		text: 'The rulebooks and sources every requirement comes from.',
		tools: ['attestbook', 'atlas']
	}
];

/** One debugging run across three tools: a verifier that keeps accepting a revoked credential. */
export const workflow: { tool: ExtraId | 'verifier' | 'pipeline'; title: string; text: string }[] =
	[
		{
			tool: 'capture-wallet',
			title: 'Issue two test credentials',
			text: 'Keep “Include a Token Status List reference” on.'
		},
		{
			tool: 'tsl',
			title: 'Revoke one, keep the other',
			text: 'The untouched credential is your control.'
		},
		{
			tool: 'verifier',
			title: 'Present both to your verifier',
			text: 'Wait out the list cache, up to 12 hours. The control should pass, the revoked one should fail.'
		},
		{
			tool: 'pipeline',
			title: 'Keep it as a regression test',
			text: 'Put the same steps in a pipeline and rerun them on every change.'
		}
	];

/** Response of `POST /credentials`, as documented in the capture-status-list README. */
export const statusEntryExample = `{
  "credential_id": "cred-001",
  "idx": 42,
  "status_list_uri": "http://localhost:8000/status/1",
  "referenced_token_status": {
    "status_list": { "idx": 42, "uri": "http://localhost:8000/status/1" }
  }
}`;

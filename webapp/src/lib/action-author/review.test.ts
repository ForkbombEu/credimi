// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { ReviewAction } from './api';

import {
	blockedBy,
	canAccept,
	canSend,
	decidedActions,
	reviewDecisions,
	startingName,
	type Choices
} from './review';

// sha256("a")
const SHA_A = 'ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb';

function action(overrides: Partial<ReviewAction>): ReviewAction {
	return {
		key: 'a1',
		name: 'install',
		build: 'org/w',
		code: 'a',
		candidate_sha256: SHA_A,
		plan: { steps: [] },
		replaces: '',
		error: '',
		starts_from: [],
		starts_from_names: {},
		used_by: 0,
		...overrides
	};
}

const onboarding = action({ key: 'a1', name: 'Onboarding' });
const credential = action({
	key: 'a2',
	name: 'Credential',
	starts_from: ['a1'],
	starts_from_names: { a1: 'Onboarding' }
});

describe('canAccept', () => {
	it('refuses a wrong hash', async () => {
		expect(await canAccept(action({ candidate_sha256: '0'.repeat(64) }))).toBe(false);
	});

	it('refuses an action without a plan', async () => {
		expect(await canAccept(action({ plan: null }))).toBe(false);
	});

	it('accepts the matching YAML with a plan', async () => {
		expect(await canAccept(action({}))).toBe(true);
	});
});

describe('blockedBy', () => {
	it('names a starting action of the review that is not accepted', () => {
		const choices: Choices = {
			a1: { decision: 'refuse', comment: '' },
			a2: { decision: null, comment: '' }
		};
		expect(blockedBy(credential, choices)).toBe('a1');
		choices.a1.decision = null;
		expect(blockedBy(credential, choices)).toBe('a1');
	});

	it('lets an action start from an accepted one', () => {
		const choices: Choices = {
			a1: { decision: 'accept', comment: '' },
			a2: { decision: null, comment: '' }
		};
		expect(blockedBy(credential, choices)).toBeUndefined();
	});

	it('ignores a starting action saved by an earlier review', () => {
		expect(blockedBy(credential, { a2: { decision: null, comment: '' } })).toBeUndefined();
	});
});

describe('startingName', () => {
	const names = { a1: 'Onboarding in review', a2: 'Credential' };

	it("prefers Credimi's name, which also names actions saved by an earlier review", () => {
		const saved = action({ starts_from: ['a3'], starts_from_names: { a3: 'Saved earlier' } });
		expect(startingName(credential, 'a1', names)).toBe('Onboarding');
		expect(startingName(saved, 'a3', names)).toBe('Saved earlier');
	});

	it("falls back to the review's name, then to the key", () => {
		const unnamed = action({ starts_from: ['a1', 'a4'] });
		expect(startingName(unnamed, 'a1', names)).toBe('Onboarding in review');
		expect(startingName(unnamed, 'a4', names)).toBe('a4');
	});
});

describe('canSend', () => {
	const actions = [onboarding, credential];

	it('needs a decision on every action that is not blocked', () => {
		expect(
			canSend(actions, {
				a1: { decision: 'accept', comment: '' },
				a2: { decision: null, comment: '' }
			})
		).toBe(false);
	});

	it('lets an action whose starting action is not accepted wait for the next review', () => {
		const choices: Choices = {
			a1: { decision: 'change', comment: 'name it E2E onboarding' },
			a2: { decision: null, comment: '' }
		};
		expect(canSend(actions, choices)).toBe(true);
		expect(decidedActions(actions, choices).map((action) => action.key)).toEqual(['a1']);
	});

	it('needs at least one decision', () => {
		expect(
			canSend(actions, {
				a1: { decision: null, comment: '' },
				a2: { decision: null, comment: '' }
			})
		).toBe(false);
	});

	it('needs a comment on every change', () => {
		const choices: Choices = {
			a1: { decision: 'change', comment: '  ' },
			a2: { decision: 'refuse', comment: '' }
		};
		expect(canSend(actions, choices)).toBe(false);
		choices.a1.comment = 'name it E2E onboarding';
		expect(canSend(actions, choices)).toBe(true);
	});

	it('refuses an accept whose starting action is not accepted', () => {
		expect(
			canSend(actions, {
				a1: { decision: 'refuse', comment: '' },
				a2: { decision: 'accept', comment: '' }
			})
		).toBe(false);
	});

	it('sends every action accepted in chain order', () => {
		expect(
			canSend(actions, {
				a1: { decision: 'accept', comment: '' },
				a2: { decision: 'accept', comment: '' }
			})
		).toBe(true);
	});
});

describe('reviewDecisions', () => {
	const profile = { pin: '123456' };
	const fcaf = action({ key: 'a1', plan: { slot: 'onboarding', fcaf_role: 'fcaf-reset' } });
	const chat = action({ key: 'a2', plan: { slot: 'onboarding' } });
	const refused = action({ key: 'a3', plan: { slot: 'onboarding', fcaf_role: 'fcaf-x' } });
	const choices: Choices = {
		a1: { decision: 'accept', comment: '' },
		a2: { decision: 'accept', comment: '' },
		a3: { decision: 'refuse', comment: '' }
	};

	it('sends the review profile with every accepted FCAF action only', () => {
		const decisions = reviewDecisions({ actions: [fcaf, chat, refused], profile }, choices);
		expect(decisions.map((decision) => decision.profile)).toEqual([
			profile,
			undefined,
			undefined
		]);
	});

	it('sends no profile when the review has none', () => {
		const decisions = reviewDecisions({ actions: [fcaf] }, choices);
		expect(decisions[0]).not.toHaveProperty('profile');
	});

	it('sends the verdict with an accepted FCAF action only', () => {
		const unsupported = action({
			key: 'a1',
			plan: { slot: 'onboarding', fcaf_role: 'fcaf-reset' },
			fcaf_status: 'unsupported',
			note: 'no Deny button'
		});
		const chatWithStatus = action({
			key: 'a2',
			plan: { slot: 'onboarding' },
			fcaf_status: 'passed'
		});
		const refusedWithStatus = action({
			key: 'a3',
			plan: { slot: 'onboarding', fcaf_role: 'fcaf-x' },
			fcaf_status: 'passed'
		});
		const decisions = reviewDecisions(
			{ actions: [unsupported, chatWithStatus, refusedWithStatus], profile },
			choices
		);
		expect(decisions[0].fcaf_result).toEqual({
			status: 'unsupported',
			note: 'no Deny button'
		});
		expect(decisions[1]).not.toHaveProperty('fcaf_result');
		expect(decisions[2]).not.toHaveProperty('fcaf_result');
	});

	it('sends a wallet finding with an accepted FCAF action', () => {
		const finding = action({
			key: 'a1',
			plan: { slot: 'onboarding', fcaf_role: 'fcaf-expect-credential-rejected' },
			fcaf_status: 'finding',
			note: 'Wallet finding: stored the credential'
		});
		const decisions = reviewDecisions({ actions: [finding], profile }, choices);
		expect(decisions[0].fcaf_result?.status).toBe('finding');
	});

	it('sends the replays with an accepted FCAF action only', () => {
		const replays = [
			{ fixture: 's#a', class: 'dc+sd-jwt, haip-vp://', status: 'passed' as const },
			{ fixture: 's#b', class: 'mso_mdoc, openid4vp://', status: 'finding' as const }
		];
		const accepted = action({
			key: 'a1',
			plan: { slot: 'verify-credential', fcaf_role: 'fcaf-engagement-haip-vp' },
			fcaf_status: 'passed',
			fcaf_replays: replays
		});
		const chat = action({ key: 'a2', plan: { slot: 'onboarding' }, fcaf_replays: replays });
		const refused = action({
			key: 'a3',
			plan: { slot: 'onboarding', fcaf_role: 'fcaf-x' },
			fcaf_replays: replays
		});
		const noReplays = action({
			key: 'a4',
			plan: { slot: 'onboarding', fcaf_role: 'fcaf-reset' },
			fcaf_replays: []
		});
		const decisions = reviewDecisions(
			{ actions: [accepted, chat, refused, noReplays], profile },
			{ ...choices, a4: { decision: 'accept', comment: '' } }
		);
		expect(decisions[0].fcaf_replays).toEqual(replays);
		expect(decisions[1]).not.toHaveProperty('fcaf_replays');
		expect(decisions[2]).not.toHaveProperty('fcaf_replays');
		expect(decisions[3]).not.toHaveProperty('fcaf_replays');
	});
});

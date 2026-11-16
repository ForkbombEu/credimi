// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import type { ReviewAction } from './api';

import { blockedBy, canAccept, canSend, type Choices } from './review';

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
		used_by: 0,
		...overrides
	};
}

const onboarding = action({ key: 'a1', name: 'Onboarding' });
const credential = action({ key: 'a2', name: 'Credential', starts_from: ['a1'] });

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

describe('canSend', () => {
	const actions = [onboarding, credential];

	it('needs a decision on every action', () => {
		expect(
			canSend(actions, {
				a1: { decision: 'accept', comment: '' },
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

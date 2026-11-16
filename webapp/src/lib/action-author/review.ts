// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Decision, ReviewAction } from './api';

//

/** The user's choice on one action of the review: no decision yet is null. */
export type Choice = { decision: Decision | null; comment: string };

/** The choices of a review, by candidate key; it holds every action of the review. */
export type Choices = Record<string, Choice>;

/** Lowercase hex SHA-256 of the UTF-8 bytes of text. */
async function sha256Hex(text: string): Promise<string> {
	const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(text));
	return Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, '0')).join(
		''
	);
}

/** Whether the action has a plan: Credimi binds the approval to it. */
export function hasPlan(action: Pick<ReviewAction, 'plan'>): boolean {
	return action.plan !== null && typeof action.plan === 'object' && !Array.isArray(action.plan);
}

/**
 * Whether the user can accept the action: it has a plan and the shown YAML matches
 * candidate_sha256, the digest Credimi's approval signs.
 */
export async function canAccept(action: ReviewAction): Promise<boolean> {
	if (!hasPlan(action)) return false;
	if (typeof action.code !== 'string' || typeof action.candidate_sha256 !== 'string')
		return false;
	try {
		return (await sha256Hex(action.code)) === action.candidate_sha256;
	} catch {
		return false;
	}
}

/**
 * The first action of this review the action starts from that the user has not accepted.
 * A starting action missing from the review was saved by an earlier one.
 */
export function blockedBy(
	action: Pick<ReviewAction, 'starts_from'>,
	choices: Choices
): string | undefined {
	return (action.starts_from ?? []).find(
		(key) => key in choices && choices[key].decision !== 'accept'
	);
}

/** Whether every action has a decision, every accept is unblocked and every change says what. */
export function canSend(actions: ReviewAction[], choices: Choices): boolean {
	return actions.every((action) => {
		const choice = choices[action.key];
		if (!choice || choice.decision === null) return false;
		if (choice.decision === 'change') return choice.comment.trim() !== '';
		if (choice.decision === 'accept') return blockedBy(action, choices) === undefined;
		return true;
	});
}

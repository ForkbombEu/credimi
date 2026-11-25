// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { Decision, PendingReview, ReviewAction, ReviewDecisionItem } from './api';

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

/**
 * The name of the action that action starts from by key: Credimi's starts_from_names,
 * else the name of the review's action of that key, else the key.
 */
export function startingName(
	action: Pick<ReviewAction, 'starts_from_names'>,
	key: string,
	names: Record<string, string>
): string {
	return action.starts_from_names?.[key] || names[key] || key;
}

/**
 * Whether the review can be sent: every action has a decision, except one whose starting
 * action is not accepted (it waits for the next review), at least one action is decided,
 * every accept is unblocked and every change says what.
 */
export function canSend(actions: ReviewAction[], choices: Choices): boolean {
	let decided = false;
	const valid = actions.every((action) => {
		const choice = choices[action.key];
		if (!choice || choice.decision === null) return blockedBy(action, choices) !== undefined;
		decided = true;
		if (choice.decision === 'change') return choice.comment.trim() !== '';
		if (choice.decision === 'accept') return blockedBy(action, choices) === undefined;
		return true;
	});
	return valid && decided;
}

/** The decided actions of a review, in review order: the ones the review sends. */
export function decidedActions(actions: ReviewAction[], choices: Choices): ReviewAction[] {
	return actions.filter((action) => (choices[action.key]?.decision ?? null) !== null);
}

/** Whether the action implements an FCAF role: its plan names one. */
export function isFCAF(action: Pick<ReviewAction, 'plan'>): boolean {
	const role = action.plan?.fcaf_role;
	return typeof role === 'string' && role !== '';
}

/**
 * The decisions the review sends, in review order. Every accepted FCAF action carries the
 * review's profile and the agent's verdict on it, which the approval binds.
 */
export function reviewDecisions(
	review: Pick<PendingReview, 'actions' | 'profile'>,
	choices: Choices
): ReviewDecisionItem[] {
	return decidedActions(review.actions, choices).map((action) => {
		const choice = choices[action.key];
		const decision = choice.decision as Decision;
		return {
			key: action.key,
			name: action.name,
			candidate_sha256: action.candidate_sha256,
			decision,
			comment: decision === 'change' ? choice.comment.trim() : '',
			plan: action.plan,
			...(decision === 'accept' && review.profile && isFCAF(action)
				? { profile: review.profile }
				: {}),
			...(decision === 'accept' && isFCAF(action) && action.fcaf_status
				? { fcaf_result: { status: action.fcaf_status, note: action.note ?? '' } }
				: {}),
			...(decision === 'accept' && isFCAF(action) && action.fcaf_replays?.length
				? { fcaf_replays: action.fcaf_replays }
				: {})
		};
	});
}

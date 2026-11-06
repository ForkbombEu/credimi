// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { PendingReview } from './api';

//

/** Lowercase hex SHA-256 of the UTF-8 bytes of text. */
async function sha256Hex(text: string): Promise<string> {
	const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(text));
	return Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, '0')).join(
		''
	);
}

/** Whether the candidate has a plan: Credimi binds the approval to it. */
export function hasPlan(review: Pick<PendingReview, 'plan'>): boolean {
	return review.plan !== null && typeof review.plan === 'object' && !Array.isArray(review.plan);
}

/**
 * Whether the user can accept the candidate: it has a plan and the shown YAML matches
 * candidate_sha256, the digest Credimi's approval signs.
 */
export async function canAccept(review: PendingReview): Promise<boolean> {
	if (!hasPlan(review)) return false;
	if (typeof review.code !== 'string' || typeof review.candidate_sha256 !== 'string')
		return false;
	try {
		return (await sha256Hex(review.code)) === review.candidate_sha256;
	} catch {
		return false;
	}
}

// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import type { WorkflowStatus } from '$lib/workflows/types';

import { ClientResponseError } from 'pocketbase';
import { err, ok, Result } from 'true-myth/result';

import { pb } from '@/pocketbase';
import { getExceptionMessage } from '@/utils/errors';

//

/** A runner device that can host an authoring session (GET /api/action-author/devices). */
export type AuthorDevice = {
	path: string;
	name: string;
	runner_name: string;
	runner_owner: string;
	model: string;
	type: string;
	platform: string;
	is_online: boolean;
	agent_available: boolean;
	queue_length?: number;
};

/** A build the agent can install (GET /api/action-author/wallet-versions). */
export type BuildOption = {
	identifier: string;
	wallet_name: string;
	organization_name: string;
	version_tag: string;
	platforms: string[];
	source: string;
};

/** The save candidate waiting for the user's decision. */
export type PendingReview = {
	tool_id: string;
	name: string;
	build: string;
	code: string;
	candidate_sha256: string;
	plan: Record<string, unknown> | null;
	/** Path of the action the candidate replaces in place, empty when none. */
	replaces: string;
	/** Why the previous save of this candidate failed, empty when none. */
	error: string;
	/** Pipelines of the caller's organization or published that use the replaced action. */
	used_by: number;
};

/** An authoring session and its state (GET /api/action-author/sessions/{id}). */
export type Session = {
	session_id: string;
	run_id: string;
	chat_workflow_id: string;
	status: WorkflowStatus;
	start_time: string;
	close_time?: string;
	wallet_version_identifier: string;
	platform: string;
	device_id: string;
	device_name: string;
	runner_name: string;
	runner_owner: string;
	model: string;
	live_view: boolean;
	chat_ready: boolean;
	/** Whether the chat workflow still runs; false once it ended, even while the session releases the device. */
	chat_running: boolean;
	closes_at: string;
	pending_review: PendingReview | null;
};

/** A row of GET /api/action-author/sessions?wallet=… */
export type SessionListItem = {
	session_id: string;
	status: WorkflowStatus;
	start_time: string;
	wallet_version_identifier: string;
	device_id: string;
	device_name: string;
};

export type StartedSession = { session_id: string; ticket_id: string; device_id: string };

export type ApiFailure = { status: number; reason: string; message: string };

export type ReviewDecision = {
	author_workflow_id: string;
	tool_id: string;
	candidate_sha256: string;
	plan: Record<string, unknown> | null;
	accepted: boolean;
};

//

const BASE = '/api/action-author';

function toFailure(e: unknown): ApiFailure {
	if (e instanceof ClientResponseError) {
		const response = e.response as
			| { message?: string; error?: { reason?: string } }
			| undefined;
		return {
			status: e.status,
			reason: response?.error?.reason ?? '',
			message: response?.message || e.message
		};
	}
	return { status: 0, reason: '', message: getExceptionMessage(e) };
}

async function call<T>(
	path: string,
	options: { method?: string; body?: unknown; keepalive?: boolean } = {}
): Promise<Result<T, ApiFailure>> {
	try {
		const res = await pb.send<T>(`${BASE}${path}`, {
			method: options.method ?? 'GET',
			body: options.body,
			keepalive: options.keepalive,
			requestKey: null
		});
		return ok(res);
	} catch (e) {
		return err(toFailure(e));
	}
}

export async function listDevices(): Promise<Result<AuthorDevice[], ApiFailure>> {
	const res = await call<{ devices: AuthorDevice[] | null }>('/devices');
	return res.map((r) => r.devices ?? []);
}

export async function listBuilds(): Promise<Result<BuildOption[], ApiFailure>> {
	const res = await call<{ wallet_versions: BuildOption[] | null }>('/wallet-versions');
	return res.map((r) => r.wallet_versions ?? []);
}

export function startSession(
	deviceId: string,
	walletVersionIdentifier: string
): Promise<Result<StartedSession, ApiFailure>> {
	return call<StartedSession>('/sessions', {
		method: 'POST',
		body: { device_id: deviceId, wallet_version_identifier: walletVersionIdentifier }
	});
}

export function getSession(id: string): Promise<Result<Session, ApiFailure>> {
	return call<Session>(`/sessions/${encodeURIComponent(id)}`);
}

export async function listSessions(
	walletPath: string
): Promise<Result<SessionListItem[], ApiFailure>> {
	const params = new URLSearchParams({ wallet: walletPath });
	const res = await call<{ sessions: SessionListItem[] | null }>(
		`/sessions?${params.toString()}`
	);
	return res.map((r) => r.sessions ?? []);
}

export function closeSession(
	id: string,
	options: { keepalive?: boolean } = {}
): Promise<Result<{ session_id: string }, ApiFailure>> {
	return call<{ session_id: string }>(`/sessions/${encodeURIComponent(id)}/close`, {
		method: 'POST',
		keepalive: options.keepalive
	});
}

/** An action author session ID: ActionAuthorSession-<runner id>-<ticket uuid>. */
export const SESSION_ID_PATTERN =
	/^ActionAuthorSession-[a-z0-9]{15}-([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$/;

/** The ticket of a session ID (ActionAuthorSession-<runner id>-<ticket>). */
export function ticketOf(sessionId: string): string {
	const ticket = SESSION_ID_PATTERN.exec(sessionId)?.[1];
	if (ticket === undefined) throw new Error('not an action author session id');
	return ticket;
}

/** Ends the session or its queued ticket while the page unloads; fire and forget. */
export function endOnExit(sessionId: string, deviceId: string, queued: boolean): void {
	if (!queued) {
		void closeSession(sessionId, { keepalive: true });
		return;
	}
	const ticket = ticketOf(sessionId);
	pb.send(
		`/api/pipeline/queue/${encodeURIComponent(ticket)}?device_ids=${encodeURIComponent(deviceId)}`,
		{ method: 'DELETE', keepalive: true, requestKey: null }
	).catch(() => {});
}

export function reviewCandidate(
	decision: ReviewDecision
): Promise<Result<{ accepted: boolean; tool_id: string }, ApiFailure>> {
	return call<{ accepted: boolean; tool_id: string }>('/reviews', {
		method: 'POST',
		body: decision
	});
}

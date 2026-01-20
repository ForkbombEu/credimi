// SPDX-FileCopyrightText: 2025 Forkbomb BV
// SPDX-License-Identifier: AGPL-3.0-or-later

import { pb } from '@/pocketbase';
import type { ScoreboardData } from './types';

/**
 * Fetch scoreboard data for the current user's organization
 * Uses pb.send which includes the auth token automatically
 */
export async function fetchMyResults(): Promise<ScoreboardData> {
	return await pb.send('/api/my/results', {
		method: 'GET'
	});
}

/**
 * Fetch scoreboard data for all organizations (public)
 * Uses pb.send which includes the auth token automatically
 */
export async function fetchAllResults(): Promise<ScoreboardData> {
	return await pb.send('/api/all-results', {
		method: 'GET'
	});
}

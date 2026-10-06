// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import {
	createInCardSession,
	isInCardExpandReady,
	isInCardStill,
	type InCardPhase
} from './in-card-session.svelte.js';

describe('isInCardStill', () => {
	it.each<[InCardPhase, boolean]>([
		['idle', false],
		['aligning', false],
		['still', true],
		['exiting', true]
	])('%s → still=%s', (phase, expected) => {
		expect(isInCardStill(phase)).toBe(expected);
	});
});

describe('isInCardExpandReady', () => {
	it('is true only while editing in still (not aligning / exiting / idle)', () => {
		expect(isInCardExpandReady(true, 'still')).toBe(true);
		expect(isInCardExpandReady(true, 'aligning')).toBe(false);
		expect(isInCardExpandReady(true, 'exiting')).toBe(false);
		expect(isInCardExpandReady(true, 'idle')).toBe(false);
		expect(isInCardExpandReady(false, 'still')).toBe(false);
	});
});

describe('createInCardSession', () => {
	it('starts idle and not still', () => {
		const inCard = createInCardSession();
		expect(inCard.phase).toBe('idle');
		expect(inCard.still).toBe(false);
	});

	it('edit-focus start → aligning (not still yet)', () => {
		const inCard = createInCardSession();
		inCard.noteEnterStart();
		expect(inCard.phase).toBe('aligning');
		expect(inCard.still).toBe(false);
	});

	it('settle while In-card edit → still', () => {
		const inCard = createInCardSession();
		inCard.noteEnterStart();
		inCard.noteEnterSettled(true);
		expect(inCard.phase).toBe('still');
		expect(inCard.still).toBe(true);
	});

	it('settle without In-card edit → idle', () => {
		const inCard = createInCardSession();
		inCard.noteEnterStart();
		inCard.noteEnterSettled(false);
		expect(inCard.phase).toBe('idle');
		expect(inCard.still).toBe(false);
	});

	it('editing ended while still → exiting (still still)', () => {
		const inCard = createInCardSession();
		inCard.noteEnterStart();
		inCard.noteEnterSettled(true);
		inCard.noteEditingEnded();
		expect(inCard.phase).toBe('exiting');
		expect(inCard.still).toBe(true);
	});

	it('noteEditingEnded is a no-op unless phase is still', () => {
		const inCard = createInCardSession();
		inCard.noteEnterStart();
		inCard.noteEditingEnded();
		expect(inCard.phase).toBe('aligning');

		inCard.noteExitComplete();
		inCard.noteEditingEnded();
		expect(inCard.phase).toBe('idle');
	});

	it('noteExitComplete → idle (idempotent)', () => {
		const inCard = createInCardSession();
		inCard.noteEnterStart();
		inCard.noteEnterSettled(true);
		inCard.noteEditingEnded();
		inCard.noteExitComplete();
		expect(inCard.phase).toBe('idle');
		expect(inCard.still).toBe(false);

		inCard.noteExitComplete();
		expect(inCard.phase).toBe('idle');
	});

	it('noteExitComplete while aligning (dismiss before settle) → idle', () => {
		const inCard = createInCardSession();
		inCard.noteEnterStart();
		inCard.noteExitComplete();
		expect(inCard.phase).toBe('idle');
		expect(inCard.still).toBe(false);
	});

	it('new enter start resets from exiting (self-heal)', () => {
		const inCard = createInCardSession();
		inCard.noteEnterStart();
		inCard.noteEnterSettled(true);
		inCard.noteEditingEnded();
		expect(inCard.phase).toBe('exiting');

		inCard.noteEnterStart();
		expect(inCard.phase).toBe('aligning');
		expect(inCard.still).toBe(false);
	});

	it('dispose → idle and ignores further enter transitions', () => {
		const inCard = createInCardSession();
		inCard.noteEnterStart();
		inCard.noteEnterSettled(true);
		inCard.dispose();
		expect(inCard.phase).toBe('idle');
		expect(inCard.still).toBe(false);

		inCard.noteEnterStart();
		expect(inCard.phase).toBe('idle');
		inCard.noteEnterSettled(true);
		expect(inCard.phase).toBe('idle');
		inCard.noteEditingEnded();
		expect(inCard.phase).toBe('idle');
	});

	it('full ADR sequence: aligning → still → exiting → idle', () => {
		const inCard = createInCardSession();
		inCard.noteEnterStart();
		expect(inCard.phase).toBe('aligning');
		inCard.noteEnterSettled(true);
		expect(inCard.phase).toBe('still');
		inCard.noteEditingEnded();
		expect(inCard.phase).toBe('exiting');
		inCard.noteExitComplete();
		expect(inCard.phase).toBe('idle');
	});
});

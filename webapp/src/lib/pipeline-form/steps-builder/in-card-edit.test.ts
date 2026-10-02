// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

import { describe, expect, it } from 'vitest';

import { editingUnit, isInCardEdit, type InCardEditMode } from './in-card-edit.js';

const idle: InCardEditMode = { id: 'idle' };
const manual: InCardEditMode = { id: 'manual' };

describe('isInCardEdit', () => {
	it('is true only for form mode with edit intent', () => {
		expect(isInCardEdit({ id: 'form', intent: 'edit', stepIndex: 0 })).toBe(true);
		expect(isInCardEdit({ id: 'form', intent: 'add' })).toBe(false);
		expect(isInCardEdit(idle)).toBe(false);
		expect(isInCardEdit(manual)).toBe(false);
	});
});

describe('editingUnit', () => {
	it('maps step edits (section defaults to steps)', () => {
		expect(editingUnit({ id: 'form', intent: 'edit', stepIndex: 2 })).toEqual({
			section: 'steps',
			index: 2
		});
		expect(editingUnit({ id: 'form', intent: 'edit', stepIndex: 1, section: 'steps' })).toEqual(
			{
				section: 'steps',
				index: 1
			}
		);
	});

	it('maps follow-up edits', () => {
		expect(
			editingUnit({ id: 'form', intent: 'edit', stepIndex: 0, section: 'follow-ups' })
		).toEqual({ section: 'follow-ups', index: 0 });
	});

	it('is null outside In-card edit or without an index', () => {
		expect(editingUnit({ id: 'form', intent: 'add' })).toBeNull();
		expect(editingUnit({ id: 'form', intent: 'edit' })).toBeNull();
		expect(editingUnit(idle)).toBeNull();
		expect(editingUnit(manual)).toBeNull();
	});
});

// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Plain counter — no $state. Checked from setInterval callbacks
// where Svelte reactivity doesn't apply.
let count = 0;

export const activeSheet = {
	get count() {
		return count;
	},
	open() {
		count++;
	},
	close() {
		if (count > 0) count--;
	}
};

// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Reactive counter — $state so TanStack Query option thunks re-run when
// sheets open/close (used to gate query refetchInterval).
let count = $state(0);

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

// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

export const load = async ({ params }) => {
	const token = params.token;
	return { token };
};

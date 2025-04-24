// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

migrate((app) => {
    const settings = app.settings();
    settings.meta.appName = "Credimi";
    app.save(settings);
});

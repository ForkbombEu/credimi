// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Keep the packaged UI's new-session buttons on the authenticated setup path: sessions start
// through Credimi from /start, never from the chat page.
document.addEventListener('click', (event) => {
  const button = event.target instanceof Element ? event.target.closest('button') : null;
  if (button && (button.getAttribute('aria-label') === 'New session'
      || button.textContent.trim().toLowerCase() === 'start a new session')) {
    event.preventDefault();
    event.stopImmediatePropagation();
    window.location.assign('/start');
  }
}, true);

// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// The workflow owns the proposal. This page supplies only a human's boolean decision.
const reviewTool = 'confirm_wallet_action_save';
const nativeSelector = `[aria-label="Response: ${reviewTool}"]`;
const settled = new Set();
let currentSession = '';
let pending = null;
let card = null;
let submitting = false;
let refreshing = false;

// The card floats over the page instead of sitting in the chat's callback queue: the queue is a
// clipped, animated fold, and a closed or narrow chat panel hid the buttons entirely.
const styles = document.createElement('style');
styles.textContent = `
${nativeSelector} { display: none !important; }
.action-save-review { background: var(--surface-1, Canvas); color: var(--text-1, CanvasText);
  border: 1px solid var(--border, ButtonBorder); border-radius: 6px; padding: 16px;
  position: fixed; right: 24px; bottom: 24px; z-index: 1000; box-sizing: border-box;
  width: min(420px, calc(100vw - 48px)); max-height: calc(100vh - 48px); overflow: auto;
  box-shadow: 0 8px 24px rgb(0 0 0 / .35); display: grid; gap: 12px; font-family: inherit; }
.action-save-review h3, .action-save-review p { margin: 0; }
.action-save-review p { overflow-wrap: anywhere; }
.action-save-review .review-buttons { display: flex; gap: 8px; }
.action-save-review button { font: inherit; color: inherit; background: var(--surface-2, ButtonFace);
  border: 1px solid var(--border, ButtonBorder); border-radius: 6px; padding: 8px 16px;
  cursor: pointer; }
.action-save-review button:focus-visible { outline: 2px solid currentColor; outline-offset: 3px; }
.action-save-review button:disabled { opacity: .6; cursor: wait; }
`;
document.head.append(styles);

function sessionId() {
  return new URLSearchParams(window.location.search).get('s') || '';
}

function statusText(text) {
  const status = card?.querySelector('[role="status"]');
  if (status) status.textContent = text;
}

function disableButtons(disabled) {
  for (const button of card?.querySelectorAll('button') || []) button.disabled = disabled;
}

async function decide(accepted) {
  if (!pending || submitting || currentSession !== sessionId()) return;
  const proposal = pending;
  const session = currentSession;
  submitting = true;
  disableButtons(true);
  statusText(accepted ? 'Saving the reviewed action…' : 'Refusing this candidate…');
  try {
    const response = await fetch('/api/callback-result', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ session_id: session, tool_id: proposal.tool_id, result: accepted }),
      signal: AbortSignal.timeout(15000),
    });
    if (!response.ok) throw new Error('The decision could not be recorded. Check the session and retry.');
    const acknowledgement = await response.json();
    if (!acknowledgement.accepted || acknowledgement.tool_id !== proposal.tool_id) {
      throw new Error('The session did not acknowledge this decision. Refresh and check its state.');
    }
    settled.add(proposal.tool_id);
    if (session !== currentSession) return;
    pending = null;
    statusText(accepted ? 'Accepted. Waiting for the save result…' : 'Not saved. You can request changes in this chat.');
  } catch (error) {
    if (session === currentSession) {
      statusText(error instanceof Error ? error.message : 'Could not record the decision.');
      disableButtons(false);
    }
  } finally {
    submitting = false;
  }
}

function render() {
  const nativeCard = document.querySelector(nativeSelector);
  if (nativeCard) nativeCard.hidden = true;
  if (!pending) return;
  if (card?.dataset.toolId !== pending.tool_id) {
    card?.remove();
    card = document.createElement('section');
    card.className = 'action-save-review';
    card.dataset.toolId = pending.tool_id;
    card.setAttribute('aria-label', 'Review and save action');
    const heading = document.createElement('h3');
    heading.textContent = `Save “${pending.tool_input.name}”?`;
    const details = document.createElement('p');
    details.textContent = `Build: ${pending.tool_input.build}. Both clean-state replays passed. The action will be saved unpublished.`;
    const status = document.createElement('p');
    status.setAttribute('role', 'status');
    status.setAttribute('aria-live', 'polite');
    status.textContent = 'Review the accepted YAML above, then confirm your decision.';
    const buttons = document.createElement('div');
    buttons.className = 'review-buttons';
    for (const [label, accepted] of [['Accept', true], ['Refuse', false]]) {
      const button = document.createElement('button');
      button.type = 'button';
      button.textContent = label;
      button.addEventListener('click', () => decide(accepted));
      buttons.append(button);
    }
    card.append(heading, details, status, buttons);
  }
  if (!card.isConnected) document.body.append(card);
}

async function refresh() {
  if (refreshing) return;
  const session = sessionId();
  if (session !== currentSession) {
    currentSession = session;
    pending = null;
    card?.remove();
    card = null;
  }
  if (!session) return;
  refreshing = true;
  try {
    const response = await fetch(`/api/status/${encodeURIComponent(session)}`, {
      cache: 'no-store', signal: AbortSignal.timeout(5000),
    });
    if (!response.ok) throw new Error('Save confirmation is temporarily unavailable.');
    const status = await response.json();
    if (session !== sessionId()) return;
    pending = (status.pending_callbacks || []).find(
      (item) => item.tool_name === reviewTool && !settled.has(item.tool_id),
    ) || null;
    if (!pending && !submitting) {
      card?.remove();
      card = null;
    }
    render();
    if (pending && !submitting) disableButtons(false);
  } catch {
    if (session === sessionId() && pending) {
      disableButtons(true);
      statusText('Cannot reach the session. Reconnecting before you can confirm…');
    }
  } finally {
    refreshing = false;
  }
}

// Native callback cards may mount after the status response or after a panel is reopened.
new MutationObserver(render).observe(document.body, { childList: true, subtree: true });
window.setInterval(refresh, 1000);
refresh();

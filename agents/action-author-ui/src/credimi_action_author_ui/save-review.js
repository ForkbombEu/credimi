// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// The workflow owns the proposals. This page sends a human's save decision to Credimi, which
// signs the approval and delivers it to the session.
// Written by the start page: the Credimi origin and the user's bearer token, for this tab.
const authStorageKey = 'credimi-action-author';
const settled = new Set();
let currentSession = '';
let pending = null;
let card = null;
let submitting = false;
let refreshing = false;

const hashMismatch = 'The reviewed YAML does not match its hash. Refuse this candidate.';
const noPlan = 'No plan: refuse this candidate.';

function textBlock(className, label, text) {
  const block = document.createElement('pre');
  block.className = className;
  block.tabIndex = 0;
  block.setAttribute('aria-label', label);
  block.textContent = text;
  return block;
}

function paragraph(text) {
  const element = document.createElement('p');
  element.textContent = text;
  return element;
}

// The plan the agent sent with the candidate: Credimi checks it again and binds the approval to
// it, so a candidate without one cannot be accepted.
function hasPlan(input) {
  return input.plan !== null && typeof input.plan === 'object' && !Array.isArray(input.plan);
}

// The save of a candidate whose clean-state replay passed.
const saveReview = {
  label: 'Review and save action',
  intro: 'Review the accepted YAML and the plan below, then confirm your decision.',
  cannotAccept: (input) => (hasPlan(input) ? hashMismatch : noPlan),
  endpoint: '/api/action-author/reviews',
  sending: (accepted) => (accepted ? 'Sending your acceptance to Credimi…' : 'Sending your refusal to Credimi…'),
  done: (accepted) => (accepted ? 'Accepted. Waiting for the save result…' : 'Not saved. You can request changes in this chat.'),
  notPending: 'This review is no longer pending. Reloading…',
  render(target, input) {
    const heading = document.createElement('h3');
    heading.textContent = `Save “${input.name}”?`;
    const details = paragraph(`Build: ${input.build}. The clean-state replay passed. The action will be saved unpublished.`);
    const code = textBlock('review-code', 'Accepted YAML', typeof input.code === 'string' ? input.code : '');
    target.append(heading, details, code);
    // The full plan as JSON: JSON escapes line breaks, so no field can add a line, and nothing
    // is cut.
    if (hasPlan(input)) {
      target.append(paragraph('Plan:'), textBlock('review-code', 'Plan', JSON.stringify(input.plan, null, 2)));
    } else {
      target.append(paragraph(noPlan));
    }
  },
  // Checks the card's YAML against candidate_sha256, the digest Credimi's approval signs, so the
  // user never accepts a candidate other than the one shown, nor one without a plan.
  async canAccept(input) {
    if (!hasPlan(input)) return false;
    if (typeof input.code !== 'string' || typeof input.candidate_sha256 !== 'string') return false;
    try {
      return (await sha256Hex(input.code)) === input.candidate_sha256;
    } catch {
      return false;
    }
  },
  body: (proposal, accepted) => ({
    candidate_sha256: proposal.tool_input.candidate_sha256,
    plan: proposal.tool_input.plan,
    accepted,
  }),
};

// Proposals this page reviews, by the callback tool that waits for them.
const tools = { confirm_wallet_action_save: saveReview };
const nativeSelector = Object.keys(tools).map((name) => `[aria-label="Response: ${name}"]`).join(', ');

// The card floats over the page instead of sitting in the chat's callback queue: the queue is a
// clipped, animated fold, and a closed or narrow chat panel hid the buttons entirely.
const styles = document.createElement('style');
styles.textContent = `
${nativeSelector} { display: none !important; }
.action-review { background: var(--surface-1, Canvas); color: var(--text-1, CanvasText);
  border: 1px solid var(--border, ButtonBorder); border-radius: 6px; padding: 16px;
  position: fixed; right: 24px; bottom: 24px; z-index: 1000; box-sizing: border-box;
  width: min(420px, calc(100vw - 48px)); max-height: calc(100vh - 48px); overflow: auto;
  box-shadow: 0 8px 24px rgb(0 0 0 / .35); display: grid; gap: 12px; font-family: inherit; }
.action-review h3, .action-review p { margin: 0; }
.action-review p { overflow-wrap: anywhere; }
.action-review .review-buttons { display: flex; gap: 8px; }
.action-review button { font: inherit; color: inherit; background: var(--surface-2, ButtonFace);
  border: 1px solid var(--border, ButtonBorder); border-radius: 6px; padding: 8px 16px;
  cursor: pointer; }
.action-review button:focus-visible { outline: 2px solid currentColor; outline-offset: 3px; }
.action-review button:disabled { opacity: .6; cursor: wait; }
.action-review .review-sign-in { display: grid; gap: 8px; }
.action-review label { display: grid; gap: 4px; }
.action-review input { font: inherit; color: inherit; background: var(--surface-2, Field);
  border: 1px solid var(--border, ButtonBorder); border-radius: 6px; padding: 8px;
  box-sizing: border-box; width: 100%; }
.action-idle { background: var(--surface-1, Canvas); color: var(--text-1, CanvasText);
  border: 1px solid var(--border, ButtonBorder); border-radius: 6px; padding: 12px 16px;
  position: fixed; top: 24px; left: 50%; transform: translateX(-50%); z-index: 1001;
  box-sizing: border-box; width: max-content; max-width: calc(100vw - 48px);
  box-shadow: 0 8px 24px rgb(0 0 0 / .35); font-family: inherit; margin: 0; }
.action-review .review-code { margin: 0; padding: 8px; max-height: 40vh; overflow: auto;
  white-space: pre; font: 12px/1.5 ui-monospace, monospace; background: var(--surface-2, Field);
  border: 1px solid var(--border, ButtonBorder); border-radius: 6px; }
`;
document.head.append(styles);

function sessionId() {
  return new URLSearchParams(window.location.search).get('s') || '';
}

function statusText(text) {
  const status = card?.querySelector('[role="status"]');
  if (status) status.textContent = text;
}

// Accept stays disabled until the card's review allows it.
function disableButtons(disabled) {
  for (const button of card?.querySelectorAll('button') || []) {
    button.disabled = disabled || (button.dataset.accept === 'true' && card.dataset.canAccept !== 'true');
  }
}

// Lowercase hex SHA-256 of the UTF-8 bytes of text.
async function sha256Hex(text) {
  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(text));
  return Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, '0')).join('');
}

async function checkAcceptable(target, review, input) {
  let acceptable = false;
  try {
    acceptable = await review.canAccept(input);
  } catch {
    acceptable = false;
  }
  if (card !== target) return;
  target.dataset.canAccept = acceptable ? 'true' : 'false';
  const reason = acceptable ? '' : review.cannotAccept(input);
  if (reason) statusText(reason);
  disableButtons(submitting);
}

function storedAuth() {
  try {
    const auth = JSON.parse(sessionStorage.getItem(authStorageKey) || 'null');
    return {
      origin: typeof auth?.origin === 'string' ? auth.origin : '',
      token: typeof auth?.token === 'string' ? auth.token : '',
    };
  } catch {
    return { origin: '', token: '' };
  }
}

function storeAuth(origin, token) {
  sessionStorage.setItem(authStorageKey, JSON.stringify({ origin, token }));
}

function credimiOrigin(value) {
  const url = new URL(value);
  if (url.protocol !== 'http:' && url.protocol !== 'https:') {
    throw new Error('Credimi URL must use HTTP or HTTPS.');
  }
  if (url.username || url.password || url.search || url.hash) {
    throw new Error('Credimi URL must not contain credentials, query parameters, or a fragment.');
  }
  return url.origin + url.pathname.replace(/\/+$/, '');
}

async function responseBody(response) {
  try {
    return await response.json();
  } catch {
    // An HTTP error may have no JSON body.
    return {};
  }
}

function apiMessage(body, status) {
  return body.message || body.error?.message || `Credimi returned ${status}.`;
}

// The organization namespace the UI is bound to, where the session runs.
async function boundNamespace() {
  const response = await fetch('/api/action-author/namespace', {
    cache: 'no-store', signal: AbortSignal.timeout(5000),
  });
  const body = await responseBody(response);
  if (!response.ok || typeof body.namespace !== 'string' || !body.namespace) {
    throw new Error('This authoring UI is not bound to an organization. Open /start and sign in again.');
  }
  return body.namespace;
}

function labelled(text, input) {
  const label = document.createElement('label');
  label.append(text, input);
  return label;
}

// Ask for a Credimi user API key inside the card; the Credimi URL too when none is known.
function showSignIn(accepted, origin) {
  if (!card) return;
  card.querySelector('.review-sign-in')?.remove();
  const toolId = pending?.tool_id;
  const form = document.createElement('form');
  form.className = 'review-sign-in';
  let urlInput = null;
  if (!origin) {
    urlInput = document.createElement('input');
    urlInput.type = 'url';
    urlInput.required = true;
    urlInput.value = 'http://localhost:8090';
    form.append(labelled('Credimi URL', urlInput));
  }
  const keyInput = document.createElement('input');
  keyInput.type = 'password';
  keyInput.autocomplete = 'off';
  keyInput.required = true;
  const button = document.createElement('button');
  button.type = 'submit';
  button.textContent = 'Sign in to Credimi';
  form.append(labelled('Credimi user API key', keyInput), button);
  form.addEventListener('submit', async (event) => {
    event.preventDefault();
    if (submitting) return;
    submitting = true;
    disableButtons(true);
    statusText('Signing in to Credimi…');
    let signedIn = false;
    try {
      const target = origin || credimiOrigin(urlInput.value);
      const response = await fetch(`${target}/api/apikey/authenticate`, {
        headers: { Accept: 'application/json', 'Credimi-Api-Key': keyInput.value },
        signal: AbortSignal.timeout(15000),
      });
      const body = await responseBody(response);
      if (!response.ok) throw new Error(apiMessage(body, response.status));
      if (typeof body.token !== 'string' || !body.token) {
        throw new Error('Credimi did not return a session token.');
      }
      storeAuth(target, body.token);
      form.remove();
      signedIn = true;
    } catch (error) {
      statusText(error instanceof Error ? error.message : 'Could not sign in to Credimi.');
    } finally {
      keyInput.value = '';
      submitting = false;
      disableButtons(false);
    }
    // Retry the decision only for the review it was made on.
    if (signedIn && pending?.tool_id === toolId) await decide(accepted);
  });
  card.insertBefore(form, card.querySelector('.review-buttons'));
}

async function decide(accepted) {
  if (!pending || submitting || currentSession !== sessionId()) return;
  const proposal = pending;
  const review = tools[proposal.tool_name];
  if (accepted && card?.dataset.canAccept !== 'true') {
    const reason = review.cannotAccept(proposal.tool_input);
    if (reason) statusText(reason);
    return;
  }
  const session = currentSession;
  submitting = true;
  disableButtons(true);
  statusText(review.sending(accepted));
  let reload = false;
  try {
    const auth = storedAuth();
    if (!auth.origin || !auth.token) {
      showSignIn(accepted, auth.origin);
      statusText('Sign in to Credimi to send your decision.');
      return;
    }
    const namespace = await boundNamespace();
    const response = await fetch(`${auth.origin}${review.endpoint}`, {
      method: 'POST',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
        Authorization: `Bearer ${auth.token}`,
      },
      body: JSON.stringify({
        namespace,
        author_workflow_id: session,
        tool_id: proposal.tool_id,
        ...review.body(proposal, accepted),
      }),
      signal: AbortSignal.timeout(30000),
    });
    const body = await responseBody(response);
    if (response.status === 401) {
      storeAuth(auth.origin, '');
      showSignIn(accepted, auth.origin);
      statusText('Your Credimi session expired. Sign in to send your decision.');
      return;
    }
    if (response.status === 409) {
      reload = true;
      statusText(review.notPending);
      return;
    }
    if (response.status === 400) {
      // Credimi refused the proposal: the card stays, so the user can still refuse it.
      statusText(apiMessage(body, response.status));
      return;
    }
    if (!response.ok) throw new Error(apiMessage(body, response.status));
    if (body.accepted !== accepted || body.tool_id !== proposal.tool_id) {
      throw new Error('Credimi did not acknowledge this decision. Refresh and check its state.');
    }
    settled.add(proposal.tool_id);
    if (session !== currentSession) return;
    pending = null;
    statusText(review.done(accepted));
  } catch (error) {
    if (session === currentSession) {
      statusText(error instanceof Error ? error.message : 'Could not send the decision.');
    }
  } finally {
    submitting = false;
    if (session === currentSession && pending) disableButtons(false);
    if (reload) await refresh();
  }
}

function render() {
  for (const nativeCard of document.querySelectorAll(nativeSelector)) nativeCard.hidden = true;
  if (!pending) return;
  if (card?.dataset.toolId !== pending.tool_id) {
    const review = tools[pending.tool_name];
    card?.remove();
    card = document.createElement('section');
    card.className = 'action-review';
    card.dataset.toolId = pending.tool_id;
    card.setAttribute('aria-label', review.label);
    review.render(card, pending.tool_input);
    const status = document.createElement('p');
    status.setAttribute('role', 'status');
    status.setAttribute('aria-live', 'polite');
    status.textContent = review.intro;
    const buttons = document.createElement('div');
    buttons.className = 'review-buttons';
    for (const [label, accepted] of [['Accept', true], ['Refuse', false]]) {
      const button = document.createElement('button');
      button.type = 'button';
      button.textContent = label;
      button.dataset.accept = String(accepted);
      button.addEventListener('click', () => decide(accepted));
      buttons.append(button);
    }
    card.append(status, buttons);
    card.dataset.canAccept = 'pending';
    disableButtons(submitting);
    checkAcceptable(card, review, pending.tool_input);
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
    if (!response.ok) throw new Error('Review is temporarily unavailable.');
    const status = await response.json();
    if (session !== sessionId()) return;
    pending = (status.pending_callbacks || []).find(
      (item) => Object.hasOwn(tools, item.tool_name) && !settled.has(item.tool_id),
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

// The session closes after a quiet idle window to free the device; warn in its last minute.
let idleBanner = null;

function showIdle(secondsLeft) {
  if (!idleBanner) {
    idleBanner = document.createElement('p');
    idleBanner.className = 'action-idle';
    idleBanner.setAttribute('role', 'status');
  }
  idleBanner.textContent = `No activity: this session closes in ${secondsLeft} s to free the device. Send a message to keep it open.`;
  if (!idleBanner.isConnected) document.body.append(idleBanner);
}

function hideIdle() {
  idleBanner?.remove();
}

async function refreshIdle() {
  const session = sessionId();
  if (!session) return hideIdle();
  try {
    const response = await fetch(`/api/action-author/idle/${encodeURIComponent(session)}`, {
      cache: 'no-store', signal: AbortSignal.timeout(5000),
    });
    if (!response.ok) throw new Error('Idle state is unavailable.');
    const { closes_at: closesAt } = await response.json();
    const secondsLeft = Math.floor((Date.parse(closesAt) - Date.now()) / 1000);
    // A closed chat answers no close time; past the close time, nothing is left to warn.
    if (session !== sessionId() || !closesAt || !(secondsLeft >= 0 && secondsLeft < 60)) return hideIdle();
    showIdle(secondsLeft);
  } catch {
    hideIdle();
  }
}

// Native callback cards may mount after the status response or after a panel is reopened.
new MutationObserver(render).observe(document.body, { childList: true, subtree: true });
window.setInterval(refresh, 1000);
window.setInterval(refreshIdle, 10000);
refresh();
refreshIdle();

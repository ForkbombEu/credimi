<!--
SPDX-FileCopyrightText: 2026 Forkbomb BV

SPDX-License-Identifier: AGPL-3.0-or-later
-->

# Changelog

History of the action author agent. Current setup and usage are in [README.md](README.md).
Sections are newest first and dated `DD/MM/YYYY`.

## 29/10/2026

### Fixed

- A Claude reply cut off by the output limit no longer fails the message: its tokens count toward the budget, its unfinished tool call is dropped, and `Continue` resumes with the completed exploration.
- `maestro_inspect_screen` refuses to run before `set_target`.

### Removed

- Masking of input values in tool outputs, replay reports and replies; test PINs are not
  secrets.

## 28/10/2026

### Added

- Native Claude Messages provider. Existing configurations with
  `ACTION_AUTHOR_MODEL_BASE_URL=https://api.anthropic.com/v1/` automatically select native
  Messages instead of Chat Completions. Other compatible providers are unchanged.

### Changed

- Upgraded `temporal-agent-harness` from 0.6.0, pinned by the initial PoC, to 0.8.0. The
  action-author dependency is now upgraded independently of the news agent.
- After upgrading, restart both `make action-author.worker` and `make action-author.ui`
  to load the new package and browser assets. The existing ChatGPT subscription provider,
  device pinning, validation and replay rules are unchanged.

### Verified

- A read-only live harness check exercised native Messages against the retained authoring
  conversation. The second request reused 45,477 cache tokens out of 45,557 total input
  tokens and reported 9,030 cleared observation tokens. The harness UI displayed those counts.
  Neither request called device tools or saved an action. This proves native caching/context
  editing, not a new native-provider physical-device authoring run.
- The initial physical-device acceptance used compatibility mode: a budget-exhausted
  Sonnet 4.6 exploration resumed with Sonnet 5.5 and produced an accepted onboarding
  candidate on its first submission: both clean-state Maestro replays passed (35.4s and 34.5s,
  12 commands each). The continuation used 6 model calls and 273,741/2,408 input/output tokens.
  After explicit human approval, the exact candidate was persisted as an unpublished wallet
  action; stored code, category and build/replay provenance were verified against the candidate.

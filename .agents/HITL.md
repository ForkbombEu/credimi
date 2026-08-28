<!--
SPDX-FileCopyrightText: 2025 Puria Nafisi Azizi
SPDX-FileCopyrightText: 2025 The Forkbomb Company

SPDX-License-Identifier: CC-BY-NC-SA-4.0
-->

# HITL

Human-in-the-loop decision backlog for Credimi agents.

Use this file when an agent finds a convention, architectural rule, dependency contract, validation rule, design rule, or workflow expectation that is missing or ambiguous in `AGENTS.md`.

Do not treat an entry here as approved policy until a human maintainer resolves it and the decision is moved into `AGENTS.md` or another canonical project document.

## Template

```md
### YYYY-MM-DD - Short question

- status: open | resolved | rejected
- owner: human maintainer | agent | unknown
- context:
- question:
- options considered:
- default risk:
- decision:
- follow-up:
```

## Open Questions

### 2026-08-28 - FCAF assertion-review backlog scope and location

- status: resolved
- owner: human maintainer
- context: Merging `origin/main` into `fix/fcaf-improvments` retains main's two generated aggregate pipelines and keeps not-fully-implemented tests in those pipelines. At merge time, the branch tree classified 272 YAML tests as pending and 56 as verifier-blocked under `_implementation`; older counts in `pkg/fcaf/MEMORY.md` were stale.
- question: Should the exact assertion-review backlog be the 328 branch-classified pending/verifier-blocked test IDs, and should those IDs be added to the existing missing-validators document or maintained in a new dedicated file?
- options considered: (1) add the checklist to `FCAF_MISSING_VALIDATORS_AND_ASSERTIONS.md`; (2) create a dedicated assertion-review backlog; (3) list every main test except cases explicitly improved on this branch.
- default risk: Choosing the wrong population can omit tests whose assertions remain placeholders or incorrectly mark implemented tests as needing review.
- decision: Keep exactly the pending and verifier-blocked classifications in a dedicated `config_templates/fcaf/wallet_solution/relying_party/ASSERTION_REVIEW_BACKLOG.md` file because the maintainer will work only from that list.
- follow-up: Remove items from the dedicated backlog only after their assertions and evidence have been reviewed and improved; keep the tests in the aggregate pipeline throughout.

### 2026-07-22 - Login Turnstile gating scope

- status: resolved
- owner: human maintainer
- context: The requested login CAPTCHA must both prevent login content from loading until verification and appear where the current `Log in` button is. The login screen also offers OAuth and WebAuthn authentication paths.
- question: Should Turnstile gate the entire login experience (including OAuth and WebAuthn), or only password login; and should its success reveal the form or merely enable its submit button?
- options considered: (1) CAPTCHA-first gate that reveals all login methods after success; (2) render the existing fields and replace the password submit button with CAPTCHA until success; (3) protect password login only and leave OAuth/WebAuthn unchanged.
- default risk: Choosing the wrong scope can either leave an authentication endpoint unprotected or impose CAPTCHA on login methods that were not intended to require it.
- decision: The existing login experience is visible only after a successful Turnstile challenge. Password login sends the resulting token to the PocketBase API, which verifies it before authentication.
- follow-up: API coverage added. Frontend type checking remains blocked because Bun is unavailable in this environment.

### 2026-08-27 - FCAF direct-validation ownership after precondition removal

- status: resolved
- owner: human maintainer
- context: FCAF pipelines now embed `fcaf-validation` and pass aggregate `pipeline_outputs` directly, while all precondition YAML files were intentionally deleted. Catalog loading, `/api/fcaf/run`, `fcaf run --tests-file`, and the legacy assessment workflow still depended on precondition definitions for pipeline IDs, output decoders, and four shared assertion gates. Many tests also appeared in multiple aggregate or diagnostic pipelines.
- question: Should legacy assessment orchestration be removed in favor of running self-validating pipeline YAML directly, or preserved through a new explicit test-to-pipeline ownership manifest?
- options considered: Remove `/api/fcaf/run`, assessment/precondition workflows, and `--tests-file`; preserve them with a new ownership manifest; infer ownership from embedded `fcaf-validation.test_ids` despite duplicate owners.
- default risk: Inferring one owner from duplicate aggregate pipelines can execute wrong evidence flow; silently retaining legacy paths leaves production entrypoints broken when precondition files are absent.
- decision: Remove `/api/fcaf/run`, assessment and precondition workflows, and `fcaf run --tests-file`. Keep embedded `fcaf-validation`; validators consume aggregate pipeline results directly. Primary product goal is one FCAF pipeline run covering all feasible tests through multiple sequential scenarios and one final aggregate validation, not one wallet interaction reused across incompatible tests.
- follow-up: Completed in current worktree: four assertion gates migrated inline; legacy registrations/types removed; 559 tests assigned exactly once; 112 maintained scenarios generate one deployable aggregate pipeline with one final validation step.

### 2026-07-14 - FCAF trusted-authorities issuer fixture

- status: open
- owner: human maintainer
- context: `WS_RP_IA_MainInteraction__003` requires more than one wallet credential from the requested issuer and proof that each returned credential matches an AKI in the DCQL `trusted_authorities` array. The repository has no documented controlled issuer certificate/AKI fixture or `oid4vp.trusted_authorities_match` validator implementation.
- question: Which issuer action and certificate chain should be the canonical fixture for AKI-based `trusted_authorities` tests, and should its AKI be configured in file-backed pipeline YAML or resolved dynamically from issued credential evidence?
- options considered: Hardcode the current reference issuer AKI; derive AKI dynamically from each issued credential's X.509 chain; add a dedicated mock-issuer credential action with a stable documented certificate chain.
- default risk: A UI-only Maestro success or a hardcoded deployment-specific AKI would mark the test ready without proving the normative issuer-match condition.
- decision:
- follow-up: Implement the multiple-credential issuance flow, AKI request, and `oid4vp.trusted_authorities_match` validator after the canonical issuer fixture is selected.

### 2026-07-13 - FCAF manual pipeline source files

- status: resolved
- owner: human maintainer
- context: FCAF pipeline preconditions fetch pipeline YAML from PocketBase records by `pipeline_id`, but the repository has no documented source-of-truth folder or import path for the pipeline record YAML. The 2026-07-13 manual DCQL work needs concrete Maestro-driven pipeline bodies for `forkbomb-bv-andrea/fcaf-wallet-solution-relying-party-dcql-*`.
- question: Should manual FCAF pipeline YAML templates live under `config_templates/fcaf/wallet_solution/relying_party/pipelines/`, and should a seed/import command be added to publish them into PocketBase pipeline records?
- options considered: Store manual templates beside the FCAF catalog without runtime import; add a first-class pipeline seed/import command; keep pipeline bodies only in live PocketBase state.
- default risk: File-backed templates without an import path can drift from live PocketBase pipeline records, while live-only pipeline records make FCAF catalog changes hard to review and reproduce.
- decision: Do not store these manual precondition implementations in SQLite/PocketBase. Keep reusable standalone Maestro YAML scripts under `config_templates/fcaf/wallet_solution/relying_party/maestro-preconditions/`.
- follow-up: Replace the temporary DCQL verifier deeplink defaults once the exact request-generation endpoint for each DCQL variant is confirmed.

### 2026-07-03 - FCAF canonical source and pipeline inventory

- status: open
- owner: human maintainer
- context: `FCAF_REAL_EXECUTION_PLAN.md` requires exact normative references from the source FCAF markdown and real reusable pipeline preconditions such as `/org-owner/fcaf-wallet-solution-relying-party-pid-sdjwt-presentation-success`. In the current workspace, the catalog still contains placeholder `TODO/` pipeline identifiers, `_implementation/*.md` summaries, and no local definitions for the named FCAF pipeline IDs were found.
- question: Should the implementation proceed by treating the current in-repo `_implementation` markdown and existing Credimi pipeline assets as the temporary source of truth, or is there another canonical local/external source for the exact FCAF markdown and the real pipeline inventory that this implementation must target?
- options considered: Proceed from `_implementation` drafts and placeholder pipeline mappings, then refine later; stop until the canonical source repo and pipeline inventory are provided; implement only the engine/DSL rewrite now and defer catalog/pipeline concretization.
- default risk: If the wrong source of truth is assumed, the catalog will encode incorrect normative references and pipeline bindings, which would make the FCAF graph executor structurally correct but semantically wrong.
- decision:
- follow-up:

### 2026-07-02 - FCAF temporary conversion files

- status: resolved
- owner: human maintainer
- context: The FCAF implementation plan needs per-test draft YAML and per-test implementation notes for the wallet-solution relying-party tests. `AGENTS.md` generally forbids leaving temporary folders in the repository, but the maintainer wants these files colocated with the test YAML during implementation and deleted later.
- question: Where should temporary FCAF conversion and implementation-note files live while implementing the FCAF DSL catalog?
- options considered: Use `/tmp/fcaf-wallet-rp-work` outside the repository; use `config_templates/fcaf/wallet_solution/relying_party/tests/_implementation/` beside the test YAML; create a separate top-level temporary folder.
- default risk: In-repo temporary files can accidentally be committed or treated as permanent catalog files.
- decision: Temporary implementation files may live beside the FCAF test YAML for now, under the test catalog folder, and must be deleted during implementation once consumed.
- follow-up: Implementation agents must keep the temporary folder clearly named and remove it before finalizing production-ready FCAF catalog work unless the maintainer explicitly keeps it.

### 2026-07-02 - FCAF GitNexus override

- status: resolved
- owner: human maintainer
- context: Repository instructions require GitNexus impact checks before editing workflows, activities, registry entries, and other architecture-sensitive code. GitNexus tools were not exposed in this Codex session.
- question: Should FCAF workflow integration wait for GitNexus availability?
- options considered: Pause workflow/registry edits until GitNexus is available; continue only inside isolated `pkg/fcaf`; proceed with workflow integration without GitNexus.
- default risk: Proceeding without GitNexus may miss affected symbols or execution flows that indexed code intelligence would have surfaced.
- decision: The maintainer explicitly instructed to skip GitNexus for this FCAF implementation pass.
- follow-up: Report this override in final summaries for workflow/registry edits.

# FCAF Single-Pipeline Execution Plan

## Goal

Run each FCAF test as one queued mobile pipeline. The pipeline owns the complete
runner and emulator lifecycle from onboarding through final validation.

## Pilot

`WS_RP_IA_Engagement__001`

The pilot pipeline will contain, in order:

1. Reusable wallet onboarding and unlock actions.
2. Reusable PID acquisition actions.
3. Hardcoded beta capture service calls for verifier-session creation.
4. The Maestro presentation action.
5. Capture-session retrieval and evidence persistence.
6. FCAF validators and assertions as terminal steps.

## Composition model

- Keep reusable Maestro actions as independently importable files and stable
  `action_id` references.
- Keep reusable preconditions as repository YAML fragments.
- Expand or compose those fragments into the executable test pipeline before
  importing or starting it.
- The database stores the imported runtime copy; repository YAML remains the
  portable source of truth.
- Do not use `${fixture.*}` URLs in the pilot. Service endpoints must be
  explicit and auditable in the pipeline YAML.

## Runner lifecycle

- Queue exactly one pipeline per FCAF test.
- Do not start preconditions as a separate queued workflow.
- Hold the runner semaphore until the final validator/evidence step completes.
- A failure in an early step must still persist the pipeline result and report.

## Validation and evidence

- Run validators after the capture output is available.
- Preserve validator ID, definition, status, message, and evidence keys.
- Link the source FCAF definition, workflow ID, Credimi execution page, JSON
  details, screenshots, and video evidence from the generated HTML report.

## Rollout

1. Implement and run the pilot for `WS_RP_IA_Engagement__001`.
2. Verify the emulator remains available through validation and evidence steps.
3. Compare the generated evidence with the FCAF scenario and inventory row.
4. Refactor the remaining supported tests to the same composition model.
5. Keep tests requiring TSL, DC API/W3C API, or unavailable mock-verifier
   behavior explicitly marked as not implemented.

## Acceptance criteria

- One FCAF test produces one queued pipeline execution.
- The pipeline contains its reusable preconditions and test actions.
- No `${fixture.*}` URL is used in the pilot.
- The emulator is not released before final validation.
- HTML and JSON reports exist for pass, fail, and blocked outcomes.

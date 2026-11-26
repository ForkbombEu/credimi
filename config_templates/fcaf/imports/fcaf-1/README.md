# FCAF Credimi import bundle

This bundle mirrors the reusable YAML referenced by the FCAF wallet-solution
definitions for `fcaf-1`, the canonified FCAF reference organization.

## Contents

- `wallet/`: reusable Maestro wallet flows used by the FCAF scenarios and by
  `cmd/fcaf-pipeline-gen` (PID issuance and the wallet reset). Every file is
  registered in `wallet-actions.yaml`.
- `wallet-actions.yaml`: portable Wallet actions import manifest. It maps each
  action's name, category, tags, and separate Maestro YAML file to the target
  wallet and version.
- `credential-issuers/`: portable credential issuer records required by FCAF
  scenarios.
- `credentials/`: portable credential definitions grouped by their canonified
  issuer name. The credential's `yaml` field remains the executable source used
  by the `credential-offer` pipeline step.

FCAF validation now runs as one aggregate pipeline generated from
`config_templates/fcaf/wallet_solution/relying_party/scenarios/`. The legacy
per-scenario pipeline exports that used to live under `pipelines/` were
removed; deployable pipeline YAML is generated on demand with
`make fcaf-generate`.

## Import layout

The intended organization-relative paths are:

```text
fcaf-1/wallet/<action-name>.yaml
fcaf-1/credential-issuers/<canonified-issuer-name>.yaml
fcaf-1/credentials/<canonified-issuer-name>/<canonified-credential-name>.yaml
```

These are YAML source copies for import or review. They do not contain local
database records, API keys, wallet APKs, screenshots, or runner state.


The FCAF scenarios additionally share flows that must exist as `wallet_actions`
records before the aggregate pipeline runs:

Every flow below stores a screenshot of the screen it reached before any wait or
assertion that can fail, so a failed step still shows where the Wallet stopped
instead of leaving the step evidence-free. Screenshot names stay inside the
budget `evidence.screenshot_named` checks
(`len(Snakecase(stepID)) + 1 + len(Snakecase(name)) <= 100`); the tightest steps
are named in the flow comments where that budget is small.

- `fcaf-engagement-haip-vp`: strict positive flow, for every request whose FCAF
  expected result is a completed presentation. It fails its step unless the
  consent screen offers `Share` and the Wallet then reports
  `successfully shared`; the `fcaf-engagement-haip-vp-request` screenshot is
  taken first, and `post-share` records the screen after Share, so a refusal or
  error screen is still stored.
- `fcaf-exercise-wallet-generic`: accepts any final screen, so it proves no
  outcome. Only the transaction-data scenarios 154-157 still use it, until
  their requests are rebuilt. It stores `consent` only while the consent screen
  lists `Requested data`, so a test can bind the consent request itself.
- `fcaf-expect-request-rejected`: strict negative flow. The Wallet answers
  with an error screen or the no-match notice, or returns Home; showing consent
  records the Verifier response and fails the Wallet action. `refused-open` is
  stored at launch and `request-refused` at the end, whichever screen the Wallet
  reaches.
- `fcaf-observe-request`: records the Wallet's reaction to a request without
  selecting or sharing a credential. A consent screen is recorded and left
  unanswered; the calling test proves engagement from the Verifier's capture
  (`oid4vp.request_uri_retrieval`), not from this action. `observe-launch`
  records the launch screen, and the unlock keypad and the readiness waits are
  optional, so the flow always reaches its terminal `request-observed`
  screenshot whichever screen the Wallet shows.
- `fcaf-expect-no-matching-document`: the Wallet reports that it holds no
  credential satisfying the request. `no-match-launch` and the terminal
  `no-match-document` bracket the run, and `no-match-notice` is stored only
  while the notice is on screen.
- `fcaf-expect-credential-absent`: the probe for a request pinned to a value
  only a credential of the scenario carries, or none (a PID issued under a
  constraint it fails, a credential the issuer denied, an engagement control).
  It passes only on the no-match notice and fails, without sharing, on any
  other screen, including the consent screen of a Wallet that would present
  that credential. It never shares, so `cmd/fcaf-pipeline-gen` issues no PID
  before it. Status-reference probes use `fcaf-expect-request-rejected`
  instead, so a kept credential reaches the Verifier. `probe-launch`,
  `no-match-notice` and `absence-probe` are all stored before the final
  assertion.
- `fcaf-expect-credential-rejected`: the Wallet accepts a credential offer and
  then refuses to store the credential, showing an error screen or returning to
  Home. It fails on the `View details` success screen that
  `getcredential-generic-credential-without-authentication` requires. `launch`
  and `offer` are stored before the commands that can fail and `outcome` before
  the assertion; `credential-rejected` stays after it, so its presence still
  means no success screen appeared. The names are at most 8 characters, the
  budget the tightest step id leaves.
- `fcaf-dc-api-present`: the Digital Credentials API flow for cases that must
  succeed. There is no link to open, so this taps the presentation page's
  "Present credential" button, completes the consent screen, closes the
  Wallet's result screen so the response reaches the page, and passes only
  when the page reports `Presentation accepted`. It stores `dc-api-page`,
  `dc-api-after-present`, `dc-api-picker` (whichever of Chrome's
  "Do you trust this site with your data?" prompt and the platform picker
  appeared), `dc-api-wallet-consent`, `dc-api-after-share`,
  `dc-api-wallet-result`, `dc-api-wallet-outcome` and
  `dc-api-invocation-outcome` before the assertions that decide, so a stall at
  any of them is evidenced. The second PIN is optional, as in
  `fcaf-engagement-haip-vp`.
- `fcaf-dc-api-expect-rejected`: the Digital Credentials API flow for cases
  the Wallet must refuse. It fails on the consent screen and on an accepted
  presentation; after the Wallet's error screen it presses Back and closes the
  platform picker, so the page reports the outcome to the service. The same
  screenshots are stored before each of those assertions.
- `fcaf-reset-and-onboard`: clears the Wallet's app state and onboards it with
  the PIN `123456`. `cmd/fcaf-pipeline-gen` runs it as the first step of every
  pipeline and as the first entry of `actions` in every scenario's first
  Wallet step, so a presentation can only select credentials its own scenario
  issued. It takes no screenshot, so it never counts as that step's evidence;
  a failure inside it therefore leaves the presentation step without
  screenshots, which is why its failure is a blocked run, not a Wallet finding.
- `fcaf-issue-pid`: the PID issuance `cmd/fcaf-pipeline-gen` injects as an
  `actions` entry of a presentation that needs a credential. It keeps the
  commands of `getcredential-generic-credential-without-authentication` except
  the screenshots, so the step's screenshots evidence the presentation alone.
- `fcaf-engage-credential-offer`: the attestation-provider engagement flow.
  It opens Capture's `/ui/sessions/{id}/engage` page (`engage_url`) in
  Chrome, follows its "Open in wallet" link, accepts the offer whenever the
  Wallet shows it, and screenshots `wallet-invoked` or `wallet-not-invoked`,
  which `oid4vci.offer_engagement` reads. It fails when the Wallet is not
  invoked (after the `wallet-not-invoked` screenshot) and otherwise asserts no
  outcome. `offer-link-presented` is still stored only while the page offers
  the link, and `offer-page` follows unconditionally before the assertion that
  the page loaded.
- `fcaf-exercise-wallet-redirect`: shares, then closes the Wallet's success
  screen so it can follow the `redirect_uri` the Response Endpoint returned.
  It never fails its step; `oid4vp.redirect_uri_visited` decides. Used by the
  callback-redirect and unknown-parameter scenarios. `wallet-open` and
  `wallet-consent` record the screens before the Share tap.
- `fcaf-exercise-wallet-error`: shares, then stores `wallet-error` only while
  the Wallet's error screen is up, which `evidence.screenshot_named` checks.
  Used for the non-JSON, HTTP error and no-redirect Verifier replies.
  `wallet-open` and `wallet-consent` record the screens before the Share tap.
- `fcaf-observe-wallet`: a 20 s wait that leaves the Wallet untouched, so
  `oid4vci.no_request_after` sees any request it sends after the issuer's
  response.

The other FCAF presentation flows (`fcaf-present-*`, `fcaf-submit-*`,
`fcaf-select-and-share-candidate`, `fcaf-user-*` and
`getcredential-generic-credential-without-authentication`) follow the same
rule: a launch or opening screenshot, the request screen, and the screen after
Share are stored before the waits and assertions that can fail. Updating a
flow's `takeScreenshot` commands therefore also requires re-syncing its
`wallet_actions` record (`fcaf sync`) on the Credimi instance.

Every flow the scenarios or `cmd/fcaf-pipeline-gen` run is also a role of the
wallet role contract `config_templates/fcaf/fcaf-roles.yaml`, which describes
it for any wallet. Changing a flow's verdict screenshots, Share taps,
parameters, launched apps or wait also requires updating its role there; the
`pkg/fcaf/walletroles` tests fail until the two agree.

The remaining wallet files are local reusable Maestro helpers. The files are
not database records and contain no instance-specific PocketBase IDs. Importers
must resolve the organization and wallet on the target Credimi instance, create
or update one `wallet_actions` record per manifest entry, and then use the
record's canonical path as `action_id`.

`runFlow.file` is a Maestro filesystem reference, not a Wallet action
reference. A pipeline submitted to a remote runner must either sequence the
referenced helper as a separate `mobile-automation` step or use a standalone
action whose code contains no `runFlow.file` dependencies.

# FCAF Credimi import bundle

This bundle mirrors the reusable YAML currently referenced by the FCAF
wallet-solution relying-party definitions for the canonified organization
`forkbomb-bv-andrea`.

## Contents

- `pipelines/`: 59 FCAF pipeline YAMLs plus the public PID acquisition pipeline
  exported from Credimi. Their filenames are the
  canonified pipeline names after the organization prefix.
- `wallet/`: reusable Maestro wallet flows used by those pipelines:
  `onboarding.yaml`, `unlock-wallet.yaml`, `choose-eudi-wallet.yaml`, the PID
  acquisition flows, and the PID presentation flows.
- `wallet-actions.yaml`: portable Wallet actions import manifest. It maps each
  action's name, category, tags, and separate Maestro YAML file to the target
  wallet and version.
- `credential-issuers/`: portable credential issuer records required by FCAF
  preconditions.
- `credentials/`: portable credential definitions grouped by their canonified
  issuer name. The credential's `yaml` field remains the executable source used
  by the `credential-offer` pipeline step.

The source inventory contains 60 distinct pipeline IDs. The PID precondition
is backed by the public Credimi pipeline:

- `forkbomb-bv-andrea/eudiw-v-06-38-pid-issue-x2-and-verify-x1-issuer-eudiw-dev`
  (exported locally under its public name and under the compatibility filename
  `fcaf-wallet-solution-relying-party-pid-presentations.yaml`; used by both the
  SD-JWT and mdoc PID presentation preconditions)

The remaining pipeline IDs are represented under `pipelines/`.

## Import layout

The intended organization-relative paths are:

```text
forkbomb-bv-andrea/pipelines/<canonified-pipeline-name>.yaml
forkbomb-bv-andrea/wallet/<action-name>.yaml
forkbomb-bv-andrea/credential-issuers/<canonified-issuer-name>.yaml
forkbomb-bv-andrea/credentials/<canonified-issuer-name>/<canonified-credential-name>.yaml
```

These are YAML source copies for import or review. They do not contain local
database records, API keys, wallet APKs, screenshots, or runner state.

The public wallet action exports included here are:

- `onboarding`
- `onboarding-1` (2026.06.38)
- `getcredential-pid-formeu-issuer-eudiw-dev`
- `verifycredential-pid-formeu-issuer-eudiw-dev`
- `unlock-wallet`
- `choose-eudi-wallet`
- `fcaf-engagement-haip-vp`

The remaining wallet files are local reusable Maestro helpers. The files are
not database records and contain no instance-specific PocketBase IDs. Importers
must resolve the organization and wallet on the target Credimi instance, create
or update one `wallet_actions` record per manifest entry, and then use the
record's canonical path as `action_id`.

`runFlow.file` is a Maestro filesystem reference, not a Wallet action
reference. A pipeline submitted to a remote runner must either sequence the
referenced helper as a separate `mobile-automation` step or use a standalone
action whose code contains no `runFlow.file` dependencies.

# FCAF wallet relying-party Maestro preconditions

Standalone Maestro flows for preparing and exercising the reference EUDIW wallet
against FCAF relying-party preconditions.

These files are plain Maestro YAML, not Credimi pipeline records and not
PocketBase seed data.

Default Android app id:

```sh
eu.europa.ec.euidi
```

The public EUDIW Demo 2026.06.38 APK is:

```sh
https://github.com/eu-digital-identity-wallet/eudi-app-android-wallet-ui/releases/download/Wallet%2FDemo_Version%3D2026.06.38-Demo_Build%3D38/2026.06.38-Demo.apk
```

The local APK originally provided at
`/home/puria/Downloads/2026_06_7stl8ywqt3.38-Demo.apk` could not be read by
`aapt` or `zipinfo`, so use the release asset when installing locally.

Run the all-preconditions flow after starting the emulator and installing the
APK:

```sh
emulator -avd credimi -port 5580 -no-boot-anim -no-snapshot -no-snapshot-load -no-snapshot-save -skip-adb-auth -no-metrics -no-location-ui -no-audio
adb -s emulator-5580 install -r /tmp/eudiw-2026.06.38-Demo.apk
maestro --device emulator-5580 test config_templates/fcaf/wallet_solution/relying_party/maestro-preconditions/all-preconditions.yaml
```

Supply every credential offer and verifier request at runtime. Maestro file-level
`env` values override runtime values, so leaf flows intentionally contain no URL
defaults:

```sh
maestro --device emulator-5580 test all-preconditions.yaml \
  -e ISSUER_USERNAME='<pid issuer realm username>' \
  -e ISSUER_PASSWORD='<pid issuer realm password>' \
  -e PID_SDJWT_DEEPLINK_URL='https://credimi.io/api/credential/deeplink?id=forkbomb-bv-andrea/misc-issuer-integration-demo/eudiw-pid-sd-jwt-vc-issuer-backend&redirect=true' \
  -e PID_MDOC_DEEPLINK_URL='https://credimi.io/api/credential/deeplink?id=forkbomb-bv-andrea/misc-issuer-integration-demo/eudiw-pid-mdoc-haip-vci&redirect=true' \
  -e PID_SDJWT_PRESENTATION_URL='<fresh eudi-openid4vp or haip-vp request>' \
  -e PID_MDOC_PRESENTATION_URL='<fresh eudi-openid4vp or haip-vp request>' \
  -e DCQL_CREDENTIAL_SETS_DEEPLINK_URL='<fresh credential_sets request>' \
  -e DCQL_CREDENTIALS_MATCH_DEEPLINK_URL='<fresh credentials-match request>' \
  -e DCQL_NO_MATCHING_CREDENTIALS_DEEPLINK_URL='<fresh no-match request>' \
  -e DCQL_CLAIM_SETS_DEEPLINK_URL='<fresh claim_sets request>' \
  -e HAIP_VP_PRESENTATION_URL='<fresh haip-vp presentation request>' \
  -e DIRECT_POST_JWT_PRESENTATION_URL='<fresh direct_post.jwt PID mdoc request>'
```

The issuer credentials are intentionally not stored in these files. Supply them
at runtime when the PID issuer realm does not already have an authenticated
browser session.

Generate the four DCQL requests from the matching templates under `../pipelines/`.
They POST distinct `dcql_query` payloads to the EUDIW verifier backend; reusing
one public verifier action for all four does not test the four preconditions.

## Coverage

`preconditions.yaml` maps every current FCAF wallet relying-party precondition
to its implementation path.

Maestro-backed preconditions:

- `pipeline.pid.presentation.sdjwt.all-claims`:
  `pipeline-pid-presentation-sdjwt-all-claims.yaml`
- `pipeline.pid.presentation.mdoc.all-claims-elements`:
  `pipeline-pid-presentation-mdoc-all-claims-elements.yaml`
- `pipeline.dcql.credential-sets`: `dcql-credential-sets.yaml`
- `pipeline.dcql.credentials-match`: `dcql-credentials-match.yaml`
- `pipeline.dcql.no-matching-credentials`: `dcql-no-matching-credentials.yaml`
- `pipeline.dcql.claim-sets`: `dcql-claim-sets.yaml`
- `pipeline.wallet.engagement.haip-vp`: `engagement-haip-vp.yaml`
- `pipeline.wallet.metadata.direct-post-jwt`: `metadata-direct-post-jwt.yaml`

Evidence-only preconditions:

- `assertion.pid.presentation.sdjwt.vct-pid`
- `assertion.pid.presentation.sdjwt.required-mandatory-claims-presented`
- `assertion.pid.presentation.mdoc.doc-type-pid`
- `assertion.pid.presentation.mdoc.required-mandatory-elements-presented`

The evidence-only preconditions are implemented by the Credimi FCAF validators
over decoded pipeline output. They have no standalone Maestro action because
the wallet has already produced the evidence before those checks run.

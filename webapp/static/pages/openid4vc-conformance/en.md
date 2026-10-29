---
title: 'OpenID4VCI and OpenID4VP conformance testing'
description: 'Run OpenID4VCI and OpenID4VP conformance tests on every release: OpenID4VCI 1.0, OpenID4VP 1.0, Drafts 23 and 24, in CI. Start a new test.'
lead: 'Check your credential issuer, verifier or wallet against OpenID4VCI 1.0 and OpenID4VP 1.0, Draft 23 and Draft 24, on every release.'
heroCards:
  - role: credential_issuers
    title: 'Credential issuer'
    detail: 'OpenID4VCI 1.0'
  - role: verifiers
    title: 'Verifier'
    detail: 'OpenID4VP 1.0 · Draft 23 · Draft 24'
  - role: pipelines
    title: 'In CI'
    detail: 'run-issuer · run-verifier'
facts:
  - term: 'OpenID4VCI'
    value: '1.0 (issuer and wallet)'
  - term: 'OpenID4VP'
    value: '1.0, Draft 23, Draft 24 (verifier and wallet)'
  - term: 'Suites'
    value: 'OpenID Foundation, WE BUILD / EWC, EUDIW verifier tests'
  - term: 'CI endpoints'
    value: 'run-issuer, run-verifier'
cta:
  title: 'Check your issuer or verifier on every release'
  text: 'Pick an issuer, verifier or wallet on the Hub and run OpenID4VCI or OpenID4VP checks against it.'
  href: '/hub?tab=conformance-checks'
  label: 'Explore conformance checks'
date: 2026-10-23
updatedOn: 2026-10-24
socialCard: /og/openid4vc-conformance.png
socialCardAlt: 'Check OpenID4VCI and OpenID4VP on every release: OpenID4VCI 1.0, OpenID4VP 1.0 and Drafts 23 and 24, run in CI with run-issuer and run-verifier.'
tags:
  - conformance-and-compliance-tools
  - compliance-and-standards
  - conformance-automation
  - testing
  - guide
  - developer-tools
---

OpenID4VCI and OpenID4VP conformance testing tells you whether your credential issuer, verifier or wallet speaks the OpenID for Verifiable Credentials protocols the way the specifications describe. Credimi, the open-source (AGPL-3.0) testing platform built by Forkbomb BV, runs the OpenID Foundation conformance suite together with WE BUILD / EWC conformance and EUDIW verifier tests, and lets you repeat them on every release. This guide explains what is tested, which versions are covered and how to read the results.

## The OpenID Foundation conformance suite

The [OpenID Foundation conformance suite](https://openid.net/certification/about-conformance-suite/) is the reference test for OpenID protocols. It is maintained by the OpenID Foundation and it is the route to OpenID certification. Credimi integrates the suite, so you can run it from the same place as your other tests, keep the results with the rest of your evidence, and repeat the run whenever your implementation changes.

Credimi does not grant OpenID certification, and it is not a conformity assessment body. Certification remains with the OpenID Foundation. What Credimi gives you is a way to run the suite often, early and automatically, so that when you go for certification or join an interoperability event, you already know where you stand.

Each suite has a public page on the Hub that lists its test cases, for example the OpenID Foundation suite for [OpenID4VCI 1.0 issuers](/hub/conformance-checks/openid4vci_issuer/1.0/openid_conformance_suite), [OpenID4VP 1.0 verifiers](/hub/conformance-checks/openid4vp_verifier/1.0/openid_conformance_suite) and [OpenID4VP 1.0 wallets](/hub/conformance-checks/openid4vp_wallet/1.0/openid_conformance_suite).

## WE BUILD / EWC and EUDIW verifier tests alongside it

The OpenID Foundation suite checks the protocol. European wallet ecosystems add their own profiles and expectations on top of it. Credimi also integrates WE BUILD / EWC conformance and the EUDIW verifier tests, and runs EUDI FCAF test cases, so one platform covers the protocol layer and the EUDI layer. If you need the EUDI view in detail, read [FCAF conformance testing for EUDI wallets](/pages/fcaf-conformance).

Credentials in these runs use SD-JWT VC and ISO mDoc PID, the formats the EUDI ecosystem relies on.

## OpenID4VCI and OpenID4VP conformance: what each test exercises

OpenID4VCI (OpenID for Verifiable Credential Issuance, often shortened to OID4VCI) covers how a credential issuer offers and delivers a credential to a wallet. OpenID4VP (OpenID for Verifiable Presentations, also written OID4VP) covers how a verifier asks a wallet for a credential and how the wallet answers.

### OpenID4VCI: from offer to credential

A typical OpenID4VCI run follows the steps the specification defines:

- **Credential offer.** The issuer makes an offer available to the wallet, usually as a QR code or a deeplink.
- **Token request.** The wallet obtains an access token, using the pre-authorized code or, where the issuer uses it, the authorization code flow.
- **Credential request.** The wallet asks for the credential with the access token and a proof of its key.
- **Credential issuance.** The issuer returns the credential, as SD-JWT VC or mDoc.

<figure>
<img src="/pages/openid4vc-conformance/oid4vci-flow.svg" alt="OpenID4VCI sequence between a wallet and a credential issuer: credential offer, token request, credential request and credential in SD-JWT VC or mDoc, with a conformance check for each step" width="1200" height="440" loading="lazy" />
<figcaption>Each OpenID4VCI step between wallet and credential issuer is checked against the expected behaviour.</figcaption>
</figure>

### OpenID4VP: presentation with DCQL and the Digital Credentials API

On the presentation side, a run exercises:

- **Presentation request.** The verifier asks for specific credentials and claims, expressed with DCQL (the Digital Credentials Query Language).
- **Response.** The wallet answers with a `vp_token` that holds the requested presentations.
- **Validation.** The verifier checks what it received.

Credimi also includes scenarios that use the Digital Credentials API, where the request reaches the wallet through the platform rather than a QR code or deeplink.

## Versions side by side

Credimi tests several versions of the specifications at the same time:

- OpenID4VCI 1.0
- OpenID4VP 1.0
- OpenID4VP Draft 23
- OpenID4VP Draft 24

This matters while ecosystems migrate. Some wallets and verifiers already follow OpenID4VP 1.0, others still run on a draft that a pilot or a partner adopted earlier. If you only test the final version, you miss the reason a counterpart on Draft 23 cannot read your request. If you only test a draft, you do not see what breaks when you move to 1.0. Running the versions side by side shows you where behaviour changes, so you can plan the migration instead of discovering it in production.

## Simulated counterpart or a real wallet on a device

A conformance suite usually tests your implementation against a simulated counterpart: a test wallet for your issuer or verifier, or a test issuer or verifier for your wallet. That is the right way to check the protocol itself, because the counterpart behaves exactly as the specification describes.

Real deployments, however, meet real wallets. Credimi pipelines can also drive real wallet apps on Android devices, emulators and iOS simulators with Maestro, and call live issuers and verifiers with StepCI. A run can chain issuer, wallet and verifier end to end in one pipeline, and the same scenario can run against many wallets. You can find live components to test against on [the Hub](/hub), and connect your own devices through a [credimi-runner](https://docs.credimi.io/manual/credimi-runner/).

Use both. The suite tells you whether you are conformant; a real wallet tells you whether you interoperate. The difference is covered in [EUDI wallet interoperability testing](/pages/eudi-wallet-interoperability-testing), and wallet teams can read [test your EUDI wallet against real issuers and verifiers](/pages/eudi-wallet-testing).

<div class="content-quote"><p>The suite tells you whether you are conformant; a real wallet tells you whether you interoperate.</p></div>

## Run it in CI

Conformance is most useful when it runs on every change, not once before an event. Credimi exposes an endpoint per role:

- `run-issuer` for credential issuers
- `run-verifier` for verifiers
- `run-wallet-apk` for wallets, where you upload a fresh APK

For each commit, you can test a temporary issuer or verifier deployment: your CI deploys the build, calls `run-issuer` or `run-verifier` with its address, and the suites run against it. A [GitHub Action](https://github.com/forkbombeu/credimi-test-action) wraps the same calls. You can also schedule runs, for example to catch changes in a counterpart you depend on. Device runs are queued across teams, you can watch the device live during a run, and results come back by API callback or email.

<figure>
<img src="/pages/openid4vc-conformance/oid4vp-ci.svg" alt="Left: a verifier sends an OpenID4VP presentation request with DCQL, the wallet answers with a vp_token and the verifier validates it. Right: a CI lane where a pull request calls run-verifier, which runs the OpenID Foundation and WE BUILD / EWC suites and returns a result" width="1200" height="440" loading="lazy" />
<figcaption>A pull request calls <code>run-verifier</code>, which runs the suites against your temporary deployment and returns a result.</figcaption>
</figure>

The [CI/CD documentation](https://docs.credimi.io/manual/ci-cd/) describes the setup, and [what conformance automation is](/pages/conformance-automation) explains the approach in more depth.

## Reading pass and fail

A pass means the flow completed and matched the expected behaviour for that test case. A failure tells you which step failed and keeps the context around it, so you can see where the implementation diverged from the specification.

Every run keeps its evidence:

- a timestamped Temporal history of every step input and output;
- the protocol messages exchanged, such as offers, requests and responses;
- device video, screenshots and the last frame, and device logs when a wallet is involved;
- SHA-256 hashed FCAF evidence, linked from the FCAF report to each test case.

When a test fails, start from the protocol messages: compare the request or response with what the version you target expects. If the failure only appears on one version, you have likely found a migration difference rather than a bug. Published pipelines feed the public [scoreboard](/scoreboard).

The [conformance documentation](https://docs.credimi.io/manual/conformance/) lists the available checks, and [why Credimi](/pages/why-credimi) compares this approach with other options.

## FAQ

### Is OID4VP the same as OpenID4VP?

Yes. OID4VP is a short name for OpenID for Verifiable Presentations, and OID4VCI is a short name for OpenID for Verifiable Credential Issuance. They refer to the same specifications.

### Do I still need OpenID certification?

If you want OpenID certification, yes. Credimi runs the OpenID Foundation conformance suite, but certification is granted by the OpenID Foundation, not by Credimi. Running the suite in Credimi helps you prepare and keep conformance on every release.

### Which versions can I test?

OpenID4VCI 1.0, and OpenID4VP 1.0, Draft 23 and Draft 24. You can run them side by side.

### Can I test a wallet, not only an issuer or verifier?

Yes. Wallets are tested on Android devices, emulators and iOS simulators, against issuers and verifiers, and in CI by uploading a fresh APK to `run-wallet-apk`.

### Can I self-host Credimi?

Yes. Credimi is open source under AGPL-3.0. You can use it at credimi.io or self-host it from the [source repository](https://github.com/forkbombeu/credimi).

<div class="content-related">
<p class="content-related-title">Read next:</p>
<div class="content-related-grid">
<a href="/pages/eudi-wallet-interoperability-testing"><span>Guide</span><strong>EUDI wallet interoperability testing</strong><em>Read guide ↗</em></a>
<a href="/pages/fcaf-conformance"><span>Guide</span><strong>FCAF conformance testing for EUDI wallets</strong><em>Read guide ↗</em></a>
<a href="/pages/eudi-wallet-testing"><span>Guide</span><strong>Test your EUDI wallet against real issuers and verifiers</strong><em>Read guide ↗</em></a>
</div>
</div>

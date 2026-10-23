---
title: 'eIDAS 2 wallet testing: from obligations to runnable checks'
description: 'eIDAS 2 wallet testing: map the rulebook, run FCAF, OpenID4VCI and OpenID4VP checks on real devices, keep hashed evidence. Start a test.'
lead: 'Go from eIDAS 2 and the EUDI ARF to FCAF test cases you can run on every release, with evidence an assessor can check.'
heroCards:
  - role: conformance_checks
    title: 'Rulebook'
    detail: 'eIDAS 2 · EUDI ARF · FCAF'
  - role: wallets
    title: 'Your wallet'
    detail: 'PID in SD-JWT VC and ISO mDoc'
  - role: pipelines
    title: 'Evidence'
    detail: 'hashed · timestamped'
facts:
  - term: 'FCAF catalog'
    value: '617 test cases in 184 scenarios (20/10/2026)'
  - term: 'Protocols'
    value: 'OpenID4VCI 1.0, OpenID4VP 1.0'
  - term: 'PID formats'
    value: 'SD-JWT VC, ISO mDoc'
  - term: 'Roles'
    value: 'Wallet, issuer, verifier'
cta:
  title: 'Turn your eIDAS 2 obligations into checks'
  text: 'Pick a live issuer, verifier or wallet on the Hub and run your first flow against it.'
  href: '/hub'
  label: 'Explore the Hub'
date: 2026-10-23
updatedOn: 2026-10-23
socialCard: /og/eidas2-wallet-testing.png
socialCardAlt: 'Turn eIDAS 2 obligations into checks you can run: from eIDAS 2 and the EUDI ARF to FCAF test cases and hashed evidence.'
tags:
  - compliance-and-standards
  - legal-and-privacy
  - identity-wallet-integration
  - testing
  - guide
---

eIDAS 2 wallet testing starts with a legal text and ends with a check you can run on every release. This guide shows how the regulation, the EUDI ARF and the EUDI Functional Conformance Assessment Framework (FCAF) turn obligations into test cases, and how you run those test cases on real devices with Credimi. It also shows where two companion tools fit: one to map the rulebook, one to audit trusted lists.

## What eIDAS 2 means for wallet, issuer and verifier teams

The revised eIDAS regulation, usually called eIDAS 2, establishes the European Digital Identity Wallet (EUDI Wallet). The wallet holds person identification data (PID) and can hold electronic attestations of attributes, and it presents them to relying parties that ask for them. Implementing acts add the technical detail that the regulation itself leaves open.

The obligations land differently depending on your role:

- **Wallet providers** must receive and hold a PID, present it to a verifier, and do so in the formats and protocols the framework specifies.
- **Credential issuers**, including PID providers, must issue credentials that real wallets can receive, over OpenID4VCI.
- **Verifiers (relying parties)** must request and accept presentations from many different wallets, over OpenID4VP.

None of these obligations is satisfied by one component alone. A wallet that passes its own tests can still fail against a real issuer, and a verifier can work with one wallet and not with the next. That is why eIDAS 2 wallet testing has two halves: conformance of each component, and interoperability between them. Read [EUDI wallet interoperability testing](/pages/eudi-wallet-interoperability-testing) for the second half.

## From regulation to test cases: the EUDI ARF and FCAF

A regulation says what must happen. It rarely says how to check it. Two layers sit between the legal text and a test you can run.

The **EUDI Architecture and Reference Framework (ARF)** describes how wallets, issuers and verifiers are built and how they talk to each other: the credential formats, the protocols and the flows. Where the regulation says a wallet must present the PID, the ARF says how that presentation works.

The **EUDI FCAF** ([conformance.eudi.dev](https://conformance.eudi.dev/)) turns that architecture into concrete test cases that a wallet, credential issuer or verifier must satisfy. As of 20/10/2026 the catalog holds 617 test cases in 184 scenarios. Each test case is a check with a defined outcome, which is what makes an obligation testable. Working through the FCAF catalog is the most direct route to EUDI ARF readiness.

<figure>
<img src="/pages/eidas2-wallet-testing/obligation-to-check.svg" alt="From the eIDAS 2 regulation through the EUDI ARF and FCAF test cases to a Credimi pipeline run and its evidence" width="1200" height="460" loading="lazy" />
<figcaption>Each layer makes the one above it concrete, until an obligation becomes a run with evidence.</figcaption>
</figure>

## Map the rulebook with the EUDI Trust & Conformance Atlas

Before you test, you need to know what applies to you. The rulebook is spread across laws, implementing acts, European Commission specifications, standards and protocols, and it keeps growing.

The [EUDI Trust & Conformance Atlas](https://atlas.credimi.io/) maps that rulebook. It lists the laws, implementing acts, EC specifications, standards and protocols behind the EUDI Wallet, and links each of them to conformance tests. Use it to answer two questions: which texts apply to your role, and which tests cover them. The answer becomes the scope of your test plan.

## Run the checks on real devices

Credimi is open-source (AGPL-3.0) conformance and interoperability testing for EUDI wallets, credential issuers and verifiers, alone and together. You use it at credimi.io or self-host it.

A Credimi pipeline turns a test case into a run:

- **Maestro** drives real wallet apps on Android devices, emulators and iOS simulators, so the wallet behaves as a user would see it.
- **StepCI** calls live issuers and verifiers, so the protocol exchange is real, not mocked.
- The run is then judged against FCAF and protocol test cases.

The suites cover EUDI FCAF, OpenID4VCI 1.0 for issuers and wallets, and OpenID4VP 1.0 (plus Drafts 23 and 24) for verifiers and wallets. The PID can be exercised in both credential formats the framework uses, SD-JWT VC and ISO mDoc. One pipeline can chain issuer, wallet and verifier end to end, and the same scenario can run against many wallets.

You can connect your own devices, emulators and simulators through a [credimi-runner](https://docs.credimi.io/manual/credimi-runner/). For the detail of each suite, read [FCAF conformance testing for EUDI wallets](/pages/fcaf-conformance) and [OpenID4VCI and OpenID4VP conformance testing](/pages/openid4vc-conformance). If you build a wallet, [test your EUDI wallet against real issuers and verifiers](/pages/eudi-wallet-testing) walks through the wallet side.

<div class="content-quote"><p>An obligation is testable once it has a test case, a run and evidence that points back to both.</p></div>

## Trusted lists: audit them with EUDI Trust Inspector

Trust in the EUDI ecosystem rests on trusted lists: who may issue, who may verify, and which keys belong to them. The lists themselves deserve their own audit.

[EUDI Trust Inspector](https://trust-inspector.credimi.io/) performs an evidence-oriented audit of EUDI and WE BUILD trusted lists: the List of Trusted Lists (LoTL) and TrustedList XML and JSON per ETSI TS 119 612. It reports implemented technical evidence, not a legal conformance decision. Treat its output as input for your own review, not as a verdict.

<figure>
<img src="/pages/eidas2-wallet-testing/three-tools.svg" alt="Three companion tools: EUDI Atlas maps the rulebook, Credimi runs the checks, Trust Inspector audits trusted lists" width="1200" height="400" loading="lazy" />
<figcaption>Map what applies, run what can be run, audit the trust layer separately.</figcaption>
</figure>

## Evidence for assessors

A passing test is only useful if someone else can check it. Every Credimi run keeps:

- a timestamped Temporal history of every step, with its inputs and outputs;
- the protocol messages: offers, requests and responses;
- device video, screenshots and the last frame;
- device logs;
- SHA-256 hashed FCAF evidence.

The FCAF report links each test case to hashed evidence in the run history, so an assessor can follow a result back to the artifacts that produced it. Published pipelines also feed the public [scoreboard](/scoreboard).

Be precise about what this evidence is. Credimi does not certify anything and is not a conformity assessment body. The evidence supports your own preparation and your conversation with an assessor; it does not replace the assessment. See [why Credimi](/pages/why-credimi) for how this compares with other approaches.

## Keep up as implementing acts and specifications change

Implementing acts are adopted over time, and the specifications they reference move with them. OpenID4VP alone appears in Credimi as version 1.0 and as Drafts 23 and 24. Running versions side by side lets you see where a component stands on each, and when it is safe to drop an older one.

The practical answer to a moving target is to test on every release. Credimi exposes CI endpoints per role, `run-wallet-apk`, `run-issuer` and `run-verifier`, plus a [GitHub Action](https://github.com/forkbombeu/credimi-test-action). Per commit, you can upload a fresh wallet APK or test a temporary issuer or verifier deployment. Scheduled runs catch changes on the other side of the flow, and results arrive by API callback or email. The [conformance documentation](https://docs.credimi.io/manual/conformance/) describes the setup.

## FAQ

### Does Credimi certify eIDAS 2 compliance?

No. Credimi is not a conformity assessment body and does not certify anything. It runs conformance and interoperability tests and keeps the evidence, which you can use to prepare for an assessment.

### Where do I see which laws and specifications apply to my role?

In the [EUDI Trust & Conformance Atlas](https://atlas.credimi.io/). It maps laws, implementing acts, EC specifications, standards and protocols, and links each to conformance tests.

### Which credential formats can I test?

SD-JWT VC and ISO mDoc PID, with scenarios that use DCQL and the Digital Credentials API.

### Can I run the checks on my own devices?

Yes. Connect your devices, emulators and simulators through a credimi-runner, or self-host Credimi entirely.

<div class="content-related">
<p class="content-related-title">Read next:</p>
<div class="content-related-grid">
<a href="/pages/fcaf-conformance"><span>Guide</span><strong>FCAF conformance testing for EUDI wallets</strong><em>Read guide ↗</em></a>
<a href="/pages/openid4vc-conformance"><span>Guide</span><strong>OpenID4VCI and OpenID4VP conformance testing</strong><em>Read guide ↗</em></a>
<a href="/pages/eudi-wallet-interoperability-testing"><span>Guide</span><strong>EUDI wallet interoperability testing</strong><em>Read guide ↗</em></a>
</div>
</div>

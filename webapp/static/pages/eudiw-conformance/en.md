---
title: 'EUDIW conformance: certification, FCAF and how to prepare'
description: 'EUDIW conformance explained: the certification rules, what FCAF tests, its maturity stages and roadmap, and how to run FCAF test cases today. Start a new test.'
lead: 'What EUDI Wallet conformance means under the certification rules, what the Functional Conformance Assessment Framework tests, where it stands today, and how to run its test cases before certification.'
heroCards:
  - role: conformance_checks
    title: 'Rules'
    detail: 'CIR (EU) 2024/2981 certification'
  - role: wallets
    title: 'Functional tests'
    detail: 'FCAF · Wallet Solution'
  - role: pipelines
    title: 'Evidence'
    detail: 'runs · reports · hashes'
facts:
  - term: 'Certification rules'
    value: 'CIR (EU) 2024/2981'
  - term: 'FCAF release'
    value: 'v0.0.10, 13/03/2026'
  - term: 'Release candidate'
    value: 'Targeted Q1 2027'
  - term: 'EUDI ARF'
    value: 'v3.0.0, 23/07/2026'
cta:
  title: 'Run FCAF test cases before your assessment'
  text: 'Browse the FCAF test cases on the Hub, then build a pipeline that runs them against your wallet.'
  href: '/hub?tab=conformance-checks'
  label: 'Explore FCAF test cases'
date: 2026-10-24
updatedOn: 2026-10-24
socialCard: /og/eudiw-conformance.png
socialCardAlt: 'Prepare your EUDI wallet for certification: the rules in CIR (EU) 2024/2981, FCAF functional tests on the Wallet Solution, and evidence for every run.'
tags:
  - compliance-and-standards
  - conformance-and-compliance-tools
  - testing
  - guide
  - faq
---

EUDIW conformance is the proof that a European Digital Identity Wallet meets the requirements it is certified against. Under Commission Implementing Regulation (CIR) (EU) 2024/2981, certification covers cybersecurity, privacy and operational assurance, and also requires a conformity assessment of the wallet's functional requirements. The Functional Conformance Assessment Framework (FCAF) is the European Commission's harmonised set of test cases for that functional part.

This page summarises what the rules require, what FCAF covers and does not cover, where it stands today, and how you can run its test cases while it matures. Every date and version on this page links to its source; check them again before you plan around them.

## What does EUDIW conformance mean?

Two kinds of assessment meet in EUDI Wallet certification:

- **Certification of the wallet solution**, set out in [CIR (EU) 2024/2981](https://eur-lex.europa.eu/eli/reg_impl/2024/2981/oj). It covers cybersecurity, privacy and operational assurance.
- **Functional conformance**, which 2024/2981 requires for the functional requirements defined in three further implementing regulations (Annex III of 2024/2981):

<table>
<thead><tr><th>Area</th><th>Implementing regulation</th></tr></thead>
<tbody>
<tr><td>Integrity and core functionalities</td><td><a href="https://eur-lex.europa.eu/eli/reg_impl/2024/2979/oj">CIR (EU) 2024/2979</a></td></tr>
<tr><td>Protocols and interfaces</td><td><a href="https://eur-lex.europa.eu/eli/reg_impl/2024/2982/oj">CIR (EU) 2024/2982</a></td></tr>
<tr><td>Person identification data and electronic attestations of attributes</td><td><a href="https://eur-lex.europa.eu/eli/reg_impl/2024/2977/oj">CIR (EU) 2024/2977</a></td></tr>
</tbody>
</table>

FCAF exists to harmonise that functional conformity assessment, so a result means the same thing for every Member State, laboratory and wallet provider. Source: [FCAF introduction](https://conformance.eudi.dev/latest/fcaf/introduction/).

## What does FCAF test, and what does it not?

According to [Using the FCAF](https://conformance.eudi.dev/latest/fcaf/using-fcaf/), its coverage is limited to functional requirements:

- protocol specifications for interactions with peers in the ecosystem;
- credential formats;
- mandatory user interactions and functionality.

A production-ready, compliant EUDI Wallet must meet many other requirements, which FCAF leaves to other frameworks. Cybersecurity and privacy evaluation are part of certification, but not of FCAF.

FCAF organises tests around a **System Under Test (SUT)**. In its initial phase the only SUT in scope is the **Wallet Solution**. The framework is expected to add PID Providers, Attestation Providers, Relying Parties and infrastructure components later ([SUTs](https://conformance.eudi.dev/latest/fcaf/suts/)). The Wallet Solution tests are grouped by the counterpart the wallet talks to, such as relying parties, PID providers and attestation providers.

Its intended audience today is wallet providers, testing laboratories and conformity assessment bodies (CABs).

<figure>
<img src="/pages/eudiw-conformance/maturity.svg" alt="FCAF maturity stages: submitted, reviewed (beta), release candidate and released baseline, with the baseline test cases at the submitted stage today and a release candidate targeted for the first quarter of 2027" width="1200" height="380" loading="lazy" />
<figcaption>FCAF content moves through four maturity stages. Only a released baseline is meant for regulatory use in certification.</figcaption>
</figure>

## Where does FCAF stand today?

FCAF is published early and refined in public. Content moves through four maturity stages ([FCAF home](https://conformance.eudi.dev/latest/)):

<table>
<thead><tr><th>Stage</th><th>Meaning</th><th>Version pattern</th></tr></thead>
<tbody>
<tr><td>Submitted</td><td>Latest working draft; not yet suitable for implementation. Selected content, such as the baseline test cases, may be used for pilot testing.</td><td>v0.0.x</td></tr>
<tr><td>Reviewed (beta)</td><td>Technically correct and implementable in isolation.</td><td>vX.Y.Z-beta.n</td></tr>
<tr><td>Release candidate</td><td>Consolidated across specifications, profiles and regulatory layers; suitable for validation and pilot use.</td><td>vX.Y.Z-rc.n</td></tr>
<tr><td>Released baseline</td><td>Stable for its defined scope and suitable for regulatory use in certification.</td><td>vX.Y.Z</td></tr>
</tbody>
</table>

And the milestones, as published:

<table>
<thead><tr><th>Item</th><th>Status</th><th>Source</th></tr></thead>
<tbody>
<tr><td>Latest FCAF tag</td><td>v0.0.10, 13/03/2026</td><td><a href="https://github.com/eu-digital-identity-wallet/eudi-doc-functional-conformance-assessment/tags">GitHub tags</a></td></tr>
<tr><td>Baseline test cases and ICS references to ISO/IEC TS 18013-6</td><td>Available at the submitted stage; usable for pilot testing, for example by CABs</td><td><a href="https://conformance.eudi.dev/latest/roadmap/">Roadmap</a></td></tr>
<tr><td>Beta release</td><td>Planned once public feedback on the baseline test cases is processed</td><td><a href="https://conformance.eudi.dev/latest/roadmap/">Roadmap</a></td></tr>
<tr><td>Release candidate</td><td>Goal: first quarter of 2027</td><td><a href="https://conformance.eudi.dev/latest/roadmap/">Roadmap</a></td></tr>
<tr><td>v1.0.0</td><td>Follows the release candidate; the first full-scope baseline suitable for certification. No date published.</td><td><a href="https://conformance.eudi.dev/latest/roadmap/">Roadmap</a></td></tr>
<tr><td>EUDI ARF</td><td>v3.0.0, released 23/07/2026</td><td><a href="https://github.com/eu-digital-identity-wallet/eudi-doc-architecture-and-reference-framework/releases">ARF releases</a></td></tr>
</tbody>
</table>

The roadmap is explicitly indicative and subject to change. Treat the release candidate date as a target, not a commitment, and leave room in your own plan for it to move.

## What is the Implementation Conformance Statement?

Before testing starts, the implementer fills in the **Implementation Conformance Statement (ICS)**: a form declaring which capabilities and profiles the wallet supports. Each test case states the capability or profile conditions under which it applies, so the ICS decides which tests run against your wallet.

The FCAF ICS follows the approach of the ICS in ISO/IEC TS 18013-6:2025 (Annex B): numbered capability statements, a value column for the implementer, and a column stating the constraint the EUDI framework imposes on each answer ([ICS](https://conformance.eudi.dev/latest/fcaf/ics/)).

Prepare it early. Writing the ICS forces you to state which flows, credential formats and profiles you actually support, which is also the list of things you need to test.

## How do you prepare a wallet for FCAF?

While FCAF matures, you can already work against it:

1. **Read the test cases for your wallet.** Start with the baseline test cases on [conformance.eudi.dev](https://conformance.eudi.dev/latest/fcaf/suts/wallet_solution/), and the same test cases on the [Credimi Hub](/hub?tab=conformance-checks), where each has its own page.
2. **Draft your ICS.** Declare the capabilities and profiles you support.
3. **Run the test cases against real counterparts.** Functional tests are about how your wallet talks to issuers and verifiers, so run them against real ones, on real devices.
4. **Keep the evidence.** Store each run with what it was judged on, so you can show a laboratory or CAB what you tested and when.
5. **Track the framework.** Follow the FCAF [content changelog](https://conformance.eudi.dev/latest/fcaf/CHANGELOG/) and re-run when test cases change.

<div class="content-quote"><p>FCAF tells you what to test. Running it on every release tells you whether you still pass.</p></div>

## How does Credimi run FCAF test cases?

Credimi is an open-source platform that runs FCAF test cases inside automated pipelines, hosted at credimi.io or self-hosted.

- **Catalog.** As of 20/10/2026 the Credimi catalog holds 617 FCAF test cases in 184 scenarios, taken from the FCAF Wallet Solution content.
- **Real flows.** Pipelines drive your wallet app on Android devices, emulators and iOS simulators with Maestro, and call live credential issuers and verifiers with StepCI. Each run is then judged against the FCAF and protocol test cases.
- **Evidence.** Every run keeps a timestamped history of each step, the protocol messages, video, screenshots and logs. The FCAF report links each test case to SHA-256 hashed evidence in that history.
- **Report.** Each run produces an FCAF report as PDF and JSON: an executive summary with the evidence hash, then one page per test case with its assertions and normative references. See a [complete example](https://credimi.io/api/files/pipeline_results/lm9cvfbiltat7mp/fcaf_assessment_hiyu281nr4.pdf) (PDF, 21 MB, 617 test cases).
- **Every release.** The `run-wallet-apk` endpoint and the GitHub Action run the same scenarios on every commit.

Credimi does not certify wallets and is not a conformity assessment body. It helps you find failing test cases before an assessment and arrive with a record of what you ran. For the details, read [FCAF conformance testing for EUDI wallets](/pages/fcaf-conformance).

### What does an FCAF test look like in Credimi?

Each FCAF test case is a short YAML file: which wallets it applies to, the normative references it checks, the evidence a pipeline run captured, and the assertions to run on that evidence. This one checks that a PID in SD-JWT VC format carries the `email` claim:

```yaml
id: WS_RP_DM_AddressData_Emailaddress_PID_IETF-sd-jwt-vc_001
applicability:
  credential_format: ietf_sd_jwt_vc
  document_type: pid
evidence:
  pid_sdjwt:
    from: pipeline.pid.presentation.sdjwt.all-claims.outputs.pid_sdjwt
assertions:
  - id: email_present
    validator: sdjwt.claim_present
    input: evidence.pid_sdjwt
    params: { claim: email }
verdict:
  pass_when: all_assertions_pass
```

The assertions use a library of 145 reusable validators for OpenID4VP requests and responses, DCQL, SD-JWT VC claims and key binding, ISO mdoc elements, JOSE and PID rules. Adding a test case usually means writing one such file; the evidence always comes from a real run, never from a fixture. The test definitions are open source in the [Credimi repository](https://github.com/forkbombeu/credimi/tree/main/config_templates/fcaf).

### How does this compare with other ways of testing?

<table>
<thead><tr><th>Approach</th><th>Good for</th><th>Leaves to you</th></tr></thead>
<tbody>
<tr><td>Reading FCAF and testing by hand</td><td>Understanding the test cases; one-off checks</td><td>Repeating the same steps on every release; keeping evidence</td></tr>
<tr><td>Protocol conformance suites (for example the OpenID Foundation suite)</td><td>Checking a protocol implementation against a simulated counterpart; the route to OpenID certification</td><td>FCAF test cases; a real wallet on a real device</td></tr>
<tr><td>Assessment by a laboratory or CAB</td><td>The conformity assessment required for certification</td><td>Finding and fixing failures before the assessment</td></tr>
<tr><td>Credimi pipelines</td><td>Running FCAF and protocol test cases on real devices against live issuers and verifiers, on every release, with evidence</td><td>The formal assessment and certification</td></tr>
</tbody>
</table>

These approaches complement each other: automated runs during development, a formal assessment for certification.

## FAQ

### Is EUDIW conformance the same as certification?

No. Certification under CIR (EU) 2024/2981 covers cybersecurity, privacy and operational assurance, and requires a conformity assessment of functional requirements. Functional conformance is one part of certification, and FCAF is the framework for that part.

### Can FCAF be used for certification today?

Not as a released baseline yet. The current content is at the submitted stage. Its baseline test cases can be used for pilot testing. The release candidate is targeted for the first quarter of 2027, and v1.0.0, which follows it, is the first baseline intended to be suitable for certification.

### Which components does FCAF test?

In its initial phase, only the Wallet Solution. PID Providers, Attestation Providers, Relying Parties and infrastructure components are expected to follow.

### Does FCAF test security?

No. FCAF covers functional requirements: protocols with ecosystem peers, credential formats, and mandatory user interactions and functionality. Cybersecurity is assessed under the certification framework, not by FCAF.

### Does Credimi certify EUDI wallets?

No. Credimi is not a conformity assessment body. It runs FCAF and protocol test cases and keeps the evidence, which you can use to prepare for an assessment.

<div class="content-related">
<p class="content-related-title">Read next:</p>
<div class="content-related-grid">
<a href="/pages/fcaf-conformance"><span>Guide</span><strong>FCAF conformance testing for EUDI wallets</strong><em>Read guide ↗</em></a>
<a href="/pages/eidas2-wallet-testing"><span>Guide</span><strong>eIDAS 2 wallet testing: from obligations to runnable checks</strong><em>Read guide ↗</em></a>
<a href="/pages/eudi-wallet-interoperability-testing"><span>Guide</span><strong>EUDI wallet interoperability testing</strong><em>Read guide ↗</em></a>
</div>
</div>

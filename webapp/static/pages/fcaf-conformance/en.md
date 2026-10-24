---
title: 'FCAF conformance testing for EUDI wallets'
description: 'FCAF conformance for your EUDI wallet: run 617 test cases in 184 scenarios on real devices, with SHA-256 hashed evidence. Start a new test.'
lead: 'Run the 617 test cases of the EUDI Functional Conformance Assessment Framework on real devices, with SHA-256 hashed evidence behind every result.'
heroCards:
  - role: conformance_checks
    title: 'FCAF catalog'
    detail: '617 test cases · 184 scenarios'
  - role: wallets
    title: 'Real wallet'
    detail: 'Android · emulators · iOS simulators'
  - role: pipelines
    title: 'FCAF report'
    detail: 'SHA-256 hashed evidence'
facts:
  - term: 'FCAF catalog'
    value: '617 test cases'
  - term: 'Scenarios'
    value: '184, as of 20/10/2026'
  - term: 'Roles'
    value: 'Wallet, issuer, verifier'
  - term: 'Evidence'
    value: 'SHA-256 hashed'
cta:
  title: 'See where your wallet stands against FCAF'
  text: 'Browse the FCAF test cases on the Hub, then build your first pipeline.'
  href: '/hub/conformance-checks/fcaf/wallet_solution/relying_party'
  label: 'Explore FCAF test cases'
date: 2026-10-23
updatedOn: 2026-10-24
socialCard: /og/fcaf-conformance.png
socialCardAlt: 'Run all 617 FCAF test cases on real devices: 184 scenarios, real flows with Maestro and StepCI, SHA-256 hashed evidence.'
tags:
  - conformance-and-compliance-tools
  - compliance-and-standards
  - conformance-automation
  - testing
  - guide
  - faq
---

FCAF conformance testing checks your EUDI wallet, credential issuer or verifier against the test
cases of the EUDI Functional Conformance Assessment Framework. Credimi runs the FCAF catalog
automatically, on real devices and against live services, and links every result to hashed
evidence you can hand to an assessor.

## What FCAF is

The EUDI Functional Conformance Assessment Framework (FCAF) is the framework used to assess whether
EUDI Wallet components meet the technical requirements of the EUDI Wallet programme. Its official
source is [conformance.eudi.dev](https://conformance.eudi.dev/), where the framework and its test
cases are published.

For you, FCAF answers a concrete question: given the flows your component must support, which test
cases does it pass and which does it fail. That is the core of EUDI wallet conformance, and it
applies to credential issuers and verifiers as well as to wallets.

Because the catalog is shared, a result means the same thing for every team that runs it. A wallet
developer, an issuer developer and a verifier developer can point to the same test case and agree
on what passed and what failed.

## FCAF and the EUDI ARF

FCAF is not the same as the EUDI ARF. The Architecture and Reference Framework (ARF) describes how
the EUDI Wallet ecosystem is built and what each role must support, while FCAF specifies the test
cases used to check conformance against it.

In practice you read the ARF to understand what to build, and you run FCAF to check what you built.
If you are mapping legal obligations to those checks, the [eIDAS 2 wallet testing
guide](/pages/eidas2-wallet-testing) explains how eIDAS 2 and the EUDI ARF lead to FCAF test cases.

## The catalog: 617 test cases in 184 scenarios

As of 20/10/2026, the FCAF catalog holds 617 test cases across 184 scenarios. A scenario groups the
test cases for one flow, such as issuing a credential or presenting it. Running a scenario means
driving that flow once and judging every test case in it against what actually happened.

<figure>
<img src="/pages/fcaf-conformance/catalog-to-report.svg" alt="The FCAF catalog of 617 test cases, grouped into 184 scenarios, runs in a pipeline on a real phone and produces an FCAF report with a pass or fail result per test case" width="1200" height="420" loading="lazy" />
<figcaption>From the FCAF catalog to a report: scenarios run in a pipeline on a real device, and every test case gets a result.</figcaption>
</figure>

Credimi does not run those cases against a mock. A wallet behaves differently on a real device than
in a unit test, and an issuer or verifier behaves differently in production than in a fixture. The
catalog is only useful when it is exercised against the real thing.

## How Credimi runs FCAF automatically

Pipelines drive real wallet apps on Android devices, emulators and iOS simulators with Maestro, and
call live credential issuers and verifiers with StepCI. Each run is then judged against the FCAF and
protocol test cases.

A single pipeline can chain issuer → wallet → verifier end to end: the issuer sends an offer, the
wallet receives and stores the credential, and the verifier requests a presentation. You see how the
components behave together, not only in isolation, and the same scenario can run against many
wallets. To set it up:

- Pick the suites you need: EUDI FCAF, OpenID4VCI 1.0, OpenID4VP 1.0 (plus Drafts 23 and 24), the
  OpenID Foundation conformance suite and WE BUILD / EWC conformance.
- Build a pipeline that chains a real issuance, a wallet interaction and a presentation, with
  credentials in SD-JWT VC or ISO mDoc PID.
- Run it on your own devices through a
  [credimi-runner](https://docs.credimi.io/manual/credimi-runner/), or on shared devices with
  queueing across teams.

Protocol-level checks sit alongside FCAF in the same run. The [OpenID4VCI and OpenID4VP conformance
guide](/pages/openid4vc-conformance) covers those suites in detail, and [testing your EUDI wallet
against real issuers and verifiers](/pages/eudi-wallet-testing) explains the device side.

## FCAF test case pages on the Hub

Every FCAF test case has its own page on the Hub, in the
[FCAF Wallet Solution suite](/hub/conformance-checks/fcaf/wallet_solution/relying_party). You can
read what a test case checks before you run it, and link to it from your own issue tracker or
release notes. For example, [WS_RP_SM_DeviceBinding__007](/hub/conformance-checks/fcaf/wallet_solution/relying_party/WS_RP_SM_DeviceBinding__007)
validates the SD-JWT VC confirmation claim, and the published
[FCAF data model mdoc validation](/hub/pipelines/fcaf-1/fcaf-wallet-relying-party-data-model-mdoc-validation)
pipeline shows how test cases are put together into a run.

The rest of the Hub lists live wallets, credential issuers, verifiers, credentials and verification
use cases, each with a working flow by QR code or deeplink. Those are the counterparts your scenarios
can run against. To check which components work together beyond the FCAF catalog, see [EUDI wallet
interoperability testing](/pages/eudi-wallet-interoperability-testing).

## The FCAF report and its evidence

Each run is recorded in its Temporal history: every step's inputs and outputs, including the
protocol messages exchanged (offers, requests and responses), each with a timestamp. The history is
append-only, so nothing in it is overwritten, and the run can be replayed from it to check that it
leads to the same result. The device video, screenshots, last frame and device logs are stored with
the run.

The FCAF report links each test case to SHA-256 hashed evidence in that run history. An assessor
does not have to take a pass or fail on trust: they can follow the link from a test case to the
artifacts it came from and check that the hash matches.

<figure>
<img src="/pages/fcaf-conformance/scenario-anatomy.svg" alt="One FCAF scenario, PID issuance, expanded into its test cases, each linked to a SHA-256 hashed evidence item in the run history that an assessor can check" width="1200" height="440" loading="lazy" />
<figcaption>Inside one scenario: each test case points to hashed evidence in the Temporal run history.</figcaption>
</figure>

Every run also produces the report as a PDF and as JSON. The PDF opens with an executive summary:
passed, failed, blocked, inconclusive, skipped and not applicable counts, plus the SHA-256 of the
JSON artifact that holds the raw protocol evidence. Each test case then gets its own page with the
objective, preconditions, test scenario, expected results, every assertion with the evidence it
was judged on, and the normative references from FCAF and the implementing regulations.

<figure>
<img src="/screens/fcaf-pdf-report.webp" alt="Three pages of an FCAF PDF report: the cover with result and run identifiers, the executive summary with test counts and the SHA-256 of the evidence, and one test case page with its assertions and normative references" width="1800" height="811" loading="lazy" />
<figcaption>Cover, executive summary and one test case page from a complete-validation run on 23/10/2026: 368 over 617 tests passed, 1,207 pages. <a href="https://credimi.io/api/files/pipeline_results/lm9cvfbiltat7mp/fcaf_assessment_hiyu281nr4.pdf">Open the full report (PDF, 21 MB)</a>.</figcaption>
</figure>

<div class="content-quote"><p>Every FCAF result in Credimi points to the evidence it came from.</p></div>

Published pipelines also feed the public [scoreboard](/scoreboard), so others can see which
scenarios a component runs.

## Preparing for an assessment

Credimi does not certify anything and is not a conformity assessment body. What it gives you is
preparation: you find failing test cases early, fix them, and arrive at an assessment with a record
of runs and the evidence behind each result.

A practical sequence for EUDIW conformance preparation:

1. Read the relevant test cases on the Hub and on [conformance.eudi.dev](https://conformance.eudi.dev/).
2. Build a pipeline that covers the scenarios for your role.
3. Run it until the report is clean, keeping the evidence for each run.
4. Share the report and its hashed evidence with whoever performs the assessment.

## Run FCAF in CI

Conformance that is checked once drifts. The `run-wallet-apk`, `run-issuer` and `run-verifier`
endpoints, plus the [GitHub Action](https://github.com/forkbombeu/credimi-test-action), let you run
the same scenarios on every commit: upload a fresh wallet APK, or test a temporary issuer or
verifier deployment.

You can also schedule runs, watch the device live during a run, and receive results by API
callback or email. The [CI/CD documentation](https://docs.credimi.io/manual/ci-cd/) shows the setup,
and [what conformance automation is](/pages/conformance-automation) explains the approach.

## FAQ

### How many FCAF test cases are there?

617 test cases across 184 scenarios, as of 20/10/2026. Credimi updates this number when the catalog
changes.

### Is FCAF the same as the ARF?

No. The ARF sets out the architecture and requirements for the EUDI Wallet ecosystem, while FCAF is
the framework that specifies the test cases used to check conformance against those requirements.

### Does passing FCAF on Credimi mean my wallet is certified?

No. Credimi is not a conformity assessment body and does not certify wallets, issuers or verifiers.
It runs the FCAF test cases and gives you a report with hashed evidence that you can use to prepare
for an assessment.

### Can I run only some FCAF scenarios?

Yes. You choose the suites and scenarios that go into your pipeline, so you can start with the
flows for your role and add more as you go. Credimi is open source under AGPL-3.0; you can use it
at credimi.io or [self-host it](https://docs.credimi.io/software-architecture/docker-deployment/).

<div class="content-related">
<p class="content-related-title">Read next:</p>
<div class="content-related-grid">
<a href="/pages/eudi-wallet-interoperability-testing"><span>Guide</span><strong>EUDI wallet interoperability testing</strong><em>Read guide ↗</em></a>
<a href="/pages/eudi-wallet-testing"><span>Guide</span><strong>Test your EUDI wallet against real issuers and verifiers</strong><em>Read guide ↗</em></a>
<a href="/pages/openid4vc-conformance"><span>Guide</span><strong>OpenID4VCI and OpenID4VP conformance testing</strong><em>Read guide ↗</em></a>
</div>
</div>

---
title: 'Why Credimi'
description: 'How Credimi compares with BrowserStack, Maestro Cloud, Sauce Labs, Firebase Test Lab, AWS Device Farm and standalone conformance suites when you build for EUDI and eIDAS 2.'
date: 2026-10-20
updatedOn: 2026-10-20
socialCard: /pages/why-credimi/hero.png
tags:
  - solutions
  - conformance-and-compliance-tools
  - compliance-and-standards
  - identity-wallet-integration
  - conformance-automation
  - testing
---

<dl class="content-facts not-prose">
<div><dt>Tests</dt><dd>Wallets, credential issuers, verifiers</dd></div>
<div><dt>Frameworks</dt><dd>EUDI FCAF, OpenID4VCI, OpenID4VP</dd></div>
<div><dt>Runs on</dt><dd>Android devices, emulators, iOS simulators</dd></div>
<div><dt>Deployment</dt><dd>credimi.io or self-hosted, AGPL-3.0</dd></div>
</dl>

## Device clouds test apps. Credimi tests whole identity ecosystems.

![A device cloud tests one app on many phones; Credimi tests issuer, wallet and verifier together](/pages/why-credimi/hero.svg)

BrowserStack, Maestro Cloud, Sauce Labs, Firebase Test Lab and AWS Device Farm are good tools for one question:
_does my app work on this device?_

A European Digital Identity Wallet is never tested alone. It succeeds only when a **credential issuer**, a **wallet** and a
**verifier / relying party** complete a flow together, under OpenID4VCI, OpenID4VP, SD-JWT VC, mDoc and the EUDI ARF, with
evidence an assessor can trust.

Credimi was built for that question:

> _Does my wallet, issuer or verifier conform, and does it interoperate with the rest of the EUDI ecosystem?_

---

## At a glance:

| | **Credimi** | Device clouds | Maestro Cloud | Google / AWS labs | Conformance suites |
|---|:---:|:---:|:---:|:---:|:---:|
| Built for EUDI and eIDAS 2 | ✓ | — | — | — | ◐ |
| Tests issuer, wallet **and** verifier in one run | ✓ | — | — | — | — |
| Mobile wallet UI automation (Maestro) | ✓ | ✓ | ✓ | ◐ | — |
| Backend protocol conformance (OpenID4VCI, OpenID4VP) | ✓ | — | — | — | ✓ |
| EUDI FCAF test cases built in | ✓ | — | — | — | ◐ |
| Several conformance suites behind one interface | ✓ | — | — | — | — |
| Public hub of live issuers, verifiers and wallets | ✓ | — | — | — | — |
| Audit-grade evidence (timeline, hashes, video, logs) | ✓ | ◐ | ◐ | ◐ | ◐ |
| Runs on your own devices, emulators and network | ✓ | ◐ | — | — | ✓ |
| Open source (AGPL-3.0) | ✓ | — | ◐ | — | ◐ |

✓ core capability · ◐ partial or with custom work · — not part of the product

Device clouds: BrowserStack, Sauce Labs, LambdaTest. Maestro Cloud: also DeviceCloud. Google / AWS labs: Firebase Test Lab, AWS Device Farm. Conformance suites: OpenID Foundation, EC Interoperability Test Bed, WE BUILD.

---

## The difference: Credimi tests between components

Most testing platforms see one system under test. Credimi sees the flow between them.

![One Credimi pipeline drives issuer, wallet and verifier, validates the run and keeps the evidence](/pages/why-credimi/flow.svg)

One Credimi pipeline can:

1. ask a live **issuer** for a fresh credential offer (StepCI),
2. pass the deeplink to a **wallet** running on a phone, emulator or iOS simulator, and accept the credential (Maestro),
3. ask a **verifier** for a presentation request and complete it in the same wallet,
4. validate what happened against **FCAF** and protocol test cases, and
5. store every input, output and artifact as reviewable evidence.

<blockquote class="content-quote not-prose"><p>A device cloud can run one step of an EUDI flow. Credimi runs the whole journey, issuer to wallet to verifier, and judges it.</p></blockquote>

---

## Role coverage matrix:

Who can you test, and against whom?

| Component under test | Credimi | Device clouds | Maestro Cloud | Standalone conformance suites |
|---|:---:|:---:|:---:|:---:|
| **Wallet** (Android, iOS) – UI flows | ✓ | ✓ | ✓ | — |
| **Wallet** – protocol conformance | ✓ | — | — | ✓ |
| **Credential issuer** – protocol conformance | ✓ | — | — | ✓ |
| **Verifier / relying party** – protocol conformance | ✓ | — | — | ✓ |
| Wallet ⇄ third-party issuers | ✓ | ◐ | ◐ | — |
| Wallet ⇄ third-party verifiers | ✓ | ◐ | ◐ | — |
| Issuer → wallet → verifier, end to end | ✓ | — | — | — |
| Same scenario against many wallets | ✓ | ◐ | ◐ | — |

◐ on device clouds means you can script it yourself, but you also have to build and host the issuer and verifier side, the
deeplink plumbing and the assertions.

---

## Standards and conformance coverage:

Credimi does not replace the reference suites. It **runs them for you**, side by side, and adds what they lack: real
wallets on real devices.

| Standard / framework | Credimi | Device clouds | Maestro Cloud |
|---|:---:|:---:|:---:|
| OpenID4VCI 1.0 – issuer and wallet | ✓ | — | — |
| OpenID4VP 1.0, Draft 23, Draft 24 – verifier and wallet | ✓ | — | — |
| OpenID Foundation conformance suite | ✓ integrated | — | — |
| WE BUILD / EWC conformance | ✓ integrated | — | — |
| EUDIW verifier tests | ✓ integrated | — | — |
| **EUDI FCAF** – wallet solution, relying party role | ✓ **617 test cases, 184 scenarios** | — | — |
| SD-JWT VC and ISO mDoc PID | ✓ | — | — |
| DCQL and Digital Credentials API scenarios | ✓ | — | — |
| vLEI | ✓ | — | — |

Counts reflect the catalog shipped in Credimi on 20/10/2026. New suites plug into the same pipelines, results pages and
evidence model.

<aside class="content-cta not-prose">
<div>
<p class="content-cta-eyebrow">Try it now</p>
<p class="content-cta-title">Get a live credential offer into your wallet in under a minute.</p>
<p class="content-cta-text">Every issuer and verifier on the Hub exposes a working flow. No account needed.</p>
</div>
<a class="content-cta-button" href="/hub">Explore the Hub</a>
</aside>

---

## FCAF: test cases that come with their evidence

The EUDI **Functional Conformance Assessment Framework (FCAF)** describes what an EUDI wallet solution must do. Credimi runs
FCAF test cases inside its pipelines:

- every scenario produces named evidence (credential offers, presentation requests, responses, screenshots),
- one final validation step judges all catalog test cases against that evidence,
- the full report is stored with the run, and each evidence item gets a SHA-256 hash and a pointer into the run's Temporal
  history.

You get a report you can hand to an assessor, and every claim in it points back to what actually happened.

![Sample FCAF report: each test case links to hashed evidence in the run history](/pages/why-credimi/fcaf-report.svg)

---

## Evidence and auditability matrix:

| Artifact per run | Credimi | Device clouds | Maestro Cloud |
|---|:---:|:---:|:---:|
| Device video | ✓ | ✓ | ✓ |
| Screenshots and last frame | ✓ | ✓ | ✓ |
| Device logs (Logcat / console) | ✓ | ✓ | ◐ |
| Timestamped history of every step input and output | ✓ Temporal | — | — |
| Protocol messages (offers, requests, responses) | ✓ | — | — |
| Hashed evidence index per test case | ✓ | — | — |
| FCAF / conformance report attached to the run | ✓ | — | — |
| Public scoreboard of pipeline results | ✓ | — | — |

---

## Automation and CI:

| | Credimi | Device clouds | Maestro Cloud |
|---|:---:|:---:|:---:|
| Run from a pull request (GitHub Action) | ✓ | ✓ | ✓ |
| Upload a fresh wallet APK per commit | ✓ | ✓ | ✓ |
| Test a temporary issuer or verifier deployment per commit | ✓ | — | — |
| Scheduled runs | ✓ | ◐ | ◐ |
| Device queueing across teams | ✓ | ✓ | ✓ |
| Live view of the device during a run | ✓ | ✓ | ◐ |
| Results by API callback or email | ✓ | ◐ | ◐ |

CI endpoints exist for each role: `run-wallet-apk`, `run-issuer` and `run-verifier`. A wallet team, an issuer team and a
verifier team can each gate their own releases on the same shared scenarios.

---

## Your devices, your network, your data

![Your own devices, emulators and simulators connect to Credimi through a credimi-runner](/pages/why-credimi/runners.svg)

Identity test data is sensitive, and many issuers and verifiers live behind VPNs or in pre-production networks.

- **Credimi runners** connect your own Android phones, emulators, Redroid instances or iOS simulators to the platform.
- Runners can be **private** to your organization or **published** for others to use.
- Each organization runs in its own isolated Temporal namespace.
- Credimi is **open source (AGPL-3.0)**, so you can inspect it or host it yourself.

Device clouds run on their hardware, in their regions. That suits generic apps. It is a harder conversation when the payload
is a Person Identification Data credential.

---

## The Hub: an interoperability network, not a test lab

![The Credimi Hub connects published issuers, wallets and verifiers](/pages/why-credimi/hub.svg)

Every issuer, verifier, wallet and credential published on the [Credimi Hub](https://credimi.io/hub) comes with a live
integration:

- open a credential page and get a fresh credential offer as a QR code or deeplink,
- open a verification use case and get a fresh presentation request,
- reuse the same entry, without changes, as a step in your automated pipelines.

When you test on Credimi, you test against the real ecosystem, not mocks.

<blockquote class="content-quote not-prose"><p>When you publish on the Hub, every wallet team in the ecosystem can test against you.</p></blockquote>

---

## When to use what

Credimi works **alongside** device clouds.

| You need to… | Best fit |
|---|---|
| Check that a shopping app renders on 3,000 device models | BrowserStack, Sauce Labs, LambdaTest |
| Run a large Maestro regression suite in parallel on emulators | Maestro Cloud, DeviceCloud |
| Run instrumentation tests on Google-hosted devices | Firebase Test Lab |
| Check one issuer against the OpenID reference suite, once | OpenID Foundation conformance suite |
| Prove an EUDI wallet, issuer or verifier conforms **and** interoperates, with evidence | **Credimi** |
| Gate releases on wallet × issuer × verifier flows in CI | **Credimi** |
| Prepare for FCAF assessment and EUDI ARF readiness | **Credimi** |

---

## Choose Credimi when

- you build an **EUDI wallet** and need proof it works with real issuers and verifiers, not only on many phones,
- you run a **credential issuer** or **verifier** and want to know which wallets actually complete your flows,
- you prepare for **eIDAS 2 / EUDI ARF** conformance and need FCAF-aligned evidence,
- you are tired of keeping three tools, two scripts and a spreadsheet in sync to answer _"does it interoperate?"_

---

<aside class="content-cta not-prose">
<div>
<p class="content-cta-eyebrow">Credimi</p>
<p class="content-cta-title">Ready to prove your EUDI solution interoperates?</p>
<p class="content-cta-text">Run conformance checks and wallet × issuer × verifier pipelines on your own components. Building a pilot or preparing an FCAF assessment? <a href="mailto:credimi@forkbomb.eu">Talk to us</a>.</p>
</div>
<a class="content-cta-button" href="/my/tests/new">Start a new test</a>
</aside>

<section class="content-related not-prose">
<p class="content-related-title">Related resources:</p>
<div class="content-related-grid">
<a href="/pages/documentation/eudi-wallet-regulatory-map"><span>Guide</span><strong>EUDI Wallet rulebook and regulatory map</strong><em>Read guide ↗</em></a>
<a href="/pages/conformance-automation"><span>Article</span><strong>What is conformance automation</strong><em>Read article ↗</em></a>
<a href="https://docs.credimi.io/manual/testing-automation/build-and-run-pipelines/"><span>Docs</span><strong>Build and run wallet × issuer × verifier pipelines</strong><em>Read docs ↗</em></a>
</div>
</section>

<small>Competitor capabilities are summarized from public product pages as of 20/10/2026 and grouped by product category.
◐ means a capability is possible with custom work or a partial feature. Product names are trademarks of their owners.</small>

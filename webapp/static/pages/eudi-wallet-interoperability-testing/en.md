---
title: 'EUDI wallet interoperability testing'
description: 'EUDI wallet interoperability testing for wallets, issuers and verifiers: try real flows on the Hub, then automate them in a pipeline. Explore the Hub.'
lead: 'Find out which wallets, credential issuers and verifiers actually work together: try real flows on the Hub, then run the same flows in a pipeline on every commit.'
heroCards:
  - role: credential_issuers
    title: 'Live issuers'
    detail: 'OpenID4VCI · SD-JWT VC · mDoc'
  - role: wallets
    title: 'Wallets'
    detail: 'Android · iOS · emulators'
  - role: verifiers
    title: 'Live verifiers'
    detail: 'OpenID4VP · DCQL'
facts:
  - term: 'Roles'
    value: 'Wallet, credential issuer, verifier'
  - term: 'Protocols'
    value: 'OpenID4VCI 1.0, OpenID4VP 1.0, Drafts 23 and 24'
  - term: 'Formats'
    value: 'SD-JWT VC, ISO mDoc'
  - term: 'Automation'
    value: 'StepCI and Maestro in one pipeline'
cta:
  title: 'Find your counterparts'
  text: 'Open a live issuer or verifier on the Hub and try the flow with your wallet.'
  href: '/hub'
  label: 'Explore the Hub'
date: 2026-10-23
updatedOn: 2026-10-24
socialCard: /og/eudi-wallet-interoperability-testing.png
socialCardAlt: 'Find out which EUDI wallets, issuers and verifiers work together: wallets on Android and iOS, issuers over OpenID4VCI with SD-JWT VC and mDoc, verifiers over OpenID4VP and DCQL.'
tags:
  - testing
  - identity-wallet-integration
  - conformance-and-compliance-tools
  - compliance-and-standards
  - guide
  - faq
---

EUDI wallet interoperability testing tells you whether your wallet, credential issuer or verifier works with the components it will meet outside your test lab, not only with its own test suite. Credimi is open-source (AGPL-3.0) conformance and interoperability testing for all three roles, alone and together. You can start by hand on the [Hub](/hub) and then move the same flow into a pipeline that runs per commit or on a schedule.

## Why conformance alone does not prove interoperability

A conformance suite checks your component against a reference. The counterpart is usually simulated: a test wallet, a test issuer or a test verifier that the suite controls. Passing tells you that your component follows the specification as that suite reads it.

Interoperability asks a different question: does the flow complete between your component and someone else's? A component can pass its own suite against a simulated counterpart and still fail with a real wallet. For EUDIW interop, what counts is the flow between real components: a live credential issuer, a wallet app on a device and a live verifier.

Credimi covers both sides. It integrates the OpenID Foundation conformance suite, WE BUILD / EWC conformance and the EUDIW verifier tests, and judges runs against EUDI FCAF and protocol test cases. It also chains real components in one pipeline. For the conformance side, read [OpenID4VCI and OpenID4VP conformance testing](/pages/openid4vc-conformance) and [FCAF conformance testing for EUDI wallets](/pages/fcaf-conformance).

## Three roles, and who tests against whom

The EUDI ecosystem has three roles, and each one tests against the other two:

- **Wallet developer.** You test against third-party credential issuers and verifiers that already run on the Hub, without building your own. Issuance runs over OpenID4VCI with SD-JWT VC and mDoc credentials, presentation over OpenID4VP. The guide [Test your EUDI wallet against real issuers and verifiers](/pages/eudi-wallet-testing) covers this role in detail.
- **Issuer developer.** You check how your credential offer works with different wallets.
- **Verifier developer.** You test how different wallets answer your presentation request, including scenarios with DCQL and the Digital Credentials API.

In each case, the component you own is fixed and the counterparts change. That is why the same scenario has to run against more than one wallet, issuer or verifier.

## Start by hand on the Hub

The [Hub](/hub) is public. It lists live wallets, credential issuers, verifiers, credentials and verification use cases, and each entry exposes a working flow through a QR code or a deeplink. Organizations that publish there include wwwallet, iGrant.io, Validated ID, Hovi, Findynet Cooperative, ShareID and UAB Odifis.

A complete flow published as a pipeline looks like [EUDIW issuance (PID SD-JWT) and verification](/hub/pipelines/forkbomb-bv-andrea/eudiw-issuance-pid-sd-jwt-and-verification): one issuer, one wallet on a device and one verifier, with the result of every run on the [scoreboard](/scoreboard).

A manual interoperability check takes four steps:

1. Open a credential or verification page on the Hub.
2. Generate the QR code or deeplink.
3. Continue in the target wallet.
4. Compare the outcomes across the components involved.

This is the quickest way to see whether two components work together at all. The documentation page [Try interop flows](https://docs.credimi.io/manual/conformance/try-interop-flows/) walks through it.

## Move the flow to automation

A manual check tells you about one moment. To repeat it, keep the StepCI integration that calls the issuer or verifier, add Maestro actions for what the wallet does on screen, and compose them in a pipeline. Maestro drives real wallet apps on Android devices, emulators and iOS simulators; StepCI calls live issuers and verifiers. Your own devices, emulators and simulators connect through a [credimi-runner](https://docs.credimi.io/manual/credimi-runner/).

<figure>
<img src="/pages/eudi-wallet-interoperability-testing/manual-to-pipeline.svg" alt="A manual interoperability flow on the Hub, from Hub page to QR code or deeplink to wallet to comparison, moved to automation as a pipeline of StepCI issuer, Maestro wallet and StepCI verifier steps that produces evidence" width="1200" height="440" loading="lazy" />
<figcaption>The steps you take by hand on the Hub become StepCI and Maestro steps in one pipeline.</figcaption>
</figure>

Once the pipeline works, connect it to CI. Each role has its own endpoint, `run-wallet-apk`, `run-issuer` and `run-verifier`, and there is a [GitHub Action](https://github.com/forkbombeu/credimi-test-action). Per commit you can upload a fresh wallet APK, or test a temporary issuer or verifier deployment. Scheduled runs, device queueing across teams and a live view of the device during a run are included, and results come back by API callback or email. Start with the [first pipeline guide](https://docs.credimi.io/manual/create-first-pipeline/), then read the [CI/CD documentation](https://docs.credimi.io/manual/ci-cd/) and [conformance automation](/pages/conformance-automation).

## One pipeline from issuer to wallet to verifier

A run can chain issuer, wallet and verifier end to end in one pipeline. A live issuer sends a credential offer, the wallet accepts it on a device, and a live verifier requests and checks the presentation. Because every step runs against a real component, a failure points to the place where the flow broke.

The same scenario can run against many wallets. Run it once per wallet and you get a matrix: each wallet against each issuer and verifier you care about, with one run and its evidence behind every cell.

<figure>
<img src="/pages/eudi-wallet-interoperability-testing/interop-grid.svg" alt="Illustrative interoperability matrix: three wallets tested against two issuers and two verifiers, with passed, failed and not yet run combinations" width="1200" height="460" loading="lazy" />
<figcaption>Illustrative results, not real data: each cell is one run of the same scenario between a wallet and an issuer or verifier.</figcaption>
</figure>

A failed cell is where you start reading the evidence. A cell that has not run yet is a combination you have not covered.

## Before and after interoperability events

The European Commission runs interoperability testing events such as the [EUDI Wallet Launchpad](https://ec.europa.eu/digital-building-blocks/sites/pages/viewpage.action?pageId=964168375). The [large-scale pilots](https://digital-strategy.ec.europa.eu/en/policies/eudi-wallet-toolbox), including POTENTIAL, EWC, WE BUILD and APTITUDE, run interoperability events too. An event brings many counterparts together for a limited time.

Running the same scenarios continuously makes that time count:

- **Before the event**, you arrive knowing which flows already work with which components, and you have the evidence to show it.
- **During the event**, you test against new counterparts. When a new combination works, you can turn it into a pipeline.
- **After the event**, the fixes you made run per commit or on a schedule, so a flow that worked at the event keeps working, or you see the run that broke it.

If you are mapping these tests to legal obligations, read [eIDAS 2 wallet testing](/pages/eidas2-wallet-testing).

## Results, evidence and the public scoreboard

Every run keeps its evidence:

- a timestamped Temporal history of every step input and output;
- the protocol messages: offers, requests and responses;
- device video, screenshots and the last frame;
- device logs;
- SHA-256 hashed FCAF evidence.

The FCAF report links each test case to hashed evidence in the run history, so you can check what each result was judged on. Published pipelines feed the public [scoreboard](/scoreboard), where others can see how components perform in the same scenarios.

<div class="content-quote"><p>An interoperability result is only as useful as the evidence you can open behind it.</p></div>

For a comparison with device clouds and standalone conformance suites, see [why Credimi](/pages/why-credimi).

## FAQ

### What is the difference between conformance and interoperability testing?

Conformance testing checks one component against a specification, usually with a simulated counterpart. Interoperability testing checks that real components complete a flow together: an issuer, a wallet and a verifier. Credimi runs both, and an interoperability run can also be judged against FCAF and protocol test cases.

### Can I test my wallet against third-party issuers without building my own?

Yes. The Hub lists live credential issuers and verifiers, each with a working flow through a QR code or a deeplink. You can try them by hand, then compose them in a pipeline with your wallet.

### Can I test my issuer or verifier against many wallets?

Yes. A pipeline can run the same scenario against many wallets, so you see how your credential offer or presentation request works with each one.

### Does Credimi certify interoperability?

No. Credimi does not certify anything and is not a conformity assessment body. It gives you runs and evidence that you can inspect and share.

### Can I run interoperability tests in CI?

Yes. Use the `run-wallet-apk`, `run-issuer` or `run-verifier` endpoint, or the GitHub Action, to run pipelines per commit or on a schedule.

### Is Credimi open source?

Yes. Credimi is released under AGPL-3.0 by Forkbomb BV, and its [source code](https://github.com/forkbombeu/credimi) is public. You can use it at credimi.io or [self-host it](https://docs.credimi.io/software-architecture/docker-deployment/). Credimi is supported by NGI TRUSTCHAIN, part of the European Commission Next Generation Internet programme.

<div class="content-related">
<p class="content-related-title">Read next:</p>
<div class="content-related-grid">
<a href="/pages/eudi-wallet-testing"><span>Guide</span><strong>Test your EUDI wallet against real issuers and verifiers</strong><em>Read guide ↗</em></a>
<a href="/pages/fcaf-conformance"><span>Guide</span><strong>FCAF conformance testing for EUDI wallets</strong><em>Read guide ↗</em></a>
<a href="/pages/openid4vc-conformance"><span>Guide</span><strong>OpenID4VCI and OpenID4VP conformance testing</strong><em>Read guide ↗</em></a>
</div>
</div>

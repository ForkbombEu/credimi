---
title: 'How much do you know about the EUDI rulebook?'
description: 'The EUDI Trust & Conformance Atlas maps 211 EUDI laws, ARF requirements, standards and protocols with adoption status and a 69-term glossary.'
lead: 'Trace every EUDI Wallet requirement — law, ARF, standards and protocols — to the source document that defines it, with its adoption status.'
heroCards:
  - role: wallets
    title: 'Wallet'
    detail: 'PID, attestations, trust lists'
  - role: credential_issuers
    title: 'Issuer'
    detail: 'Credentials, signatures, seals'
  - role: verifiers
    title: 'Verifier'
    detail: 'Presentation, request validation'
facts:
  - term: 'Documents mapped'
    value: '211 across four layers'
  - term: 'Glossary'
    value: '69 terms, cross-linked'
  - term: 'Adoption status'
    value: '8 statuses on every document'
  - term: 'Updated'
    value: 'October 2026 review'
cta:
  title: 'Open the Atlas and see what you are missing'
  text: 'Browse the rulebook by layer, status and role, and cite every term with a stable link.'
  href: 'https://atlas.credimi.io'
  label: 'Open the Atlas'
date: 2026-10-27
updatedOn: 2026-10-27
socialCard: /og/eudi-atlas.png
socialCardAlt: 'How much do you know about the EUDI rulebook? 211 documents mapped, 69 glossary terms and 28 drafts, beside three stat cards, in the EUDI Trust & Conformance Atlas.'
tags:
  - guide
  - faq
  - reference
  - documentation
  - conformance
  - compliance-and-standards
  - solutions
---

## What the Atlas is

The [EUDI Trust & Conformance Atlas](https://atlas.credimi.io) is a browsable map of the entire EUDI Wallet rulebook: the law, the implementing acts, the architecture and reference framework, the standards, and the protocols. Every document carries an adoption status — in force, adopted, published, final, draft, in development, proposal, or superseded — plus its status date and, where relevant, the versions pinned by the FCAF suite, the ARF essential-standards list and HAIP.

It is a reference, not an editor and not an API. It does not add another layer of interpretation: it links each requirement, standard and test back to the source document that defines it, so you can trace a legal obligation through the ARF to a runnable check.

## The whole rulebook, not just the tech

Most EUDI tooling starts at the protocol layer and stops at the API. The Atlas starts at the top and goes down. It organises 211 documents across four stack layers:

- **Law & implementing acts** — eIDAS, the EUDI amendment, implementing and delegated regulations and decisions, horizontal EU law and pending proposals.
- **Architecture & EC specifications** — the ARF, EC technical specifications, attestation rulebooks, FCAF and pilot working documentation.
- **Standards** — ETSI, CEN and ISO/IEC standards, standardisation roadmaps and cybersecurity guidance.
- **Protocols & platform specs** — OpenID4VC, IETF, W3C and FIDO specifications used by wallets, issuers and relying parties.

No public resource maps the full stack this way: every document placed on a layer, dated, and cross-referenced to the versions the conformance and trust machinery actually pins. That is the part "technical" tooling skips, and it is the part that decides whether a wallet passes an audit rather than just a test.

## The six views

- **Home and Radar** — the rulebook stack at a glance, the regulatory radar (every document as a dot placed by layer and status), and the momentum of change: 55 documents in force, 83 published, 28 in draft and 23 in development today.
- **Map** — browse the ecosystem by task, and filter by role, to see which document governs which actor.
- **Reference** — filter all documents by layer, status, type, ARF-essential, or free text. Preselect with `?q=`, `?stack=` and `?role=`.
- **Tests** — the conformance inventory and the evidence that backs it.
- **Glossary** — the rulebook by topic.
- **Data** — direct access to the source data for anyone who wants to build on it.

## The glossary

The glossary turns the rulebook into a dictionary you can actually read. Each of its 69 terms carries a definition, the abbreviations and phrases it matches, and — the part that makes it useful for audits — how many documents mention it, split by stack layer, with a link to every one of them.

For example, **EUDI Wallet** is defined as the electronic identification means under Article 5a of eIDAS: the wallet that lets a user store and present person identification data and electronic attestations of attributes, and create qualified electronic signatures and seals. Each term is a stable, shareable link, so a definition can be cited in a spec discussion or a conformance claim.

## How to consume it

The Atlas is built for four different kinds of reader:

- **Building a wallet, issuer or verifier** — start in Map and Reference, filtered by your role, to see every document that constrains your component.
- **Writing conformance evidence** — use Tests and the pinned versions to see exactly which revision of FCAF, the ARF and HAIP your evidence must cite.
- **Tracking change** — the Radar's momentum chart and the dated changelog show what moved each review. The October 2026 review alone added the mDL implementing regulation, the digital euro and digital-travel proposals, the ISO/IEC 27566 age-assurance series, EN 18037, the SD-CWT, JSON Web Proof and BBS drafts, HAIP 1.1 and seven ETSI work items.
- **Researching or citing** — the Glossary and Data views give you stable links to definitions and to the raw source data.

Filter URLs stay readable, so a query is itself a citation:

```
atlas.credimi.io/reference?stack=legal&q=wallet
```

## FAQ

### Is the Atlas an official EU source?

No. It is a community-maintained reference that maps and links to the official documents — regulations, decisions, standards and specifications — and states each one's status and date. It does not replace them; it makes them navigable.

### How does it stay current?

Every review is recorded in a dated changelog, and the document map is manifest-driven, so a change is one entry in the source data, not a rewrite of the site.

### How does it relate to Credimi?

The Atlas is the research layer and Credimi is the testing layer. The Atlas tells you what the rulebook requires; Credimi runs it. Promotional blocks for the Credimi platform are labelled "From the maintainers", so the reference content and the product pitch are never mixed.

### Who maintains it?

The Credimi team at Forkbomb BV, as open source under Apache-2.0. The same maintainers run the Credimi conformance platform, which is why the research and the tests stay in lockstep.

<div class="content-related">
<p class="content-related-title">Read next:</p>
<div class="content-related-grid">
<a href="/pages/fcaf-conformance"><span>Guide</span><strong>FCAF conformance testing for EUDI wallets</strong><em>Read guide ↗</em></a>
<a href="/pages/eudi-wallet-testing"><span>Guide</span><strong>Test your EUDI wallet against real issuers and verifiers</strong><em>Read guide ↗</em></a>
<a href="/pages/openid4vc-conformance"><span>Guide</span><strong>OpenID4VCI and OpenID4VP conformance testing</strong><em>Read guide ↗</em></a>
</div>
</div>

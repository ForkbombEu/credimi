---
title: 'EUDI Wallet rulebook and regulatory map'
description: 'A practical map of the legal, technical, certification, trust-list and qualified trust-service documents shaping the European Digital Identity Wallet ecosystem.'
date: 2026-04-16
tags:
  - documentation
  - guide
  - architecture
  - compliance
  - reference
  - legal-and-privacy
  - compliance-and-standards
  - identity-wallet-integration
  - developer-tools
updatedOn: 2026-05-20
---

# Navigating the EUDI Wallet Rulebook — Expanded Regulatory Map

A practical guide to the legal, technical, certification and trust-infrastructure documents behind the European Digital Identity Wallet ecosystem.

This document is written for mixed teams: product people, developers, legal/compliance stakeholders, Member State representatives, certification-adjacent actors, Wallet implementers, Issuers and Relying Parties.

The goal is not to make everyone read every regulation. The goal is to show which documents matter for which part of the EUDI Wallet ecosystem, and how they connect to Credimi-style conformance and interoperability evidence.

---

> **Legend**
>
> 🟦 **Legal / Regulation** — primarily useful for legal, policy, compliance, CAB, and Member State audiences.  
> 🟧 **Technical / Protocol / Implementation** — primarily useful for engineers, architects, wallet/issuer/verifier developers, and test implementers.  
> 🟨 **Standards / ETSI / Normative technical profiles** — primarily useful when turning legal/ARF requirements into concrete checks, profiles, policies and validation rules.  
> 🟩 **Assurance / Certification / Audit** — primarily useful for certification, conformity assessment, evidence review, and supervisory discussions.  
> 🟪 **Trust Infrastructure / Qualified Trust Services** — primarily useful for Trusted Lists, QTSPs, certificates, revocation, validation services, and advanced trust-service integrations.  
> 🟦🟧 **Legal + Technical** — directly bridges legal obligations and implementable technical behaviour.  
> 🟦🟩 **Legal + Assurance** — legal basis for evidence, certification, audit, or supervisory reporting.  
> 🟧🟪 **Technical + Trust Infrastructure** — implementation-heavy trust, certificate, list, revocation, or validation material.  
> 🟨🟪 **Standards + Trust Infrastructure** — ETSI and similar standards for certificates, TSP policies, validation, timestamps, trusted lists and qualified trust services.

## 🧭 How to read the colors

The colours are not legal categories. They are a reading aid for mixed legal/technical teams.  
A document can appear under more than one logical area, and the same document can be legal, technical, standards-related, assurance-related and trust-infrastructure-related depending on how Credimi uses it.

## 1. The short mental model

The EUDI Wallet rulebook is easier to understand as a layered system:

```text
Legal foundation
  eIDAS + EUDI amendment

Binding operational rules
  Commission Implementing Regulations and Decisions

Technical architecture
  EUDI Architecture and Reference Framework (ARF)

Pilot / ecosystem interpretation
  WE BUILD WP4 trust-registry and conformance work

Executable evidence
  Credimi pipelines: StepCI + Maestro + Temporal + third-party conformance tests + trust-helper checks
```

For Credimi, the practical question is:

> Which part of this rulebook can be turned into reproducible evidence?

Examples:

- A Wallet receives and completes a PID issuance flow.
- A Wallet opens a presentation request and waits for user approval.
- A Verifier generates a valid presentation request.
- A Relying Party only requests attributes it is registered or entitled to request.
- An Issuer is present in the relevant trusted list or registry.
- A run produces a reproducible evidence bundle: logs, screenshots, video, protocol traces, metadata snapshots, hashes and external conformance results.

---

## 2. Legend

```text
Legal foundation
  Core regulations defining the legal framework.

Binding implementing rule
  Commission Implementing Regulations / Decisions.

Technical reference
  ARF, standards, project documentation or implementation guidance.

Operational evidence target
  Something Credimi can test, observe, validate or attach to a report.

Advanced assurance / QTSP
  Relevant for qualified trust services, qualified validation, timestamping or high-assurance bundles.
```

---

## 🟦 3. The legal foundation

### eIDAS and the EUDI amendment

The original eIDAS Regulation created the EU framework for electronic identification and trust services. The 2024 EUDI amendment introduced the European Digital Identity Wallet framework and expanded the role of Wallets, Issuers, Relying Parties and trust services.

| Document | Why it matters |
|---|---|
| [Regulation (EU) No 910/2014 — eIDAS](https://data.europa.eu/eli/reg/2014/910/oj) | Original legal foundation for eID and trust services. |
| [Consolidated eIDAS after EUDI amendments](https://data.europa.eu/eli/reg/2014/910/2024-10-18) | Current working legal baseline after amendments. |
| [Regulation (EU) 2024/1183 — European Digital Identity Framework](https://data.europa.eu/eli/reg/2024/1183/oj) | Establishes the EUDI Wallet framework. |
| [Commission Recommendation (EU) 2021/946 — Common Union Toolbox](https://data.europa.eu/eli/reco/2021/946/oj) | Procedural origin of the ARF / toolbox work. |

**Credimi angle:** these documents are the legal source of the ecosystem. They do not directly define a StepCI or Maestro test, but they explain why Wallet certification, interoperability, Relying Party registration and trust-service supervision exist.

---

## 🟧🟪 4. The technical blueprint: ARF and WE BUILD

### 🟦🟧 EUDI Architecture and Reference Framework

The ARF is the main technical bridge between the legal framework and implementation. It describes roles, flows, technical architecture, high-level requirements, Wallet behaviour, Issuer/Verifier expectations, security and privacy considerations.

| Document | Why it matters |
|---|---|
| [EUDI ARF latest documentation](https://eudi.dev/latest/) | Primary technical and architectural reference. |
| [EUDI ARF GitHub repository](https://github.com/eu-digital-identity-wallet/eudi-doc-architecture-and-reference-framework) | Source repository for ARF documentation and high-level requirements. |

### 🟧🟪 WE BUILD WP4

WE BUILD WP4 is more concrete around trust evaluation, trusted lists, participant certificates, onboarding and conformance/interoperability tests.

| Document | Why it matters |
|---|---|
| [WE BUILD WP4 Trust Group repository](https://github.com/webuild-consortium/wp4-trust-group) | Practical trust-registry, trust-list and conformance work. |


### 🟨 ETSI ESI standards

The ETSI ESI standards are not laws. They are the technical and policy standards that many of the implementing acts and trust-service profiles point to when the legal text says that a process, certificate, timestamp, validation service, trusted-list mechanism or qualified trust service must follow recognised reference standards.

| Document family | Why it matters |
|---|---|
| ETSI EN/TS/TR 319/119 47x | EUDI-specific EAA/PID, selective disclosure, authentic-source and relying-party attribute work. |
| ETSI EN/TS 319/119 411 / 412 | CA policy requirements and certificate profiles, including qualified certificates. |
| ETSI EN/TS 319/119 421 / 422 | Time-stamping policies and protocols. |
| ETSI TS 119 431 / 432 / 441 / 442 | Remote signature creation and signature validation services. |
| ETSI EN/TS 319/119 401 / 403 | General TSP policy and conformity-assessment requirements. |

**Credimi angle:** ETSI standards are where the abstract legal/trust requirements become checkable technical profiles: metadata checks, certificate parsing, policy OID checks, signature/seal validation, timestamp validation, qualified validation-service evidence and trust-helper fixtures.

**Credimi angle:** ARF and WE BUILD are the best sources for translating legal expectations into executable test/evidence claims.

---

## 🟦🟩 5. Wallet rules: identity data, functions, certification and protocols

These are the core Wallet-era implementing acts from 2024. They are the first documents to read for Wallet / Issuer / Verifier conformance evidence.

| Document | Why it matters |
|---|---|
| [CIR 2024/2977](https://data.europa.eu/eli/reg_impl/2024/2977/oj) | **PID and electronic attestations of attributes issued to Wallets.** Credential-offer, issuance, issuer metadata, validity status and revocation evidence. |
| [CIR 2024/2979](https://data.europa.eu/eli/reg_impl/2024/2979/oj) | **Wallet integrity and core functionalities.** Wallet transaction logs, portability, pseudonyms and user-control evidence. |
| [CIR 2024/2980](https://data.europa.eu/eli/reg_impl/2024/2980/oj) | **Notifications to the Commission.** Machine-readable ecosystem information and future Wallet/provider list checks. |
| [CIR 2024/2981](https://data.europa.eu/eli/reg_impl/2024/2981/oj) | **Wallet certification.** Assurance documentation, certification support and dependency-analysis evidence. |
| [CIR 2024/2982](https://data.europa.eu/eli/reg_impl/2024/2982/oj) | **Protocols and interfaces.** Remote issuance, remote presentation, RP information display, user approval and selective disclosure. |

### Practical flow

```text
Issuer/Provider generates credential offer
  ↓
Wallet receives offer and starts issuance
  ↓
Wallet completes PID/EAA issuance
  ↓
Verifier/RP generates presentation request
  ↓
Wallet displays RP identity and requested attributes
  ↓
User approves disclosure
  ↓
Verifier receives and validates presentation
```

### Evidence Credimi can produce

- Credential offer captured by StepCI.
- Issuer metadata snapshot.
- Real Wallet interaction driven by Maestro.
- Screenshots and video.
- Verifier callback/result.
- Temporal workflow ID and run ID.
- Step input/output traces.
- Normalized conformance evidence report.

---

## 6. 2025 Wallet ecosystem rules

These acts govern the ecosystem around Wallets: identity matching, incident response, Relying Party registration and certified Wallet lists.

| Document | Why it matters |
|---|---|
| [CIR 2025/846](https://data.europa.eu/eli/reg_impl/2025/846/oj) | **Cross-border identity matching.** Public-sector and PID matching scenarios. |
| [CIR 2025/847](https://data.europa.eu/eli/reg_impl/2025/847/oj) | **Wallet security breach reactions.** Monitoring and operational assurance evidence. |
| [CIR 2025/848](https://data.europa.eu/eli/reg_impl/2025/848/oj) | **Registration of Wallet-relying parties.** RP register lookup, entitlements, access certificates and registration certificates. |
| [CIR 2025/849](https://data.europa.eu/eli/reg_impl/2025/849/oj) | **List of certified Wallets.** Checking Wallet certification metadata once lists are available. |

### Practical flow for RP registration and entitlement

```text
Relying Party creates a presentation request
  ↓
Credimi parses requested attributes
  ↓
Credimi resolves RP registration / certificate / entitlement
  ↓
Credimi compares request against registered entitlement
  ↓
Wallet behaviour and/or helper result becomes evidence
```

### Evidence Credimi can produce

- Presentation request snapshot.
- RP identifier and register lookup.
- RP access certificate / registration certificate snapshot.
- Attribute entitlement comparison.
- Negative tests where RP asks for unauthorized attributes.
- Wallet warning/rejection screenshot where applicable.

---

## 🟦🟧 7. Issuers, PID Providers, EAA Providers and QEAA/QEEA

Issuers are not only endpoints that issue credentials. Depending on the credential type, they may need to be recognised, registered, qualified or entitled to issue a certain attestation.

| Document | Why it matters |
|---|---|
| [CIR 2024/2977](https://data.europa.eu/eli/reg_impl/2024/2977/oj) | **PID/EAA issuance to Wallets.** Core issuance evidence. |
| [CIR 2025/1566](https://data.europa.eu/eli/reg_impl/2025/1566/oj) | **Identity and attribute verification standards.** QTSP/QEAA issuer assurance. |
| [CIR 2025/1569](https://data.europa.eu/eli/reg_impl/2025/1569/oj) | **Qualified EAAs and public-sector authentic-source EAAs.** QEAA/QEEA issuer qualification, authentic-source attestations and revocation. |
| [CIR 2025/2530](https://data.europa.eu/eli/reg_impl/2025/2530/oj) | **Requirements for QTSPs.** Relevant where issuers are QTSPs or QTSP-like actors. |
| [CIR 2025/1572](https://data.europa.eu/eli/reg_impl/2025/1572/oj) | **Notification of intention to provide qualified trust services.** Relevant for QTSP onboarding/conformity evidence. |
| [CIR 2025/2162](https://data.europa.eu/eli/reg_impl/2025/2162/oj) | **CAB accreditation and conformity assessment.** Relevant to auditor/CAB-facing evidence. |

### Practical sub-steps

```text
Issuer metadata is reachable
  ↓
Credential offer is valid
  ↓
Real Wallet or headless Wallet completes issuance
  ↓
Issuer status / entitlement is resolved
  ↓
Credential validity and revocation mechanism is checked
```

### Credimi MVP evidence

- Issuer metadata and credential offer.
- Real Wallet issuance flow.
- 🟧🟪 External OpenID / WE BUILD issuer conformance results.
- Revocation/status endpoint checks.

### 🟨 Related ETSI standards for Issuers and Attestation Providers

The most directly relevant ETSI family here is the EAA/PID and authentic-source work. These documents help connect EAA/PID issuance and presentation to concrete profiles, policy/security requirements, authentic-source interfaces and extended validation-service concepts.

| ETSI document | Details | Why it belongs here |
|---|---|---|
| [ETSI TS 119 471 V1.1.1 (2025-05)](https://www.etsi.org/deliver/etsi_ts/119400_119499/119471/01.01.01_60/ts_119471v010101p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=63664) | Policy/security requirements for EAA service providers. |
| [ETSI TS 119 472-1 V1.2.1 (2026-02)](https://www.etsi.org/deliver/etsi_ts/119400_119499/11947201/01.02.01_60/ts_11947201v010201p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=77499) | General EAA profile requirements. |
| [ETSI TS 119 472-2 V1.2.1 (2026-03)](https://www.etsi.org/deliver/etsi_ts/119400_119499/11947202/01.02.01_60/ts_11947202v010201p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=77507) | EAA/PID presentation profiles to relying parties. |
| [ETSI TS 119 472-3 V1.1.1 (2026-03)](https://www.etsi.org/deliver/etsi_ts/119400_119499/11947203/01.01.01_60/ts_11947203v010101p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=74935) | EAA/PID issuance profiles. |
| [ETSI TS 119 478 V1.1.1 (2026-01)](https://www.etsi.org/deliver/etsi_ts/119400_119499/119478/01.01.01_60/ts_119478v010101p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=74951) | Authentic-source interfaces. |
| [ETSI TR 119 479-1 V1.1.1 (2026-05)](https://www.etsi.org/deliver/etsi_tr/119400_119499/11947901/01.01.01_60/tr_11947901v010101p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=73774) | Foundational EAA concepts and architecture. |
| [ETSI TR 119 479-2 V1.1.1 (2025-07)](https://www.etsi.org/deliver/etsi_tr/119400_119499/11947902/01.01.01_60/tr_11947902v010101p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=73779) | EAA extended validation-service framework. |

### Credimi Level 2 evidence

- Trusted List / registry lookup.
- Issuer qualification status.
- Issuer entitlement to issue a specific attestation type.
- Negative tests for revoked, suspended or untrusted issuers.

---

## 🟦🟧 8. Relying Parties and Verifiers

Relying Parties are central to the EUDI trust model because users must know who is asking for attributes and whether that actor is entitled to request them.

| Document | Why it matters |
|---|---|
| [CIR 2025/848](https://data.europa.eu/eli/reg_impl/2025/848/oj) | **RP registration and entitlements.** Core RP trust-helper work. |
| [CIR 2024/2982](https://data.europa.eu/eli/reg_impl/2024/2982/oj) | **Protocols/interfaces and RP info display.** Wallet display of RP identity, requested attributes and approval. |
| [CIR 2025/1943](https://data.europa.eu/eli/reg_impl/2025/1943/oj) | **Qualified certificate standards.** Useful where RP certificates rely on qualified certificate profiles. |
| [CIR 2025/1945](https://data.europa.eu/eli/reg_impl/2025/1945/oj) | **Signature/seal validation.** Useful for verifier-side validation evidence. |
| [WE BUILD WP4](https://github.com/webuild-consortium/wp4-trust-group) | **RP trust evaluation use cases.** Concrete pilot-style trust evaluation scenarios. |

### 🟨 Related ETSI standards for Relying Parties and Verifiers

For Relying Parties, the most relevant ETSI layer is a combination of relying-party attributes, EAA/PID presentation profiles, certificate profiles and validation-service standards.

| ETSI document | Details | Why it belongs here |
|---|---|---|
| [ETSI TS 119 475 V1.2.1 (2026-03)](https://www.etsi.org/deliver/etsi_ts/119400_119499/119475/01.02.01_60/ts_119475v010201p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=77921) | Relying-party attributes supporting Wallet user authorisation decisions. |
| [ETSI TS 119 472-2 V1.2.1 (2026-03)](https://www.etsi.org/deliver/etsi_ts/119400_119499/11947202/01.02.01_60/ts_11947202v010201p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=77507) | EAA/PID presentation profiles to relying parties. |
| [ETSI TS 119 441 V1.3.1 (2025-10)](https://www.etsi.org/deliver/etsi_ts/119400_119499/119441/01.03.01_60/ts_119441v010301p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=74891) | Policy requirements for signature validation services. |
| [ETSI TS 119 442 V1.1.1 (2019-02)](https://www.etsi.org/deliver/etsi_ts/119400_119499/119442/01.01.01_60/ts_119442v010101p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=47246) | Protocol profiles for AdES validation services. |
| [ETSI TS 119 495 V1.8.1 (2026-04)](https://www.etsi.org/deliver/etsi_ts/119400_119499/119495/01.08.01_60/ts_119495v010801p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=75421) | Sector-specific certificate and TSP policy requirements for open banking. |
| [ETSI EN 319 412-1 V1.7.1 (2026-05)](https://www.etsi.org/deliver/etsi_en/319400_319499/31941201/01.07.01_60/en_31941201v010701p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=75441) | Certificate profile overview and common data structures. |
| [ETSI EN 319 412-2 V2.5.0 (2026-05)](https://www.etsi.org/deliver/etsi_en/319400_319499/31941202/02.05.00_20/en_31941202v020500a.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=78235) | Certificate profile for natural persons. |
| [ETSI EN 319 412-3 V1.4.0 (2026-04)](https://www.etsi.org/deliver/etsi_en/319400_319499/31941203/01.04.00_20/en_31941203v010400a.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=78236) | Certificate profile for legal persons. |
| [ETSI EN 319 412-4 V1.4.1 (2025-06)](https://www.etsi.org/deliver/etsi_en/319400_319499/31941204/01.04.01_60/en_31941204v010401p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=73791) | Certificate profile for web site certificates. |
| [ETSI EN 319 412-5 V2.6.1 (2026-05)](https://www.etsi.org/deliver/etsi_en/319400_319499/31941205/02.06.01_60/en_31941205v020601p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=76590) | Qualified certificate statements. |
| [ETSI TS 119 412-6 V1.2.1 (2026-04)](https://www.etsi.org/deliver/etsi_ts/119400_119499/11941206/01.02.01_60/ts_11941206v010201p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=77496) | Distributed ledger identifiers in certificates. |

### Practical sub-steps

```text
Verifier generates presentation request
  ↓
Request contains RP information / certificate / registration evidence
  ↓
Requested attributes are extracted
  ↓
RP registration and entitlement are resolved
  ↓
Wallet displays RP and requested attributes
  ↓
Verifier validates credential response
```

### Evidence Credimi can produce

- Presentation request capture.
- RP access certificate / registration certificate parsing.
- Requested-attribute vs entitlement comparison.
- Wallet screenshot showing RP identity and requested attributes.
- Verifier accepts valid presentation or rejects invalid/revoked/untrusted credentials.

---

## 🟧🟪 9. Trusted Lists, LoTL and trust infrastructure

Trusted Lists and Lists of Trusted Lists are the machine-readable trust fabric. They are needed to resolve actors, trust anchors, statuses, certificates and revocation information.

| Document | Why it matters |
|---|---|
| [CID 2015/1505](https://data.europa.eu/eli/dec_impl/2015/1505/oj) | **Trusted List formats.** Base for TL parsing and validation. |
| [CID 2025/2164](https://data.europa.eu/eli/dec_impl/2025/2164/oj) | **Common Trusted List template standard.** Newer TL template/versioning reference. |
| [CIR 2024/2980](https://data.europa.eu/eli/reg_impl/2024/2980/oj) | **Notifications and machine-readable information.** Ecosystem list publication context. |
| [CIR 2025/848](https://data.europa.eu/eli/reg_impl/2025/848/oj) | **RP registers and entitlement data.** RP trust and entitlement resolution. |
| [CIR 2025/1569](https://data.europa.eu/eli/reg_impl/2025/1569/oj) | **QEAA / public-sector EAAs.** Issuer status and entitlement context. |
| [WE BUILD WP4](https://github.com/webuild-consortium/wp4-trust-group) | **LoTL/TL integration and trust evaluation.** Pilot-facing trust infrastructure. |

### 🟨 Related ETSI standards for trust infrastructure

Trusted Lists, certificates, validation and QTSP evidence are where the ETSI standards become most operational. The most relevant families are **ETSI EN 319 401** for general TSP policy, **ETSI EN 319 403-1** and related 119 403 parts for conformity assessment, **ETSI EN 319 411-1/2** and related 119 411 parts for CA policy, **ETSI EN 319 412-1..5** and related 119 412 parts for certificate profiles, and **ETSI EN 319 421/422** for timestamps.

### Practical sub-steps

```text
Fetch LoTL
  ↓
Discover referenced Trusted Lists
  ↓
Validate schema and signature/seal
  ↓
Resolve actor entry
  ↓
Check actor status
  ↓
Check certificate / revocation / entitlement
```

### Evidence Credimi can produce

- LoTL/TL snapshots.
- Hash manifest.
- Schema validation result.
- Signature/seal validation result.
- Actor status result.
- Certificate chain and revocation result.
- Entitlement result.

### 🟨🟪 Related ETSI standards for trust infrastructure

These standards are the most relevant ETSI anchors for the trust-helper side: TSP policies, conformity assessment, certificate-issuing TSP requirements, certificate profiles and timestamping.

| ETSI document | Details | Why it belongs here |
|---|---|---|
| [ETSI EN 319 401 V2.3.1 (2021-05)](https://www.etsi.org/deliver/etsi_en/319400_319499/319401/02.03.01_60/en_319401v020301p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=59276) | General policy requirements for Trust Service Providers. |
| [ETSI EN 319 403-1 V2.3.1 (2020-06)](https://www.etsi.org/deliver/etsi_en/319400_319499/31940301/02.03.01_60/en_31940301v020301p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=56885) | Conformity assessment requirements for TSPs. |
| [ETSI EN 319 411-1 V1.5.1 (2025-04)](https://www.etsi.org/deliver/etsi_en/319400_319499/31941101/01.05.01_60/en_31941101v010501p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=70003) | Policy/security requirements for certificate-issuing TSPs. |
| [ETSI EN 319 411-2 V2.6.1 (2025-06)](https://www.etsi.org/deliver/etsi_en/319400_319499/31941102/02.06.01_60/en_31941102v020601p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=72255) | Requirements for TSPs issuing EU qualified certificates. |
| [ETSI EN 319 412-1 V1.7.1 (2026-05)](https://www.etsi.org/deliver/etsi_en/319400_319499/31941201/01.07.01_60/en_31941201v010701p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=75441) | Certificate profile overview. |
| [ETSI EN 319 412-5 V2.6.1 (2026-05)](https://www.etsi.org/deliver/etsi_en/319400_319499/31941205/02.06.01_60/en_31941205v020601p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=76590) | Qualified certificate statements. |
| [ETSI EN 319 421 V1.3.1 (2025-07)](https://www.etsi.org/deliver/etsi_en/319400_319499/319421/01.03.01_60/en_319421v010301p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=70004) | Policy/security requirements for time-stamp providers. |
| [ETSI EN 319 422 V1.1.1 (2016-03)](https://www.etsi.org/deliver/etsi_en/319400_319499/319422/01.01.01_60/en_319422v010101p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=39370) | Time-stamping protocol and token profiles. |

---

## 🟦🟩 10. Assurance levels and eID schemes

The assurance-level framework predates the EUDI Wallet, but it remains important because Wallet/eID schemes and identity proofing refer back to low/substantial/high assurance levels.

| Document | Why it matters |
|---|---|
| [CIR 2015/1502](https://data.europa.eu/eli/reg_impl/2015/1502/oj) | **Assurance levels for electronic identification means.** Low/substantial/high assurance model. |
| [eIDAS Regulation](https://data.europa.eu/eli/reg/2014/910/oj) | **Legal basis for assurance levels.** Source framework. |
| [CIR 2025/1568](https://data.europa.eu/eli/reg_impl/2025/1568/oj) | **Peer reviews of eID schemes.** Scheme-level assurance and Member State review context. |
| [CIR 2024/2981](https://data.europa.eu/eli/reg_impl/2024/2981/oj) | **Wallet certification.** Certification and dependency-analysis context. |

### Practical sub-steps

```text
Understand claimed assurance level
  ↓
Map to enrolment, eID means management, authentication and organisation controls
  ↓
Attach evidence or certification material
  ↓
Treat as assurance documentation, not a normal Maestro UI test
```

---

## 🟦🟩 11. Certification, CABs, NABs and assurance evidence

Certification is where legal and technical requirements are assessed. Credimi should not claim to certify Wallets unless explicitly mandated/accredited. The right claim is that Credimi can generate reproducible assurance evidence.

| Document | Why it matters |
|---|---|
| [CIR 2024/2981](https://data.europa.eu/eli/reg_impl/2024/2981/oj) | **Wallet certification.** Central for certification-support evidence. |
| [CIR 2025/849](https://data.europa.eu/eli/reg_impl/2025/849/oj) | **Certified Wallet list.** Certified Wallet metadata lookup. |
| [CIR 2025/2162](https://data.europa.eu/eli/reg_impl/2025/2162/oj) | **CAB accreditation and conformity assessment.** CAB-facing evidence and assessment process. |
| [Regulation (EC) No 765/2008](https://data.europa.eu/eli/reg/2008/765/oj) | **Accreditation and market surveillance.** CAB/legal accreditation context. |
| [Cybersecurity Act](https://data.europa.eu/eli/reg/2019/881/oj) | **EU cybersecurity certification framework.** Cybersecurity certification context. |
| [EUCC Regulation 2024/482](https://data.europa.eu/eli/reg_impl/2024/482/oj) | **EUCC scheme.** Security/certification standards context. |
| [EUCC amendment 2024/3144](https://data.europa.eu/eli/reg_impl/2024/3144/oj) | **EUCC amendments/corrections.** Updated EUCC context. |

### Credimi evidence bundle

```text
Temporal workflow/run IDs
  + step input/output
  + protocol traces
  + screenshots/videos
  + external conformance test outputs
  + trust-helper results
  + hash manifest
  + normalized conformance_result.json
  + optional PDF report
```

### Important language

Use:

> Credimi produces structured, reproducible conformance and interoperability evidence.

Avoid:

> Credimi certifies EUDI Wallet conformance.

---

## 🟧🟩 12. Third-party conformance suites

Third-party conformance suites are evidence sources, not replacements for real Wallet/RP/Issuer ecosystem testing.

| Source | Why it matters |
|---|---|
| [OpenID Foundation conformance tools](https://openid.net/certification/) | Protocol conformance evidence. |
| [WE BUILD WP4 tests](https://github.com/webuild-consortium/wp4-trust-group) | Trust and interoperability evidence. |
| [ARF](https://eudi.dev/latest/) | Requirement and architecture mapping. |

### Practical sub-steps

```text
Run external conformance suite
  ↓
Store raw external result
  ↓
Normalize result into Credimi evidence report
  ↓
Map assertions to evidence claims
  ↓
Show limitations clearly
```

---

## 🟪 13. Qualified trust services and advanced assurance

These documents matter, but they should not block Credimi's first conformance-evidence MVP. They are mainly for advanced assurance, QTSP integrations, qualified timestamps, qualified certificates, signature/seal validation, preservation, archiving, ledgers and QERDS.

| Document | Why it matters |
|---|---|
| [CIR 2025/1567](https://data.europa.eu/eli/reg_impl/2025/1567/oj) | **Remote QSCD/QSealCD management.** Advanced signature/seal trust-service evidence. |
| [CIR 2025/1570](https://data.europa.eu/eli/reg_impl/2025/1570/oj) | **Certified QSCD/QSealCD notification.** Device notification evidence. |
| [CIR 2025/1572](https://data.europa.eu/eli/reg_impl/2025/1572/oj) | **QTSP notification of intention.** QTSP onboarding evidence. |
| [CIR 2025/1929](https://data.europa.eu/eli/reg_impl/2025/1929/oj) | **Qualified timestamps.** Optional qualified timestamping of evidence bundles. |
| [CIR 2025/1942](https://data.europa.eu/eli/reg_impl/2025/1942/oj) | **Qualified validation services.** Optional qualified validation service integration. |
| [CIR 2025/1943](https://data.europa.eu/eli/reg_impl/2025/1943/oj) | **Qualified certificate standards.** Advanced certificate profile validation. |
| [CIR 2025/1944](https://data.europa.eu/eli/reg_impl/2025/1944/oj) | **QERDS.** Future qualified delivery of evidence, not MVP. |
| [CIR 2025/1945](https://data.europa.eu/eli/reg_impl/2025/1945/oj) | **Signature/seal validation.** Advanced verifier/signature validation evidence. |
| [CIR 2025/1946](https://data.europa.eu/eli/reg_impl/2025/1946/oj) | **Qualified preservation services.** Long-term qualified preservation evidence. |
| [CIR 2025/2527](https://data.europa.eu/eli/reg_impl/2025/2527/oj) | **Qualified website authentication certificates.** Advanced endpoint trust evidence. |
| [CIR 2025/2530](https://data.europa.eu/eli/reg_impl/2025/2530/oj) | **QTSP requirements.** QTSP assurance and qualified services. |
| [CIR 2025/2531](https://data.europa.eu/eli/reg_impl/2025/2531/oj) | **Qualified electronic ledgers.** Future adjacent trust infrastructure. |
| [CIR 2025/2532](https://data.europa.eu/eli/reg_impl/2025/2532/oj) | **Qualified electronic archiving.** Future long-term evidence archiving. |

### Recommended Credimi treatment

```text
Level 1 — Reproducible Credimi evidence
  Temporal + StepCI + Maestro + screenshots/videos + metadata + hash manifest

Level 2 — Trust-resolved evidence
  Level 1 + trusted lists + RP/issuer status + entitlement + revocation/cert validation

Level 3 — Qualified evidence add-ons
  Level 2 + QTSP timestamp + qualified validation service + preservation/archiving/QERDS if needed
```

Do not expose Level 3 in the normal Maestro Action editor. Keep it in advanced trust-helper / enterprise assurance settings.

### 🟨🟪 Related ETSI standards for qualified trust services

These ETSI documents belong to the advanced assurance layer: remote signature/seal creation, qualified validation services, timestamping, and qualified certificate profiles.

| ETSI document | Details | Why it belongs here |
|---|---|---|
| [ETSI TS 119 431-1 V1.3.1 (2024-12)](https://www.etsi.org/deliver/etsi_ts/119400_119499/11943101/01.03.01_60/ts_11943101v010301p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=69586) | Remote QSCD / SCDev operation. |
| [ETSI TS 119 431-2 V1.2.1 (2023-06)](https://www.etsi.org/deliver/etsi_ts/119400_119499/11943102/01.02.01_60/ts_11943102v010201p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=61997) | AdES digital signature creation service components. |
| [ETSI TS 119 432 V1.3.1 (2026-03)](https://www.etsi.org/deliver/etsi_ts/119400_119499/119432/01.03.01_60/ts_119432v010301p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=73032) | Protocols for remote digital signature creation. |
| [ETSI TS 119 441 V1.3.1 (2025-10)](https://www.etsi.org/deliver/etsi_ts/119400_119499/119441/01.03.01_60/ts_119441v010301p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=74891) | Policy requirements for signature validation services. |
| [ETSI TS 119 442 V1.1.1 (2019-02)](https://www.etsi.org/deliver/etsi_ts/119400_119499/119442/01.01.01_60/ts_119442v010101p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=47246) | Protocol profiles for validation services. |
| [ETSI EN 319 421 V1.3.1 (2025-07)](https://www.etsi.org/deliver/etsi_en/319400_319499/319421/01.03.01_60/en_319421v010301p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=70004) | Time-stamp provider policy/security requirements. |
| [ETSI EN 319 422 V1.1.1 (2016-03)](https://www.etsi.org/deliver/etsi_en/319400_319499/319422/01.01.01_60/en_319422v010101p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=39370) | Time-stamp protocol and token profiles. |
| [ETSI EN 319 411-2 V2.6.1 (2025-06)](https://www.etsi.org/deliver/etsi_en/319400_319499/31941102/02.06.01_60/en_31941102v020601p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=72255) | Qualified certificate issuance requirements. |
| [ETSI EN 319 412-5 V2.6.1 (2026-05)](https://www.etsi.org/deliver/etsi_en/319400_319499/31941205/02.06.01_60/en_31941205v020601p.pdf) | [details](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=76590) | Qualified certificate statements. |

---


## 🟨 14. ETSI standards layer

The uploaded ETSI catalogue adds the standards layer beneath the legal and ARF/WE BUILD layers. These documents are not themselves the EUDI legal basis, but they are the material implementers and auditors use when a legal or ARF requirement needs to become a concrete certificate profile, policy requirement, protocol profile or validation check.

For Credimi, these standards mostly affect the **trust-helper**, **issuer/verifier validation**, **qualified trust-service evidence**, and **advanced assurance** layers. The core StepCI + Maestro Wallet automation still focuses on observable Wallet/Issuer/RP behaviour, but the ETSI standards make the deeper checks more precise.

> Catalog check: the uploaded `ETSICatalog.csv` contains **55** ETSI entries. The PDF links in the catalogue match the **55** PDF filenames in the uploaded ETSI ZIP.
>
Source note: ETSI links in this section are taken from the uploaded `ETSICatalog.csv`. The catalogue includes ETSI work item detail links and direct PDF links; the PDF filenames match the uploaded ETSI ZIP.

### EUDI Wallet, EAA/PID, selective disclosure and authentic-source standards

These are the most EUDI-specific ETSI documents in the uploaded set. They are closest to Wallet/Issuer/RP interoperability, EAA/PID issuance and presentation, selective disclosure, authentic-source interfaces and EAA validation-service concepts.

| ETSI deliverable | Short topic | PDF |
|---|---|---|
| [ETSI TR 119 479-2 V1.1.1 (2025-07)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=73779) | Technological Solutions for the EU Digital Identity Framework; Part 2: EAA Extended Validation Services Framework and Application | [PDF](https://www.etsi.org/deliver/etsi_tr/119400_119499/11947902/01.01.01_60/tr_11947902v010101p.pdf) |
| [ETSI TR 119 479-1 V1.1.1 (2026-05)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=73774) | Technological Solutions for the EU Digital Identity Framework; Part 1: Foundational EAA Concepts and Architectural Models | [PDF](https://www.etsi.org/deliver/etsi_tr/119400_119499/11947901/01.01.01_60/tr_11947901v010101p.pdf) |
| [ETSI TS 119 478 V1.1.1 (2026-01)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=74951) | Specification of interfaces related to Authentic Sources | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/119478/01.01.01_60/ts_119478v010101p.pdf) |
| [ETSI TR 119 476-1 V1.3.1 (2025-08)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=73046) | Selective disclosure and zero-knowledge proofs applied to Electronic Attestation of Attributes; Part 1: Feasibility study | [PDF](https://www.etsi.org/deliver/etsi_tr/119400_119499/11947601/01.03.01_60/tr_11947601v010301p.pdf) |
| [ETSI TR 119 476 V1.2.1 (2024-07)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=69479) | Analysis of selective disclosure and zero-knowledge proofs applied to Electronic Attestation of Attributes | [PDF](https://www.etsi.org/deliver/etsi_tr/119400_119499/119476/01.02.01_60/tr_119476v010201p.pdf) |
| [ETSI TS 119 475 V1.2.1 (2026-03)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=77921) | Relying party attributes supporting EUDI Wallet user's authorization decisions | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/119475/01.02.01_60/ts_119475v010201p.pdf) |
| [ETSI TS 119 472-3 V1.1.1 (2026-03)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=74935) | Profiles for Electronic Attestation of Attributes; Part 3: Profiles for issuance of EAA or PID | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/11947203/01.01.01_60/ts_11947203v010101p.pdf) |
| [ETSI TS 119 472-2 V1.2.1 (2026-03)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=77507) | Profiles for Electronic Attestation of Attributes; Part 2: Profiles for EAA/PID Presentations to Relying Party | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/11947202/01.02.01_60/ts_11947202v010201p.pdf) |
| [ETSI TS 119 472-1 V1.2.1 (2026-02)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=77499) | Profiles for Electronic Attestation of Attributes; Part 1: General requirements | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/11947201/01.02.01_60/ts_11947201v010201p.pdf) |
| [ETSI TS 119 471 V1.1.1 (2025-05)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=63664) | Policy and Security requirements for Providers of Electronic Attestation of Attributes Services | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/119471/01.01.01_60/ts_119471v010101p.pdf) |
| [ETSI TR 119 462 V1.1.1 (2026-03)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=63566) | Wallet interfaces for trust services and signing | [PDF](https://www.etsi.org/deliver/etsi_tr/119400_119499/119462/01.01.01_60/tr_119462v010101p.pdf) |

### Identity proofing for trust-service subjects

These documents support identity proofing and attribute verification discussions, especially around qualified certificates, QEAAs and QTSP-style issuance assurance.

| ETSI deliverable | Short topic | PDF |
|---|---|---|
| [ETSI TS 119 461 V2.1.1 (2025-02)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=68548) | Policy and security requirements for trust service components providing identity proofing of trust service subjects | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/119461/02.01.01_60/ts_119461v020101p.pdf) |
| [ETSI TS 119 461 V1.1.1 (2021-07)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=57792) | Policy and security requirements for trust service components providing identity proofing of trust service subjects | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/119461/01.01.01_60/ts_119461v010101p.pdf) |
| [ETSI TR 119 460 V1.1.1 (2021-02)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=58431) | Survey of technologies and regulatory requirements for identity proofing for trust service subjects | [PDF](https://www.etsi.org/deliver/etsi_tr/119400_119499/119460/01.01.01_60/tr_119460v010101p.pdf) |

### TSP policy and conformity-assessment standards

These documents frame policy requirements for trust service providers and conformity-assessment bodies. They are relevant to CAB/QTSP evidence and assurance-review conversations.

| ETSI deliverable | Short topic | PDF |
|---|---|---|
| [ETSI TR 119 404 V1.1.1 (2023-02)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=66935) | NIS2 and its impact on eIDAS standards | [PDF](https://www.etsi.org/deliver/etsi_tr/119400_119499/119404/01.01.01_60/tr_119404v010101p.pdf) |
| [ETSI TS 119 403-3 V1.1.1 (2019-03)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=54389) | Trust Service Provider Conformity Assessment; Part 3: Additional requirements for conformity assessment bodies assessing EU qualified trust service providers | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/11940303/01.01.01_60/ts_11940303v010101p.pdf) |
| [ETSI TS 119 403-2 V1.3.1 (2023-03)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=66922) | Trust Service Provider Conformity Assessment; Part 2: Additional requirements for Conformity Assessment Bodies auditing Trust Service Providers that issue Publicly-Trusted Certificates | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/11940302/01.03.01_60/ts_11940302v010301p.pdf) |
| [ETSI EN 319 403-1 V2.3.1 (2020-06)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=56885) | Trust Service Provider Conformity Assessment; Part 1: Requirements for conformity assessment bodies assessing Trust Service Providers | [PDF](https://www.etsi.org/deliver/etsi_en/319400_319499/31940301/02.03.01_60/en_31940301v020301p.pdf) |
| [ETSI EN 319 403 V2.2.2 (2015-08)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=39371) | Trust Service Provider Conformity Assessment - Requirements for conformity assessment bodies assessing Trust Service Providers | [PDF](https://www.etsi.org/deliver/etsi_en/319400_319499/319403/02.02.02_60/en_319403v020202p.pdf) |
| [ETSI TS 119 403 V2.2.1 (2015-08)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=47256) | Trust Service Provider Conformity Assessment - Requirements for conformity assessment bodies assessing Trust Service Providers | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/119403/02.02.01_60/ts_119403v020201p.pdf) |
| [ETSI EN 319 401 V3.2.1 (2026-01)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=74317) | General Policy Requirements for Trust Service Providers | [PDF](https://www.etsi.org/deliver/etsi_en/319400_319499/319401/03.02.01_60/en_319401v030201p.pdf) |
| [ETSI EN 319 401 V2.3.1 (2021-05)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=59276) | General Policy Requirements for Trust Service Providers | [PDF](https://www.etsi.org/deliver/etsi_en/319400_319499/319401/02.03.01_60/en_319401v020301p.pdf) |
| [ETSI TS 119 401 V2.0.1 (2015-07)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=47089) | General Policy Requirements for Trust Service Providers | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/119401/02.00.01_60/ts_119401v020001p.pdf) |
| [ETSI TR 119 400 V1.1.1 (2016-03)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=39394) | Guidance on the use of standards for trust service providers supporting digital signatures and related services | [PDF](https://www.etsi.org/deliver/etsi_tr/119400_119499/119400/01.01.01_60/tr_119400v010101p.pdf) |

### CA policy and certificate-profile standards

These documents are relevant when Credimi parses, validates or reports on certificates, qualified certificates, policy requirements, sector-specific certificate profiles or CA trust chains.

| ETSI deliverable | Short topic | PDF |
|---|---|---|
| [ETSI TS 119 495 V1.8.1 (2026-04)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=75421) | Sector Specific Requirements; Certificate Profiles and TSP Policy Requirements for Open Banking | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/119495/01.08.01_60/ts_119495v010801p.pdf) |
| [ETSI TS 119 412-6 V1.2.1 (2026-04)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=77496) | Certificate Profiles; Part 6: Certificate profile requirements for PID, Wallet, EAA, QEAA, and PSBEAA providers | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/11941206/01.02.01_60/ts_11941206v010201p.pdf) |
| [ETSI EN 319 412-5 V2.6.1 (2026-05)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=76590) | Certificate Profiles; Part 5: QCStatements | [PDF](https://www.etsi.org/deliver/etsi_en/319400_319499/31941205/02.06.01_60/en_31941205v020601p.pdf) |
| [ETSI TS 119 412-5 V2.0.13 (2015-07)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=47098) | Certificate Profiles; Part 5: QCStatements | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/11941205/02.00.13_60/ts_11941205v020013p.pdf) |
| [ETSI EN 319 412-4 V1.4.1 (2025-06)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=73791) | Certificate Profiles; Part 4: Certificate profile for web site certificates | [PDF](https://www.etsi.org/deliver/etsi_en/319400_319499/31941204/01.04.01_60/en_31941204v010401p.pdf) |
| [ETSI TS 119 412-4 V1.0.1 (2015-07)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=47097) | Certificate Profiles; Part 4: Certificate profile for web site certificates issued to organizations | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/11941204/01.00.01_60/ts_11941204v010001p.pdf) |
| [ETSI EN 319 412-3 V1.4.0 (2026-04)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=78236) | Certificate Profiles; Part 3: Certificate profile for certificates issued to legal persons | [PDF](https://www.etsi.org/deliver/etsi_en/319400_319499/31941203/01.04.00_20/en_31941203v010400a.pdf) |
| [ETSI TS 119 412-3 V1.0.1 (2015-07)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=47096) | Certificate Profiles; Part 3: Certificate profile for certificates issued to legal persons | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/11941203/01.00.01_60/ts_11941203v010001p.pdf) |
| [ETSI EN 319 412-2 V2.5.0 (2026-05)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=78235) | Certificate Profiles; Part 2: Certificate profile for certificates issued to natural persons | [PDF](https://www.etsi.org/deliver/etsi_en/319400_319499/31941202/02.05.00_20/en_31941202v020500a.pdf) |
| [ETSI TS 119 412-2 V2.0.16 (2015-07)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=47095) | Certificate Profiles; Part 2: Certificate profile for certificates issued to natural persons | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/11941202/02.00.16_60/ts_11941202v020016p.pdf) |
| [ETSI EN 319 412-1 V1.7.1 (2026-05)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=75441) | Certificate Profiles; Part 1: Overview and common data structures | [PDF](https://www.etsi.org/deliver/etsi_en/319400_319499/31941201/01.07.01_60/en_31941201v010701p.pdf) |
| [ETSI TS 119 412-1 V1.4.1 (2020-07)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=59590) | Certificate Profiles; Part 1: Overview and common data structures | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/11941201/01.04.01_60/ts_11941201v010401p.pdf) |
| [ETSI TR 119 411-9 V1.1.1 (2026-03)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=73788) | Policy and security requirements for Trust Service Providers issuing certificates; Part 9: Requirements on a Certificate Transparency (CT) Ecosystem to make the issuing of certificates transparent and verifiable | [PDF](https://www.etsi.org/deliver/etsi_tr/119400_119499/11941109/01.01.01_60/tr_11941109v010101p.pdf) |
| [ETSI TS 119 411-8 V1.1.1 (2025-10)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=74431) | Policy and security requirements for Trust Service Providers issuing certificates; Part 8: Access Certificate Policy for EUDI Wallet Relying Parties | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/11941108/01.01.01_60/ts_11941108v010101p.pdf) |
| [ETSI TS 119 411-6 V1.1.1 (2023-08)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=67990) | Policy and security requirements for Trust Service Providers issuing certificates; Part 6: Requirements for Trust Service Providers issuing publicly trusted S/MIME certificates | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/11941106/01.01.01_60/ts_11941106v010101p.pdf) |
| [ETSI TS 119 411-5 V2.1.1 (2025-02)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=69587) | Policy and security requirements for Trust Service Providers issuing certificates; Part 5: Implementation of qualified certificates for website authentication as in amended regulation 910/2014 | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/11941105/02.01.01_60/ts_11941105v020101p.pdf) |
| [ETSI TR 119 411-5 V1.1.1 (2023-01)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=66167) | Policy and security requirements for Trust Service Providers issuing certificates; Part 5: Guidelines for the coexistence of web browser and EU trust controls | [PDF](https://www.etsi.org/deliver/etsi_tr/119400_119499/11941105/01.01.01_60/tr_11941105v010101p.pdf) |
| [ETSI TR 119 411-4 V1.3.1 (2025-09)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=74864) | Policy and security requirements for Trust Service Providers issuing certificates; Part 4: Checklist supporting audit of TSP against ETSI EN 319 411-1 or ETSI EN 319 411-2 | [PDF](https://www.etsi.org/deliver/etsi_tr/119400_119499/11941104/01.03.01_60/tr_11941104v010301p.pdf) |
| [ETSI EN 319 411-2 V2.6.1 (2025-06)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=72255) | Policy and security requirements for Trust Service Providers issuing certificates; Part 2: Requirements for trust service providers issuing EU qualified certificates | [PDF](https://www.etsi.org/deliver/etsi_en/319400_319499/31941102/02.06.01_60/en_31941102v020601p.pdf) |
| [ETSI TS 119 411-2 V2.0.7 (2015-07)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=47091) | Policy and security requirements for Trust Service Providers issuing certificates; Part 2: Requirements for trust service providers issuing EU qualified certificates | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/11941102/02.00.07_60/ts_11941102v020007p.pdf) |
| [ETSI EN 319 411-1 V1.5.1 (2025-04)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=70003) | Policy and security requirements for Trust Service Providers issuing certificates; Part 1: General requirements | [PDF](https://www.etsi.org/deliver/etsi_en/319400_319499/31941101/01.05.01_60/en_31941101v010501p.pdf) |
| [ETSI TS 119 411-1 V1.0.1 (2015-07)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=47090) | Policy and security requirements for Trust Service Providers issuing certificates; Part 1: General requirements | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/11941101/01.00.01_60/ts_11941101v010001p.pdf) |

### Timestamping standards

These documents support qualified timestamp evidence and time-stamp protocol/policy validation. They are useful for future Level 3 evidence bundles, not the first MVP.

| ETSI deliverable | Short topic | PDF |
|---|---|---|
| [ETSI EN 319 422 V1.1.1 (2016-03)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=39370) | Time-stamping protocol and time-stamp token profiles | [PDF](https://www.etsi.org/deliver/etsi_en/319400_319499/319422/01.01.01_60/en_319422v010101p.pdf) |
| [ETSI TS 119 422 V1.0.1 (2015-07)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=47093) | Time-stamping protocol and time-stamp profiles | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/119422/01.00.01_60/ts_119422v010001p.pdf) |
| [ETSI EN 319 421 V1.3.1 (2025-07)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=70004) | Policy and Security Requirements for Trust Service Providers issuing Time-Stamps | [PDF](https://www.etsi.org/deliver/etsi_en/319400_319499/319421/01.03.01_60/en_319421v010301p.pdf) |
| [ETSI TS 119 421 V1.0.1 (2015-07)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=47092) | Policy and Security Requirements for Trust Service Providers issuing Time-Stamps | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/119421/01.00.01_60/ts_119421v010001p.pdf) |

### Remote signing and validation-service standards

These documents are relevant for remote QSCD/QSealCD, AdES creation and signature/seal validation services, especially where Credimi integrates or ingests qualified validation-service evidence.

| ETSI deliverable | Short topic | PDF |
|---|---|---|
| [ETSI TS 119 442 V1.1.1 (2019-02)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=47246) | Protocol profiles for trust service providers providing AdES digital signature validation services | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/119442/01.01.01_60/ts_119442v010101p.pdf) |
| [ETSI TS 119 441 V1.3.1 (2025-10)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=74891) | Policy requirements for TSP providing signature validation services | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/119441/01.03.01_60/ts_119441v010301p.pdf) |
| [ETSI TS 119 432 V1.3.1 (2026-03)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=73032) | Protocols for remote digital signature creation | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/119432/01.03.01_60/ts_119432v010301p.pdf) |
| [ETSI TS 119 431-2 V1.2.1 (2023-06)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=61997) | Policy and security requirements for trust service providers; Part 2: TSP service components supporting AdES digital signature creation | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/11943102/01.02.01_60/ts_11943102v010201p.pdf) |
| [ETSI TS 119 431-1 V1.3.1 (2024-12)](https://webapp.etsi.org/workprogram/Report_WorkItem.asp?WKI_ID=69586) | Policy and security requirements for trust service providers; Part 1: TSP services operating a remote QSCD / SCDev | [PDF](https://www.etsi.org/deliver/etsi_ts/119400_119499/11943101/01.03.01_60/ts_11943101v010301p.pdf) |

### How the ETSI layer changes the taxonomy

The earlier map had four main dimensions: legal, technical, assurance and trust infrastructure. The ETSI documents make it useful to add a fifth dimension:

```text
Standards / normative technical profiles
```

In practice this means:

```text
Legal requirement
  ↓
ARF / WE BUILD interpretation
  ↓
ETSI / OpenID / ISO / W3C technical profile
  ↓
Credimi evidence claim
  ↓
StepCI / Maestro / external conformance / trust-helper result
```

### Credimi implementation impact

```text
MVP impact
  Low to medium: document and reference the standards, but do not block Wallet automation on them.

Trust-helper impact
  High: certificate profiles, TSP policies, signature validation, timestamps, EAA/PID profiles and relying-party attributes should feed helper checks.

Conformance evidence impact
  High for advanced reports: Credimi can show which ETSI standard family a check is related to, without pretending to certify the full standard.

UX impact
  Keep ETSI standards out of the basic Maestro Action modal. Show them in advanced details, trust-helper results, evidence claim pages and report appendices.
```


## 🟦 15. Privacy, cybersecurity and accessibility context

These horizontal documents shape obligations around data protection, cybersecurity, operational security and accessibility.

| Document | Why it matters |
|---|---|
| [GDPR](https://data.europa.eu/eli/reg/2016/679/oj) | **Personal data protection.** User data, RP requests, erasure, minimisation. |
| [ePrivacy Directive](https://data.europa.eu/eli/dir/2002/58/oj) | **Privacy in electronic communications.** Communications privacy context. |
| [Regulation (EU) 2018/1725](https://data.europa.eu/eli/reg/2018/1725/oj) | **EU-institution data protection.** Commission/list processing context. |
| [NIS2 Directive](https://data.europa.eu/eli/dir/2022/2555/oj) | **Cybersecurity.** Operational/security evidence and supervisory context. |
| [Cyber Resilience Act](https://data.europa.eu/eli/reg/2024/2847/oj) | **Product cybersecurity.** Wallet/software cybersecurity context. |
| [Web Accessibility Directive](https://data.europa.eu/eli/dir/2016/2102/oj) | **Public-sector web/mobile accessibility.** Public-sector Wallet/RP interfaces. |
| [European Accessibility Act](https://data.europa.eu/eli/dir/2019/882/oj) | **Accessibility requirements.** Consumer-facing digital service accessibility. |

---

## 🟦🟧 16. Recommended reading routes

### Route A — Fastest route for Credimi MVP

Read these first:

1. CIR 2024/2982 — protocols and interfaces
2. CIR 2024/2977 — PID and EAA issuance
3. CIR 2025/848 — RP registration and entitlements
4. CIR 2024/2981 — certification / assurance evidence
5. ARF latest
6. WE BUILD WP4 trust use cases

Why: this route covers real Wallet automation, Issuer/Verifier traces, RP trust checks and certification-support evidence.

### Route B — Trust-helper MVP

Read these first:

1. CIR 2025/848 — RP registration and entitlements
2. CID 2015/1505 — Trusted List formats
3. CID 2025/2164 — Trusted List template standard
4. CIR 2024/2980 — machine-readable ecosystem information
5. CIR 2025/1569 — QEAA / public-sector EAAs
6. WE BUILD WP4

Why: this route covers the actual data sources needed to resolve trust, status, entitlements and lists.

### 🟦🟩 Route C — Certification and assurance

Read these first:

1. CIR 2024/2981 — Wallet certification
2. CIR 2025/849 — certified Wallet list
3. CIR 2015/1502 — assurance levels
4. CIR 2025/2162 — CAB accreditation / conformity assessment
5. CIR 2025/1571 — supervisory reporting

Why: this route supports conversations with certification-adjacent people, Member States and auditors.

### Route D — Advanced QTSP / qualified evidence

Read these only when needed:

1. CIR 2025/1929 — qualified timestamps
2. CIR 2025/1942 — qualified validation services
3. CIR 2025/1943 — qualified certificates
4. CIR 2025/1945 — signature/seal validation
5. CIR 2025/2530 — QTSP requirements

Why: this route matters for QTSP partnerships and higher-assurance evidence bundles, but should not block the core Credimi conformance story.

---

## 17. Key takeaways

- The legal framework explains **why** EUDI Wallet actors must behave in specific ways.
- The CIRs define binding operational rules.
- 🟦🟧 ARF and WE BUILD help translate those rules into technical and trust-evaluation scenarios.
- Credimi should focus on producing reproducible evidence, not claiming to replace certification.
- The first valuable Credimi evidence layer is StepCI + Maestro + Temporal + screenshots/videos + normalized reports.
- The second, stronger layer is trust-helper validation: RP entitlements, issuer status, trusted lists, revocation and certificate checks.
- Qualified trust-service integrations are important later, but not MVP blockers.
- The ETSI ESI standards layer gives concrete profiles for EAA/PID, certificates, TSP policies, timestamps, signature validation and trust-service assurance.

---

## 18. Suggested product mapping

| Product area | Primary documents | Credimi feature |
|---|---|---|
| Wallet issuance | 2024/2977, 2024/2982, ARF | StepCI credential offer + Maestro Wallet flow |
| Wallet presentation | 2024/2982, 2025/848, ARF | StepCI presentation request + Maestro approval flow |
| Issuer conformance evidence | 2024/2977, 2025/1569, WE BUILD | Issuer metadata, offer validation, headless tests |
| RP trust evidence | 2025/848, 2024/2982, WE BUILD | RP register/cert/entitlement helper |
| Trusted Lists | 2015/1505, 2025/2164, WE BUILD | LoTL/TL fetch, validate, resolve actor |
| Certification support | 2024/2981, 2025/849, 2015/1502 | Normalized evidence report + hash manifest |
| Advanced assurance | 2025/1929, 2025/1942, 2025/1945 | Optional QTSP timestamp / validation-service integrations |
| ETSI standards layer | 119471/472/475/476/478/479, 319401/403/411/412, 119441/442 | Trust-helper checks, advanced evidence claim references, report appendices |

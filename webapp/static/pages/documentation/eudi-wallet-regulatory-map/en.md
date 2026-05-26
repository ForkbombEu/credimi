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
updatedOn: 2026-05-26
---

# EUDI Trust & Conformance Map

This page is a readable map of the documents around EUDI Wallet conformance. It is meant to help humans understand where the legal, technical and standards pieces fit.

## Legend

| Marker | Meaning |
|---|---|
| 🟦 | Legal / Regulation |
| 🟧 | Technical / Protocol / Implementation |
| 🟩 | Assurance / Certification / Audit |
| 🟪 | Trust Infrastructure / Qualified Trust Services |
| 🟨 | Standards / Normative technical profiles |

---

## Fast reading: the minimum mental model

| Layer | Main sources | What it means for Credimi |
|---|---|---|
| Legal foundation | [EUDI Regulation 2024/1183](https://data.europa.eu/eli/reg/2024/1183/oj), consolidated eIDAS | Defines the legal Wallet ecosystem and actors. |
| Wallet / Issuer / RP rules | CIRs 2024/2977, 2979, 2981, 2982; CIR 2025/848 | Turns legal obligations into implementable Wallet, Issuer and RP expectations. |
| Architecture | [EUDI ARF latest](https://eudi.dev/latest/) | Gives the technical structure, actors, protocols and high-level requirements. |
| OpenID4VC protocol layer | [OID4VCI](https://openid.net/specs/openid-4-verifiable-credential-issuance-1_0.html#section-), [OID4VP](https://openid.net/specs/openid-4-verifiable-presentations-1_0.html#section-), [HAIP](https://openid.net/specs/openid4vc-high-assurance-interoperability-profile-1_0.html#section-) | Defines issuance/presentation/high-assurance interoperability. |
| Credential format layer | [SD-JWT](https://www.ietf.org/archive/id/draft-ietf-oauth-selective-disclosure-jwt-14.html#section-), [SD-JWT VC](https://www.ietf.org/archive/id/draft-ietf-oauth-sd-jwt-vc-13.html#section-), ISO 18013-5/7 | Defines SD-JWT and mdoc/mDL credential/presentation formats. |
| Trust infrastructure | Trusted Lists, RP registers, certificates, OpenID Federation where used | Lets Credimi check “who is trusted to do what”. |
| Evidence layer | Temporal + StepCI + Maestro + trust-helper | Produces machine-readable evidence and later PDF/HTML reports. |

---

## How the standards fit together

The EUDI Wallet documentation stack is easier to understand if read in layers:

```text
Legal framework
  ↓
Commission Implementing Regulations
  ↓
EUDI Architecture and Reference Framework
  ↓
Protocol profiles and credential formats
  ↓
Trust infrastructure, certification and assurance standards
```

The legal acts define the obligations and actors. The ARF explains the target architecture and high-level requirements. OpenID4VC, HAIP, SD-JWT, ISO 18013 and ETSI standards then describe the concrete technical and trust-service mechanisms used to implement and assess the ecosystem.

The same document can matter to several audiences. For example, a Wallet developer reads CIR 2024/2982 for interfaces, an Issuer developer reads it for issuance flows, and an assurance reviewer reads it to understand what evidence should exist.

---

## Reading routes

### I build a Wallet

Read first:

- [CIR 2024/2979 — Wallet integrity and core functionalities](https://data.europa.eu/eli/reg_impl/2024/2979/oj)
- [CIR 2024/2982 — Protocols and interfaces](https://data.europa.eu/eli/reg_impl/2024/2982/oj)
- [EUDI ARF latest](https://eudi.dev/latest/)
- [OID4VCI](https://openid.net/specs/openid-4-verifiable-credential-issuance-1_0.html#section-) / [OID4VP](https://openid.net/specs/openid-4-verifiable-presentations-1_0.html#section-) / [HAIP](https://openid.net/specs/openid4vc-high-assurance-interoperability-profile-1_0.html#section-)
- SD-JWT / SD-JWT VC or ISO 18013-5/7 depending on supported formats

Implementation questions: receive PID/EAA offers; complete issuance; display issued credentials; process presentation requests; display RP identity/requested attributes; require user approval; handle revoked/untrusted/expired issuers or RPs.

### I build an Issuer / Attestation Provider

Read first:

- [CIR 2024/2977 — PID and EAAs](https://data.europa.eu/eli/reg_impl/2024/2977/oj)
- [CIR 2024/2982 — Protocols and interfaces](https://data.europa.eu/eli/reg_impl/2024/2982/oj)
- [OID4VCI](https://openid.net/specs/openid-4-verifiable-credential-issuance-1_0.html#section-)
- [HAIP](https://openid.net/specs/openid4vc-high-assurance-interoperability-profile-1_0.html#section-)
- ETSI EAA/PID profile and trust-service material where applicable

Implementation questions: issuer metadata; valid credential offers; credential configuration matching; grant types; WUA validation; status/revocation; issuance to a real Wallet.

### I build a Verifier / Relying Party

Read first:

- [CIR 2024/2982 — Protocols and interfaces](https://data.europa.eu/eli/reg_impl/2024/2982/oj)
- [CIR 2025/848 — Registration of Wallet-relying parties](https://data.europa.eu/eli/reg_impl/2025/848/oj)
- [OID4VP](https://openid.net/specs/openid-4-verifiable-presentations-1_0.html#section-)
- [HAIP](https://openid.net/specs/openid4vc-high-assurance-interoperability-profile-1_0.html#section-)
- RP registration / access-certificate material
- OpenID Federation if your trust model uses it

Implementation questions: valid presentation requests; requested attributes within entitlement; RP identity/authentication; verifier callback/result; credential signature/status/trust validation.

### I care about certification / assurance

Read first:

- [CIR 2024/2981 — Wallet certification](https://data.europa.eu/eli/reg_impl/2024/2981/oj)
- [CIR 2025/849 — List of certified Wallets](https://data.europa.eu/eli/reg_impl/2025/849/oj)
- [CIR 2015/1502 — Assurance levels](https://data.europa.eu/eli/reg_impl/2015/1502/oj)
- [CIR 2025/2162 — CAB accreditation and conformity assessment](https://data.europa.eu/eli/reg_impl/2025/2162/oj)

Assurance evidence to look for: reproducible inputs/outputs; artifact hashes; reproducible execution evidence; test artifacts; implementation logs; conformance-suite results; optional external/CAB/QTSP evidence.

### I care about trust infrastructure

Read first:

- [CIR 2024/2980 — Notifications](https://data.europa.eu/eli/reg_impl/2024/2980/oj)
- [CIR 2025/848 — RP registration](https://data.europa.eu/eli/reg_impl/2025/848/oj)
- [CIR 2015/1505 — Trusted Lists formats](https://data.europa.eu/eli/dec_impl/2015/1505/oj)
- [Decision 2025/2164 — Trusted List template standard](https://data.europa.eu/eli/dec_impl/2025/2164/oj)
- [OpenID Federation](https://openid.net/specs/openid-federation-1_0-45.html#section-) if federation is used
- ETSI certificate, validation and timestamping standards

Implementation questions: fetch LoTL/TLs; validate signatures/seals and freshness; resolve actor status; validate chains; check revocation/status; compare RP requested attributes with entitlements.

---

## Map by functional area

### 🟦 Wallet legal/core

| Document | Why it matters |
|---|---|
| [CIR 2024/2979](https://data.europa.eu/eli/reg_impl/2024/2979/oj) | Core Wallet functionality, integrity and user-facing behaviour. |
| [CIR 2024/2981](https://data.europa.eu/eli/reg_impl/2024/2981/oj) | Wallet certification and assurance evidence. |
| [CIR 2024/2982](https://data.europa.eu/eli/reg_impl/2024/2982/oj) | Protocols and interfaces used by Wallets. |
| [EUDI ARF latest](https://eudi.dev/latest/) | Architecture and high-level requirements. |

### 🟦🟧 Issuer / PID / EAA / QEAA

| Document | Why it matters |
|---|---|
| [CIR 2024/2977](https://data.europa.eu/eli/reg_impl/2024/2977/oj) | PID/EAA issuance, validation information and status/revocation. |
| [CIR 2025/1569](https://data.europa.eu/eli/reg_impl/2025/1569/oj) | QEAA and public-sector authentic-source EAA rules. |
| [OID4VCI](https://openid.net/specs/openid-4-verifiable-credential-issuance-1_0.html#section-) | Credential issuance protocol. |
| [HAIP](https://openid.net/specs/openid4vc-high-assurance-interoperability-profile-1_0.html#section-) | High-assurance OpenID4VC profile. |

### 🟦🟧 Verifier / RP

| Document | Why it matters |
|---|---|
| [CIR 2025/848](https://data.europa.eu/eli/reg_impl/2025/848/oj) | RP registration, entitlements and certificates. |
| [CIR 2024/2982](https://data.europa.eu/eli/reg_impl/2024/2982/oj) | Protocols/interfaces for presentation and RP interaction. |
| [OID4VP](https://openid.net/specs/openid-4-verifiable-presentations-1_0.html#section-) | Presentation protocol. |
| [HAIP](https://openid.net/specs/openid4vc-high-assurance-interoperability-profile-1_0.html#section-) | High-assurance presentation/interoperability profile. |

### 🟨 Credential formats

| Document | Why it matters |
|---|---|
| [SD-JWT](https://www.ietf.org/archive/id/draft-ietf-oauth-selective-disclosure-jwt-14.html#section-) | Selective disclosure mechanism. |
| [SD-JWT VC](https://www.ietf.org/archive/id/draft-ietf-oauth-sd-jwt-vc-13.html#section-) | SD-JWT-based Verifiable Credentials. |
| ISO/IEC 18013-5 | mDL/mdoc device engagement, data retrieval and security mechanisms. |
| [ISO/IEC 18013-7](https://www.iso.org/standard/82772.html#) | Reference-only here; relevant to mdoc/mDL online presentation but not fully integrated until the document is available. |

### 🟪 Trust infrastructure

| Document | Why it matters |
|---|---|
| [CIR 2024/2980](https://data.europa.eu/eli/reg_impl/2024/2980/oj) | Notification and machine-readable ecosystem information. |
| [CIR 2025/848](https://data.europa.eu/eli/reg_impl/2025/848/oj) | RP registers and entitlements. |
| [CIR 2015/1505](https://data.europa.eu/eli/dec_impl/2015/1505/oj) | Trusted List formats. |
| [Decision 2025/2164](https://data.europa.eu/eli/dec_impl/2025/2164/oj) | Common Trusted List template standard. |
| [OpenID Federation](https://openid.net/specs/openid-federation-1_0-45.html#section-) | Optional/partner-driven federation trust-chain layer. |
| [OAuth Status List](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-status-list-15#section-) | Credential status/revocation where used. |

### 🟩 Assurance / certification / audit

| Document | Why it matters |
|---|---|
| [CIR 2024/2981](https://data.europa.eu/eli/reg_impl/2024/2981/oj) | Wallet certification and evidence/dependency context. |
| [CIR 2025/849](https://data.europa.eu/eli/reg_impl/2025/849/oj) | Certified Wallet list. |
| [CIR 2015/1502](https://data.europa.eu/eli/reg_impl/2015/1502/oj) | eID assurance levels. |
| [CIR 2025/2162](https://data.europa.eu/eli/reg_impl/2025/2162/oj) | CAB accreditation and conformity assessment. |

---

## Full legal and regulatory reference catalogue

This catalogue is intentionally dense. It is here so the page remains complete without forcing readers to treat the whole list as narrative prose.

| ID | Document | Why it matters |
|---|---|---|

| `EU-2014-910` | [Regulation (EU) No 910/2014 — eIDAS Regulation](https://data.europa.eu/eli/reg/2014/910/oj) | Original eIDAS framework for electronic identification and trust services. |
| `EU-2014-910-CONSOLIDATED-2024-10-18` | [Consolidated eIDAS Regulation after EUDI amendments](https://data.europa.eu/eli/reg/2014/910/2024-10-18) | Operational legal baseline after EUDI amendments. |
| `EU-2024-1183` | [Regulation (EU) 2024/1183 — European Digital Identity Framework](https://data.europa.eu/eli/reg/2024/1183/oj) | Creates the European Digital Identity Wallet framework. |
| `EU-2024-2977` | [Commission Implementing Regulation (EU) 2024/2977 — PID and electronic attestations of attributes](https://data.europa.eu/eli/reg_impl/2024/2977/oj) | PID/EAA issuance, validation and status information. |
| `EU-2024-2979` | [Commission Implementing Regulation (EU) 2024/2979 — Wallet integrity and core functionalities](https://data.europa.eu/eli/reg_impl/2024/2979/oj) | Wallet integrity and core user-facing functionality. |
| `EU-2024-2980` | [Commission Implementing Regulation (EU) 2024/2980 — Notifications to the Commission](https://data.europa.eu/eli/reg_impl/2024/2980/oj) | Notification and public/machine-readable ecosystem information. |
| `EU-2024-2981` | [Commission Implementing Regulation (EU) 2024/2981 — Wallet certification](https://data.europa.eu/eli/reg_impl/2024/2981/oj) | Wallet certification, assurance documentation and dependency analysis. |
| `EU-2024-2982` | [Commission Implementing Regulation (EU) 2024/2982 — Protocols and interfaces](https://data.europa.eu/eli/reg_impl/2024/2982/oj) | Issuance, presentation, RP authentication, erasure/reporting interfaces. |
| `EU-2025-846` | [Commission Implementing Regulation (EU) 2025/846 — Cross-border identity matching](https://data.europa.eu/eli/reg_impl/2025/846/oj) | Cross-border identity matching for public services. |
| `EU-2025-847` | [Commission Implementing Regulation (EU) 2025/847 — Reactions to Wallet security breaches](https://data.europa.eu/eli/reg_impl/2025/847/oj) | Security-breach assessment and reaction rules. |
| `EU-2025-848` | [Commission Implementing Regulation (EU) 2025/848 — Registration of Wallet-relying parties](https://data.europa.eu/eli/reg_impl/2025/848/oj) | RP registration, entitlements, access and registration certificates. |
| `EU-2025-849` | [Commission Implementing Regulation (EU) 2025/849 — List of certified Wallets](https://data.europa.eu/eli/reg_impl/2025/849/oj) | Certified Wallet list and related machine-readable data. |
| `EU-2025-1566` | [Commission Implementing Regulation (EU) 2025/1566 — Identity and attribute verification standards](https://data.europa.eu/eli/reg_impl/2025/1566/oj) | Identity/attribute verification before qualified certificates or QEAAs. |
| `EU-2025-1567` | [Commission Implementing Regulation (EU) 2025/1567 — Remote QSCD/QSealCD management](https://data.europa.eu/eli/reg_impl/2025/1567/oj) | Remote qualified signature/seal creation-device management. |
| `EU-2025-1568` | [Commission Implementing Regulation (EU) 2025/1568 — Peer reviews of eID schemes](https://data.europa.eu/eli/reg_impl/2025/1568/oj) | Peer review procedures for eID schemes. |
| `EU-2025-1569` | [Commission Implementing Regulation (EU) 2025/1569 — QEAA and public-sector authentic-source EAAs](https://data.europa.eu/eli/reg_impl/2025/1569/oj) | QEAAs and public-sector authentic-source attestations. |
| `EU-2025-1570` | [Commission Implementing Regulation (EU) 2025/1570 — Certified QSCD/QSealCD notification](https://data.europa.eu/eli/reg_impl/2025/1570/oj) | Notification information for certified qualified signature/seal devices. |
| `EU-2025-1571` | [Commission Implementing Regulation (EU) 2025/1571 — Supervisory body annual reports](https://data.europa.eu/eli/reg_impl/2025/1571/oj) | Annual reporting by supervisory bodies. |
| `EU-2025-1572` | [Commission Implementing Regulation (EU) 2025/1572 — Notification of intention for qualified trust services](https://data.europa.eu/eli/reg_impl/2025/1572/oj) | QTSP intention notification and supervisory verification methodology. |
| `EU-2025-1929` | [Commission Implementing Regulation (EU) 2025/1929 — Qualified electronic time stamps](https://data.europa.eu/eli/reg_impl/2025/1929/oj) | Qualified timestamp standards/specifications. |
| `EU-2025-1942` | [Commission Implementing Regulation (EU) 2025/1942 — Qualified validation services](https://data.europa.eu/eli/reg_impl/2025/1942/oj) | Qualified validation services for qualified signatures/seals. |
| `EU-2025-1943` | [Commission Implementing Regulation (EU) 2025/1943 — Qualified certificate standards](https://data.europa.eu/eli/reg_impl/2025/1943/oj) | Qualified certificate reference standards. |
| `EU-2025-1944` | [Commission Implementing Regulation (EU) 2025/1944 — QERDS and interoperability](https://data.europa.eu/eli/reg_impl/2025/1944/oj) | Qualified electronic registered delivery services. |
| `EU-2025-1945` | [Commission Implementing Regulation (EU) 2025/1945 — Validation of electronic signatures and seals](https://data.europa.eu/eli/reg_impl/2025/1945/oj) | Validation of qualified/advanced signatures and seals. |
| `EU-2025-1946` | [Commission Implementing Regulation (EU) 2025/1946 — Qualified preservation services](https://data.europa.eu/eli/reg_impl/2025/1946/oj) | Qualified preservation of qualified signatures/seals. |
| `EU-2025-2160` | [Commission Implementing Regulation (EU) 2025/2160 — Risk management for non-qualified trust services](https://data.europa.eu/eli/reg_impl/2025/2160/oj) | Risk management for non-qualified trust services. |
| `EU-2025-2162` | [Commission Implementing Regulation (EU) 2025/2162 — CAB accreditation and conformity assessment](https://data.europa.eu/eli/reg_impl/2025/2162/oj) | Accreditation of CABs and conformity assessment of QTSPs. |
| `EU-2025-2164` | [Commission Implementing Decision (EU) 2025/2164 — Trusted List template standard](https://data.europa.eu/eli/dec_impl/2025/2164/oj) | Common Trusted List template standard. |
| `EU-2025-2527` | [Commission Implementing Regulation (EU) 2025/2527 — Qualified certificates for website authentication](https://data.europa.eu/eli/reg_impl/2025/2527/oj) | Qualified website-authentication certificate standards. |
| `EU-2025-2530` | [Commission Implementing Regulation (EU) 2025/2530 — Requirements for QTSPs](https://data.europa.eu/eli/reg_impl/2025/2530/oj) | Requirements for qualified trust service providers. |
| `EU-2025-2531` | [Commission Implementing Regulation (EU) 2025/2531 — Qualified electronic ledgers](https://data.europa.eu/eli/reg_impl/2025/2531/oj) | Qualified electronic ledger standards. |
| `EU-2025-2532` | [Commission Implementing Regulation (EU) 2025/2532 — Qualified electronic archiving services](https://data.europa.eu/eli/reg_impl/2025/2532/oj) | Qualified electronic archiving standards. |
| `EU-2015-1502` | [Commission Implementing Regulation (EU) 2015/1502 — Assurance levels for eID means](https://data.europa.eu/eli/reg_impl/2015/1502/oj) | Low/substantial/high assurance level framework. |
| `EU-2015-1505` | [Commission Implementing Decision (EU) 2015/1505 — Trusted Lists formats](https://data.europa.eu/eli/dec_impl/2015/1505/oj) | Technical specifications and formats for Trusted Lists. |
| `EU-2024-2847` | [Regulation (EU) 2024/2847 — Cyber Resilience Act](https://data.europa.eu/eli/reg/2024/2847/oj) | Cybersecurity requirements for products with digital elements. |
| `EU-2024-482` | [Commission Implementing Regulation (EU) 2024/482 — EUCC cybersecurity certification scheme](https://data.europa.eu/eli/reg_impl/2024/482/oj) | Common Criteria-based cybersecurity certification scheme. |
| `EU-2024-3144` | [Commission Implementing Regulation (EU) 2024/3144 — EUCC amendments/corrections](https://data.europa.eu/eli/reg_impl/2024/3144/oj) | Amendments/corrections to EUCC. |
| `EU-2016-679` | [Regulation (EU) 2016/679 — GDPR](https://data.europa.eu/eli/reg/2016/679/oj) | Personal-data protection framework. |
| `EU-2002-58` | [Directive 2002/58/EC — ePrivacy Directive](https://data.europa.eu/eli/dir/2002/58/oj) | Privacy and electronic communications. |
| `EU-2018-1725` | [Regulation (EU) 2018/1725 — EU institutions data protection](https://data.europa.eu/eli/reg/2018/1725/oj) | EU institutions/bodies data protection. |
| `EU-2022-2555` | [Directive (EU) 2022/2555 — NIS2 Directive](https://data.europa.eu/eli/dir/2022/2555/oj) | Common cybersecurity level across the EU. |
| `EU-2019-881` | [Regulation (EU) 2019/881 — Cybersecurity Act](https://data.europa.eu/eli/reg/2019/881/oj) | ENISA and EU cybersecurity certification framework. |
| `EU-2008-765` | [Regulation (EC) No 765/2008 — Accreditation and market surveillance](https://data.europa.eu/eli/reg/2008/765/oj) | Accreditation and CAB framework. |
| `EU-2016-2102` | [Directive (EU) 2016/2102 — Web Accessibility Directive](https://data.europa.eu/eli/dir/2016/2102/oj) | Public-sector website/mobile accessibility. |
| `EU-2019-882` | [Directive (EU) 2019/882 — European Accessibility Act](https://data.europa.eu/eli/dir/2019/882/oj) | Accessibility requirements for products/services. |

---

## OpenID / OAuth / ISO source-scope note

The OpenID Conformance Suite covers many standards. This EUDI-targeted map only includes the subset relevant to Wallet/Issuer/Verifier conformance.

| Prefix | Source | Link |
|---|---|---|
| `OID4VCI-1FINAL-` | OpenID for Verifiable Credential Issuance 1.0 | [link](https://openid.net/specs/openid-4-verifiable-credential-issuance-1_0.html#section-) |
| `OID4VP-1FINAL-` | OpenID for Verifiable Presentations 1.0 | [link](https://openid.net/specs/openid-4-verifiable-presentations-1_0.html#section-) |
| `HAIP-` | OpenID4VC High Assurance Interoperability Profile | [link](https://openid.net/specs/openid4vc-high-assurance-interoperability-profile-1_0.html#section-) |
| `SDJWT-` | Selective Disclosure JWT | [link](https://www.ietf.org/archive/id/draft-ietf-oauth-selective-disclosure-jwt-14.html#section-) |
| `SDJWTVC-` | SD-JWT VC | [link](https://www.ietf.org/archive/id/draft-ietf-oauth-sd-jwt-vc-13.html#section-) |
| `OTSL-` | OAuth Status List | [link](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-status-list-15#section-) |
| `OIDFED-` | OpenID Federation | [link](https://openid.net/specs/openid-federation-1_0-45.html#section-) |
| `ISO18013-7-` | ISO 18013-7 | [reference-only link](https://www.iso.org/standard/82772.html#) |

Supporting OAuth/OIDC dependencies such as PKCE, PAR, JAR, JARM, DPoP, JWT/JWK/JWA, OAuth AS Metadata and OIDC Discovery should only be attached to concrete tests when directly used. FAPI, CIBA, Open Banking/regional profiles, CAEP/RISC/Shared Signals and AuthZEN are intentionally out of scope for now.

---

## Maintainer note

This page is maintained by Credimi / Forkbomb as a navigation aid for the EUDI Wallet ecosystem.

Credimi uses these standards and legal references internally to structure interoperability and conformance-evidence work, but this page is intentionally focused on documenting the standards landscape rather than Credimi’s internal taxonomies or product architecture.

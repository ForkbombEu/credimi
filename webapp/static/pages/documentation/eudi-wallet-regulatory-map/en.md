---
title: 'Navigating the EUDI wallet rulebook'
description: A practical map of the legal, technical, and certification documents that shape the European Digital Identity Wallet ecosystem.
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
updatedOn: 2026-04-16
---

# Navigating the EUDI wallet rulebook

*Last updated: 2026*

---

# 1. Introduction

This document is a practical guide to the legal, technical, and certification texts behind the European Digital Identity (EUDI) Wallet. It is meant to help readers understand how the main sources relate to one another, what each one is for, and where to start reading.

The ecosystem is built on three main layers:

* **eIDAS Regulation**: the legal foundation
* **Commission Implementing Regulations (CIRs)**: binding technical and operational rules
* **Architecture and Reference Framework (ARF)**: the technical blueprint

Together, these define how digital identity wallets must be designed, implemented, certified, and operated across the European Union.

---

## Legend

* 🟦 Legal / Regulation (primarily legal audience)
* 🟧 Technical / Protocol / Implementation (primarily engineering audience)
* 🟦🟧 Both (relevant to both legal and technical roles)

---

# 2. The legal foundation: eIDAS 2.0

The revised eIDAS Regulation establishes the legal basis for the European Digital Identity Wallet. It defines the roles, obligations, and rights of all actors involved, including users, wallet providers, relying parties, and supervisory authorities.

It introduces the concept of the EUDI Wallet and mandates that Member States provide citizens with a digital identity solution that can be used across borders.

* 🟦 eIDAS 2.0 Regulation:
  https://eur-lex.europa.eu/eli/reg/2024/2847/oj

---

# 3. The technical blueprint: ARF

The Architecture and Reference Framework is a non-binding but authoritative technical specification. It describes how the EUDI Wallet ecosystem should be implemented in practice.

The ARF defines:

* System architecture (Wallet, Issuer, Relying Party, Supervisory Authorities)
* Data models (e.g. Personal Identification Data, Electronic Attestations of Attributes)
* Protocol flows (issuance, presentation, verification)
* Security and privacy requirements

Although not legally binding, the ARF is effectively enforced through certification processes defined in the CIRs.

* 🟧 ARF GitHub repository:
  https://github.com/eu-digital-identity-wallet/eudi-doc-architecture-and-reference-framework

* 🟧 Rendered documentation:
  https://eu-digital-identity-wallet.github.io/eudi-doc-architecture-and-reference-framework/

---

# 4. The operational rulebook: CIRs

Commission Implementing Regulations translate the high-level requirements of eIDAS into concrete, legally binding rules. They define how wallets must operate, how credentials are issued and verified, and how the ecosystem is governed.

Each CIR focuses on a specific aspect of the system. Together, they form the operational rulebook for the EUDI Wallet.

---

## 4.1 2024 wallet rules

These regulations define the fundamental components of the wallet: identity data, functionality, certification, and communication protocols.

* **Identity data and attestations**  
  🟦🟧 [CIR 2024/2977](https://eur-lex.europa.eu/eli/reg_impl/2024/2977/oj)  
  How PID and EAA data are structured and issued.

* **Wallet integrity and functionality**  
  🟦🟧 [CIR 2024/2979](https://eur-lex.europa.eu/eli/reg_impl/2024/2979/oj)  
  Minimum requirements for wallet behavior, integrity, and reliability.

* **Ecosystem registration and notifications**  
  🟦 [CIR 2024/2980](https://eur-lex.europa.eu/eli/reg_impl/2024/2980/oj)  
  How wallet providers and relying parties are registered and recognized.

* **Certification framework (CAB/NAB)**  
  🟦🟧 [CIR 2024/2981](https://eur-lex.europa.eu/eli/reg_impl/2024/2981/oj), [PDF](https://eur-lex.europa.eu/legal-content/EN/TXT/PDF/?uri=OJ:L_202402981)  
  How wallets are evaluated and certified before deployment.

* **Protocols and interfaces**  
  🟧 [CIR 2024/2982](https://eur-lex.europa.eu/eli/reg_impl/2024/2982/oj)  
  How issuance and presentation systems communicate.

---

## 4.2 2025 ecosystem rules

These regulations govern how the ecosystem operates across Member States, including security, interoperability, and participant registration.

* **Cross-border identity matching**  
  🟦🟧 [CIR 2025/846](https://eur-lex.europa.eu/eli/reg_impl/2025/846/oj)  
  Identity matching across Member States.

* **Security and incident handling**  
  🟦🟧 [CIR 2025/847](https://eur-lex.europa.eu/eli/reg_impl/2025/847/oj)  
  Security obligations and incident response.

* **Relying party registration**  
  🟦 [CIR 2025/848](https://eur-lex.europa.eu/eli/reg_impl/2025/848/oj)  
  Registration and recognition of relying parties.

* **Certified wallet listings**  
  🟦 [CIR 2025/849](https://eur-lex.europa.eu/eli/reg_impl/2025/849/oj)  
  Publication and management of certified wallet lists.

---

## 4.3 2025 advanced and trust service rules

These regulations extend the framework into trust services and advanced identity verification mechanisms.

* **Identity verification**  
  🟦🟧 [CIR 2025/1566](https://eur-lex.europa.eu/eli/reg_impl/2025/1566/oj)  
  Advanced identity verification requirements.

* **Remote QSCD management**  
  🟧 [CIR 2025/1567](https://eur-lex.europa.eu/eli/reg_impl/2025/1567/oj)  
  Technical rules for remote QSCD operation and management.

* **Electronic attestations from public sources**  
  🟦🟧 [CIR 2025/1569](https://eur-lex.europa.eu/eli/reg_impl/2025/1569/oj)  
  Reuse and issuance of attestations sourced from public records.

* **Certified device notification**  
  🟦 [CIR 2025/1570](https://eur-lex.europa.eu/eli/reg_impl/2025/1570/oj)  
  Notification duties around certified devices.

* **Supervisory reporting**  
  🟦 [CIR 2025/1571](https://eur-lex.europa.eu/eli/reg_impl/2025/1571/oj)  
  Reporting obligations toward supervisory authorities.

* **Trust service initiation**  
  🟦🟧 [CIR 2025/1572](https://eur-lex.europa.eu/eli/reg_impl/2025/1572/oj)  
  Rules for starting trust services from the wallet context.

---

## 4.4 Signature and trust infrastructure rules

These regulations define the technical trust layer, including signatures, certificates, and validation services.

* **Time-stamping**  
  🟧 [CIR 2025/1929](https://eur-lex.europa.eu/eli/reg_impl/2025/1929/oj)  
  Time-stamp services and related technical requirements.

* **Validation services**  
  🟧 [CIR 2025/1942](https://eur-lex.europa.eu/eli/reg_impl/2025/1942/oj)  
  Validation service rules and interoperability expectations.

* **Certificate standards**  
  🟧 [CIR 2025/1943](https://eur-lex.europa.eu/eli/reg_impl/2025/1943/oj)  
  Certificate profiles and technical standards.

* **Registered electronic delivery**  
  🟧 [CIR 2025/1944](https://eur-lex.europa.eu/eli/reg_impl/2025/1944/oj)  
  Delivery service rules and trust requirements.

* **Signature validation**  
  🟧 [CIR 2025/1945](https://eur-lex.europa.eu/eli/reg_impl/2025/1945/oj)  
  Technical validation rules for signatures.

---

# 5. How certification works (CABs and NABs)

The certification framework ensures that wallets meet the required security, privacy, and interoperability standards before they are deployed.

* **Conformity Assessment Bodies (CABs)** perform technical evaluations of wallet solutions
* **National Accreditation Bodies (NABs)** accredit CABs
* **Supervisory Authorities** oversee the ecosystem

Certification is mandatory and combines legal compliance and technical validation.

* 🟦🟧 [CIR 2024/2981](https://eur-lex.europa.eu/eli/reg_impl/2024/2981/oj)

---

# 6. How it all fits together

The EUDI Wallet ecosystem is structured as a layered system:

* [eIDAS Regulation](https://eur-lex.europa.eu/eli/reg/2024/2847/oj) (🟦)  
  Defines the legal foundation.

* [CIRs](https://eur-lex.europa.eu/search.html?scope=EURLEX&text=%22European+Digital+Identity+Wallet%22+implementing+regulation) (🟦🟧)  
  Define the binding operational rules.

* [ARF](https://eu-digital-identity-wallet.github.io/eudi-doc-architecture-and-reference-framework/) (🟧)  
  Defines the technical architecture.

* [Certification](https://eur-lex.europa.eu/eli/reg_impl/2024/2981/oj) (🟦🟧)  
  Ensures compliance before deployment.

* Deployment  
  Wallets operate across the EU.

---

# 7. Key Takeaways

* Legal and technical layers are tightly coupled
* CIRs are binding and drive both compliance and implementation
* ARF provides the technical model used during certification
* Certification is the enforcement point between law and engineering
* The ecosystem requires collaboration between legal and technical teams

---

# 8. Suggested Reading Order

1. 🟧 [CIR 2024/2982](https://eur-lex.europa.eu/eli/reg_impl/2024/2982/oj) (protocols and interfaces)
2. 🟦🟧 [CIR 2024/2977](https://eur-lex.europa.eu/eli/reg_impl/2024/2977/oj) (identity and attestations)
3. 🟦🟧 [CIR 2024/2981](https://eur-lex.europa.eu/eli/reg_impl/2024/2981/oj) (certification)
4. 🟧 [ARF latest version](https://eu-digital-identity-wallet.github.io/eudi-doc-architecture-and-reference-framework/)
5. 🟦🟧 [CIR 2025/847](https://eur-lex.europa.eu/eli/reg_impl/2025/847/oj) (security and incidents)

---

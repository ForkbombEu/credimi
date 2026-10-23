---
title: "📜 Privacy Policy"
description: ""
---
**Effective Date:** 24 October 2026  
**Applies to:** Credimi  
**Data Controller:** Forkbomb B.V.  
**Note:** Some of this document was generated with the help of AI

---

## 💻 Credimi app

### 1. Introduction

This Privacy Policy explains how **Forkbomb B.V.** ("we", "our", "us") collects, uses, and protects your personal data when you use the **Credimi app** ("the Dashboard"). By accessing or using the Dashboard, you agree to the terms of this policy.

### 2. Data Collection

#### Personal Data
The Dashboard may collect the following types of personal data:
- Email address
- IP address
- Billing information

#### Collection Methods
Data is collected through:
- Direct user input (e.g., registration or settings)
- Invitations sent to new users via email

The Dashboard does **not** collect personal data from third-party sources.

#### Legal Basis for Processing
The legal bases for processing your personal data are:

- **Contract performance** (Art. 6(1)(b)): creating and managing your account, enabling organization access, billing.
- **Legitimate interests** (Art. 6(1)(f)): securing access (IP logging on authentication), preventing fraud and abuse (Cloudflare Turnstile), and analytics (Matomo). We have a legitimate interest in protecting our service from automated abuse and measuring legitimate usage.
- **Consent** (Art. 6(1)(a) and Art. 4(11)): where we rely on consent, we will ask you directly and you may withdraw at any time without affecting lawfulness of processing before withdrawal.

### 3. Data Processing

#### Purpose
Collected data is used for the following purposes:
- Creating and managing user accounts
- Enabling organization-level access and permissions
- Personalizing the user experience
- Administrative communication and billing

#### Automated Decision-Making
We do **not** use your data for automated decision-making or profiling.

#### Data Sharing
We do **not** sell or share your personal data with external parties for marketing or commercial purposes.

#### Subprocessors
We engage the following subprocessors for the purposes described in this policy:

| Service | Country | Purpose | Legal basis |
|---|---|---|---|
| Cloudflare (Turnstile) | United States / EU (Cloudflare network) | Bot detection and fraud prevention on registration and login forms | Legitimate interests |
| Matomo (self-hosted at matomo.dyne.org) | Netherlands | Analytics to understand service usage | Legitimate interests |

#### Third-Party Services for Testing and Compliance
To ensure technical interoperability, standards compliance, and security, we may use automated testing tools that send pseudonymized or synthetic data to third-party services. For example, we use the [OpenID Conformance Test Suite](https://www.certification.openid.net/) to test compatibility with identity protocols.

These activities:
- Use non-production or synthetic test data
- Are executed in controlled environments
- Are strictly limited to development, testing, and certification purposes

### 4. Data Storage & Security

#### Storage Location
All personal data is stored on secure, cloud-based infrastructure located in the European Union.

#### Data Retention
We retain your personal data only for as long as necessary to fulfill the purposes described in this policy, or until you request deletion, whichever comes first. Pipeline result files (including evidence) are automatically deleted on a schedule configured by the operator (minimum 1 day), and you may request immediate deletion of your account data at any time.

#### Security Measures
We implement appropriate technical and organizational measures to protect your data, including:
- Encrypted storage and communication (TLS, HTTPS)
- Access control and authentication
- Regular software updates and vulnerability scans

### 5. User Rights

You have the following rights under GDPR and other applicable privacy laws:

- **Access**: Request a copy of your personal data
- **Rectification**: Correct inaccurate or incomplete data
- **Erasure** ("right to be forgotten"): Request deletion of your data
- **Restrict Processing**: Request limitation of processing under Art. 18
- **Data Portability**: Receive your data in a structured, machine-readable format (Art. 20)
- **Object**: Object to processing based on legitimate interests (Art. 21)
- **Withdraw Consent**: Withdraw any consent given, without affecting lawfulness of processing before withdrawal (Art. 7.3)

#### How to Exercise Your Rights
You can manage your data directly through the Dashboard interface or by contacting us at:

**Email:** info@forkbomb.eu

#### Identity Verification
At present, we do not require identity verification for exercising rights, but we may implement a **KYC (Know Your Customer)** process in the future.

#### Right to Lodge a Complaint
You have the right to lodge a complaint with a supervisory authority, in particular the Dutch Data Protection Authority (Autoriteit Persoonsgegevens), if you believe your personal data has been processed in violation of the GDPR.

### 6. Cookies, Local Storage & Analytics

The Dashboard uses cookies, web storage and an analytics service to improve functionality and analyze usage. No personal data is read from these via third parties.

#### Cookies

| Cookie | Domain | Purpose | Expiry |
|---|---|---|---|
| `pb_auth` | `credimi.io` | PocketBase authentication token and model, kept in sync with the client-side auth store. | Browser session (cleared on logout; not `httpOnly` or `secure` in the browser client export) |
| `sidebar:state` | `credimi.io` | Remembers the collapsible sidebar state in the authenticated dashboard app. | 24 hours |
| `PARAGLIDE_LOCALE` | `credimi.io` | Stores the active language locale chosen by the user, set by the Paraglide i18n library. | 400 days |

#### Web Storage
The following `localStorage` and `sessionStorage` keys are used to persist non-sensitive UI and session state:

| Key | Scope | Purpose | Expiry |
|---|---|---|---|
| `keyring` | `localStorage` | Encrypted local keypair cache for the user's identity (keypairoom). | Persistent |
| `pipeline_custom_integration_configs` | `localStorage` | Saved configuration for custom-integration pipeline steps. | Persistent |
| `credimi-latestCheckRuns` | `localStorage` | Recently run conformance checks, for quick recall on the start page. | Persistent |
| `welcome` | `sessionStorage` | Tracks whether the first-run welcome modal has been dismissed. | Session end |
| `organization_invite` | `sessionStorage` | Temporary state for accepting an organization invite flow. | Session end |
| `keypairoom_regenerate` | `sessionStorage` | Tracks the regenerate-keyring session step. | Session end |

#### Analytics
We use **Matomo** (self-hosted, `matomo.dyne.org`) for analytics purposes. Matomo respects "Do Not Track" settings in your browser.

#### Managing Cookies & Storage
You can manage or disable cookies and clear web storage via your browser preferences. Clearing storage may sign you out of the Dashboard or reset UI preferences.

### 7. International Data Transfers

We do **not** transfer your personal data outside the European Economic Area (EEA). Cloudflare's Turnstile service (used for bot detection on login and registration forms) processes data on our behalf; Cloudflare is committed to the EU-U.S. Data Privacy Framework and Standard Contractual Clauses.

### 8. Minors

The Dashboard is not intended for children under the age of 16. We do not knowingly collect data from minors.

### 9. Updates to This Policy

We may update this Privacy Policy from time to time. Users will be notified of significant changes via email.

### 10. Contact

If you have questions, requests, or concerns about this Privacy Policy, please contact us at:

**Forkbomb B.V.**  
Haparandadam 7-A1  
1013AK Amsterdam  
Netherlands  
Email: info@forkbomb.eu

### 11. Legal Compliance

We comply with the **General Data Protection Regulation (GDPR)** and applicable privacy laws in the Netherlands.

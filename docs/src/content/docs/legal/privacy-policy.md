---
title: "📜 Privacy Policy"
description: "How Forkbomb B.V. handles personal data, cookies, analytics and browser storage on Credimi."
---
**Effective Date:** 24 October 2026  
**Applies to:** Credimi  
**Data Controller:** Forkbomb B.V.  
**Note:** Some of this document was generated with the help of AI

---

## 💻 Credimi app

### 1. Introduction

This Privacy Policy explains how **Forkbomb B.V.** ("we", "our", "us") processes personal data when you visit [credimi.io](https://credimi.io/) or use its dashboard, APIs and hosted testing services.

### 2. Data Collection

#### Personal Data
Depending on which features you use, we process:

- **Account and organization data:** name, email address, sign-in and verification information, invitations, organization memberships and roles, and billing information where applicable.
- **Service and test data:** pipeline definitions and inputs, uploaded wallet installers, integration settings, run history, reports, logs, screenshots and recordings. These can contain personal data if it is included in your test inputs or captured during a run.
- **Technical and diagnostic data:** IP address, browser or device information, and request and error logs. See [Cookies, local storage and diagnostics](#6-cookies-local-storage-and-diagnostics).

#### Collection Methods
We receive data from you, from other users who invite you to an organization, and from your browser and the systems involved in your tests. If you choose an available third-party sign-in provider, we also receive account information from that provider.

#### Legal Basis for Processing
The legal bases for processing your personal data are:

- **Contract performance** (Art. 6(1)(b)): creating and managing your account, enabling organization access, billing.
- **Legitimate interests** (Art. 6(1)(f)): securing access, preventing fraud and abuse, and diagnosing service failures.
- **Consent** (Art. 6(1)(a) and Art. 4(11)): where we rely on consent, we will ask you directly and you may withdraw at any time without affecting lawfulness of processing before withdrawal.

### 3. Data Processing

#### Purpose
We use this data to manage accounts and organization access; run and troubleshoot tests; store and display results; protect and improve the service; and communicate with users, including about billing where applicable. If you enable pipeline notifications, we use your browser's push subscription to send run updates through its push service. You can turn notifications off in Credimi or revoke permission in your browser.

#### Automated Decision-Making
We do not use personal data to make solely automated decisions that produce legal or similarly significant effects. Automated security checks can block an attempted sign-in or registration.

#### Data Sharing
We do not sell your personal data or share it for third-party marketing. If you publish a resource on the Hub, its published information is visible to other visitors. Wallet installers are offered for download when the version is made downloadable.

#### Service providers

| Service | Purpose |
|---|---|
| [Cloudflare Turnstile](https://www.cloudflare.com/turnstile-privacy-policy/) | Bot and abuse prevention on sign-in and registration. Cloudflare receives browser and network signals such as IP address and user agent. |

#### Third-Party Services for Testing and Compliance
When you run a test, Credimi may send configured inputs to issuers, verifiers, other systems under test, certification services such as the [OpenID Conformance Test Suite](https://www.certification.openid.net/), or the runner that executes your test. Use synthetic or non-production data where possible. Do not include personal data or secrets unless you are authorized to use them and they are needed for the test.

If you use the Action Author, session messages and test context needed to generate an action may be sent to the AI model provider configured on the runner. The session transcript and review decisions may be stored in workflow history.

### 4. Data Storage & Security

#### Data Retention
We retain personal data as long as needed for the purposes described in this policy. Where an operator enables pipeline retention, the configured schedule removes older result files and evidence from result records. It does not delete the run record or its separate workflow history. You may request erasure of personal data as described below.

#### Security Measures
We implement appropriate technical and organizational measures to protect your data, including:
- Encryption in transit (TLS/HTTPS)
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

### 6. Cookies, local storage and diagnostics

Credimi uses browser storage for sign-in, preferences and specific features. The cookies below are set on `credimi.io` when the relevant feature is used.

#### Cookies

| Cookie | Domain | Purpose | Expiry |
|---|---|---|---|
| `pb_auth` | `credimi.io` | Holds your PocketBase sign-in token and account record. Also authenticates embedded views such as run history and Action Author chat. | When the sign-in token expires or you sign out |
| `sidebar:state` | `credimi.io` | Remembers the dashboard sidebar state. | 7 days |
| `PARAGLIDE_LOCALE` | `credimi.io` | Remembers your active language. | 400 days |

Matomo measures page views and link interactions only if you select **Accept analytics** in the banner or explicitly opt in using the optional analytics checkbox at sign-up. Signing up without opting in does not grant consent. Matomo sends usage data, including page URLs and referrers, to `matomo.dyne.org`. It can set `_pk_id.12.*` (13 months), `_pk_ses.12.*` (30 minutes) and `_pk_ref.12.*` (6 months, when applicable).

Credimi remembers your choice in `localStorage` under `credimi-analytics-consent`. Use **Analytics settings** to change your choice. Rejecting after acceptance stops future tracking and clears Matomo analytics cookies on this site.

#### Web Storage
The browser also uses local storage (no automatic expiry) and session storage (normally cleared when the tab closes). These are not cookies.

| Key | Scope | Purpose | Expiry |
|---|---|---|---|
| `pocketbase_auth` | `localStorage` | PocketBase browser sign-in token and account record. Sign-out clears the stored credentials. | No automatic storage expiry; the token expires separately |
| `mode-watcher-mode`, `mode-watcher-theme` | `localStorage` | Appearance state. | Persistent |
| `dark mode` | `localStorage` | Sets embedded run history to use a light theme. | Persistent |
| `keyring` | `localStorage` | Stores the private keyring on your device if you use Keypairoom. | Persistent |
| `pipeline_custom_integration_configs` | `localStorage` | Remembers parameters entered for custom integrations. | Persistent |
| `Credimi-latestCheckRuns` | `localStorage` | Remembers recent conformance check runs. | Persistent |
| `welcome` | `sessionStorage` | Tracks whether the first-run welcome modal has been dismissed. | Session end |
| `organization_invite` | `sessionStorage` | Temporary state for accepting an organization invite flow. | Session end |
| `login_return_to` | `sessionStorage` | Holds a return path so you can continue after signing in. | Session end |
| `keypairoom_regenerate` | `sessionStorage` | Tracks the regenerate-keyring session step. | Session end |

#### Diagnostics

Where browser diagnostics are enabled by the service operator, Credimi collects error reports and traces for requests to Credimi. These can include the page path, error details and your account ID if signed in. Browser diagnostics do not require an analytics cookie.

#### Managing Cookies & Storage
You can block or clear cookies and site data in your browser. Clearing data may sign you out, reset preferences or remove a locally stored keyring.

### 7. International Data Transfers

Providers and external systems involved in sign-in, security checks or tests may process data outside the European Economic Area (EEA). Where we transfer personal data subject to the GDPR outside the EEA, the GDPR's transfer rules apply. Cloudflare describes its transfer mechanisms in its [privacy policy](https://www.cloudflare.com/privacypolicy/). Check the privacy information of external systems you choose to connect before sending personal data to them.

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

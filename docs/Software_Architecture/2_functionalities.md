<!--
SPDX-FileCopyrightText: 2024 Puria Nafisi Azizi
SPDX-FileCopyrightText: 2024 Puria Nafisi Azizi
SPDX-FileCopyrightText: 2024 The Forkbomb Company

SPDX-License-Identifier: CC-BY-NC-SA-4.0
-->

# 📋 Planned Functionalities

## 1. Credential Issuer and Verifier Compliance Testing

**Description**:
The core functionality of credimi is to allow developers to submit their credential issuance and verification services for compliance testing against multiple standards, such as OpenID4VCI, OpenID4VP, and OpenID Federation. The service will assess cryptographic methods, data formats, and communication protocols.

**Motivation**:
Given the fragmentation in standards across decentralized identity ecosystems, developers need an easy way to verify that their services comply with required standards. This ensures interoperability and trust within the ecosystem.

## 2. Periodic Compliance Checks

**Description**:
Offer developers and providers the option to schedule periodic checks to ensure ongoing compliance of their services with evolving standards. Developers can opt-in for these checks, and notifications will be sent if any issues are detected over time.

**Motivation**:
Standards and best practices evolve over time, and a service that was compliant last year may no longer be. Periodic checks help maintain compliance, which is critical for ongoing trust and interoperability.

## 3. Privacy-First Data Publishing

**Description**:
Ensure that any data or results from the compliance checks are only published or made publicly visible if the developers or service providers explicitly consent to it. By default, all data remains private.

**Motivation**:
Privacy is a cornerstone of trust. This feature ensures that developers and providers maintain control over their data, fostering a more trustworthy environment.

## 4. Debugging and Monitoring Tool

**Description**:
Provide a robust debugging tool that can be deployed locally or as a microservice. This tool will help developers identify issues in real-time during credential issuance or verification processes, enabling faster troubleshooting and iteration.

**Motivation**:
Debugging decentralized identity solutions can be complex. This tool simplifies the process by offering real-time insights, allowing developers to fix issues quickly and efficiently.

## 5. Interoperability Testing

**Description**:
Enable developers to test their services for interoperability with other services in the ecosystem. This functionality includes running tests against various identity wallets and verification tools to ensure seamless integration.

**Motivation**:
Interoperability is key to a cohesive decentralized identity ecosystem. This feature ensures that services can work together, providing a smooth user experience across different platforms and tools.

## 6. Report Generation and Export

**Description**:
Allow users to generate detailed reports on compliance checks, debugging sessions, and interoperability tests. These reports can be exported in various formats (e.g., PDF, JSON, CSV) for documentation and analysis.

**Motivation**:
Reports are essential for documentation, compliance audits, and internal reviews. This functionality provides users with the ability to easily generate and share these reports with stakeholders.

## 7. Developer Dashboard

**Description**:
Create a comprehensive dashboard where developers can manage their services, view compliance statuses, schedule periodic checks, and access reports. The dashboard will also provide insights and recommendations based on test results.

**Motivation**:
A centralized dashboard streamlines the management of identity services, providing developers with a user-friendly interface to monitor and improve their offerings.

## 8. End-User Comparison Tool

**Description**:
Develop a comparison tool for end-users that allows them to browse and compare different credential issuance and verification services based on their compliance status, interoperability, and other relevant metrics.

**Motivation**:
End-users need the ability to make informed decisions about which identity services to use. This tool empowers them by providing transparent, easy-to-understand comparisons.

## 9. Plugin System for Extensibility

**Description**:
Implement a plugin system that allows for the integration of additional checks and standards as the ecosystem evolves. This system will enable third-party developers to contribute plugins that extend the functionality of credimi.

**Motivation**:
As the decentralized identity space grows, new standards and requirements will emerge. A plugin system ensures that credimi remains flexible and can adapt to future needs without requiring significant core changes.

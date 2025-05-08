---
# https://vitepress.dev/reference/default-theme-home-page
layout: home

hero:
    name: "Credimi"
    tagline: "The go-to platform for testing, validating, and ensuring EUDI-ARF compliance of wallet, issuers, relying parties, using multiple testing suites."
    text: "\nEUDI-ARF 1.8 §7, \nwallet certification: \nhere we go!"
    image:
        src: https://raw.githubusercontent.com/ForkbombEu/credimi/main/docs/images/logo/credimi_logo-transp_emblem.png
    actions:
        - theme: brand
          text: 🕹 Get started
          link: /Manual/
        - theme: alt
          text: 🐝 API reference
          target: _self
          link: /API/index.html

features:
    - title: Credential Issuer and Verifier Compliance Testing
      details: Test OpenID4VCI and OpenID4VP services and applications for interoperability, with periodic scheduling debugging and report
    - title: Developer Dashboard
      details: developers can manage their services, view compliance statuses, schedule periodic checks, and access reports
    - title: Plugin System for Extensibility
      details: integration of additional checks and standards, enable third-party developers to contribute
    - title: Marketplace
      details: listing and comparison tool for end-users that allows them to browse and compare different digital identity products and services
---

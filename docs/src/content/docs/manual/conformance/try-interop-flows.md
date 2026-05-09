---
title: Try Interop Flows
description: Test Wallet interoperability with third-party Issuers and Verifiers using Credimi.
---

Conformance checks are structured suite-driven tests. Interop flows are closer to real operational testing between components.

Typical examples:

- a Wallet developer testing against third-party Issuers
- an Issuer developer checking how their offer works with different Wallets
- a Verifier developer testing how different Wallets answer a presentation request

## Start from published components

The easiest way to begin is to use marketplace entries that already expose a live StepCI integration.

## Manual flow

A common manual interop flow is:

1. open a credential or verification page
2. generate the QR code or deeplink
3. continue the flow in the target Wallet
4. observe behavior and compare outcomes across components

## Move from manual to automated

Once the same steps are stable, they can be moved into Credimi automation by:

- keeping the StepCI integration
- adding Maestro actions for Wallet behavior
- composing them in a pipeline

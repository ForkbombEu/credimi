---
title: Create Maestro Actions
description: Add Maestro actions that automate Wallet behavior inside Credimi.
---

Maestro is the UI automation layer used by Credimi for Wallet-side behavior.

A Maestro action typically contains steps such as:

- open the app
- complete onboarding
- enter a PIN
- open a deeplink
- navigate to a screen
- accept or present a credential

![Maestro editor](../images/maestro-editor.png)

## Typical action categories

Common reusable actions include:

- onboarding
- receive credential from deeplink
- present credential to a verifier
- utility actions for navigation or reset

## Why create actions first

In Credimi, pipelines are composed from assets that already exist.  
That means you usually create:

1. StepCI integrations
2. Maestro actions
3. the pipeline that combines them

> Placeholder: later add one short subsection with Maestro Studio and a link to the command reference.

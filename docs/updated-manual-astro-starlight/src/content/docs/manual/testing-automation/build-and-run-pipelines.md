---
title: Build and Run Pipelines
description: Compose StepCI and Maestro assets into a repeatable end-to-end pipeline.
---

A pipeline is assembled from assets that already exist in Credimi.

That is why the practical order is:

1. integrate an Issuer or Verifier with StepCI
2. create the required Maestro actions
3. compose them in a pipeline

## Open the pipelines page

From the Pipelines section, create a new pipeline.

![Pipelines list](../images/pipelines-control.png)

![Pipeline editor](../images/pipeline-editor.png)

## Compose the sequence

A common flow is:

1. Wallet onboarding (Maestro)
2. Credential offer generation (StepCI)
3. Credential receipt on device (Maestro)

Later, you can extend the pipeline with:

- verifier flows
- conformance checks
- multiple issuance or verification steps

## Choose the runner and execute

When launching a pipeline, select the runner where the mobile automation should execute.

![Runner selection](../images/select-pipeline-runner.png)

Then run the pipeline from the list view.

---
title: What is a runner
description: Local execution agent used to run automated mobile testing pipelines.
---

A **Credimi runner** is a local execution agent that connects your devices to the Credimi platform.

It acts as a bridge between:

Credimi (PocketBase + Temporal)
↓
credimi-runner (local agent)
↓
Device / Emulator / Simulator


---

## What a runner does

A runner is responsible for:

- receiving execution requests from Credimi  
- running pipelines on a connected device  
- executing StepCI and Maestro steps  
- collecting logs, outputs and artifacts  

All executions are triggered remotely from the Credimi web interface.

---

## One runner = one device

Each runner is associated with a single device:

- Android phone  
- Android emulator  
- Redroid instance  
- iOS simulator  

This ensures:

- isolation between runs  
- predictable performance  
- compatibility with Maestro execution  

---

## Execution model

A runner executes **one pipeline at a time**.

If multiple executions are triggered:

- they are queued automatically  
- execution order is managed by Credimi  

Pipelines can be triggered in different ways:

- manually from the UI  
- scheduled (e.g. periodic runs)  
- via CI/CD  

---

## Fully remote execution

Once a runner is configured:

- no manual interaction is required  
- the device is controlled entirely by Credimi  
- pipelines run unattended  

Typical flow:

1. user presses **Run** in the Credimi UI  
2. the runner receives the job  
3. the device is prepared automatically  
4. the pipeline is executed  
5. results are sent back to Credimi  

---

## Public and private runners

Runners can be:

- **public** → available to all users  
- **private** → available only to the owner  

---

## Stability and lifecycle

Runners are designed to be long-lived:

- once configured, they can run for extended periods  
- devices are prepared automatically for each execution  
- emulator-based setups use ephemeral environments  

Common maintenance events include:

- device OS updates  
- browser updates  
- periodic image refresh (for emulator-based setups)  
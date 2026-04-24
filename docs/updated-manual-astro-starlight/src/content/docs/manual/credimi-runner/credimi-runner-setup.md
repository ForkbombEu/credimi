---
title: Set up your own runner
description: Configure a local runner to execute pipelines on your devices.
---

This guide explains how to install and configure a Credimi runner.

---

## Installation

Run:

```bash
curl -sL credimi.run | sh
```

This starts an interactive setup process.

---

:::tip
API KEYS
You'll need an API Key to setup a runner, see: [Configure API Keys](./api-keys.md)
:::

---

## Interactive setup

The setup will:

- ask for configuration parameters  
- configure your device type  
- store local configuration  
- register the runner in Credimi  

At the end of the process, a local service is installed:

```bash
credimi-runner-service
```

:::tip
See the detailed [Step-by-step Walkthrough](./credimi-runner-setup-explained/) and 
in-depth [Devices and Emulators config](https://github.com/ForkbombEu/credimi-runner/blob/main/README.md)
:::



---

## Starting the runner

The runner is not started automatically.

You must run:

```bash
credimi-runner-service quick
```

Other modes are available:

- `quick` → temporary public endpoint (Cloudflare tunnel)  
- `direct` → use a fixed public IP  
- `named` → use a configured domain/tunnel  
- `down` → stop the runner  

---

## Runner registration

During setup:

- the runner is registered in Credimi  
- it appears in the **Runners** section  
- it can be used immediately in pipelines  

---

## Supported device types

You can configure the runner to use different environments.

---

### Android device

- real physical device  
- connected via USB or network  
- requires developer mode and ADB debugging  

Recommended settings:

- disable auto-lock  
- disable screen timeout  
- keep device unlocked  
- enable debugging  

---

### Android emulator (x86)

- runs locally on your machine  
- automatically managed by the runner  

Note:

- some Wallet apps do not run on x86 emulators  
- due to ARM64-only native libraries  

---

### Redroid

- Android running in a container  
- data stored on disk and accessible during execution  

Advantages:

- full control over `/data`  
- useful for advanced debugging  

Trade-offs:

- more complex setup  
- harder to customize images  

---

### iOS simulator

- runs on macOS  
- controlled by the runner  
- integrated with pipeline execution  

---

## Device lifecycle

Device management is fully automatic.

When a pipeline runs:

- the environment is prepared  
- the device is started (if needed)  
- the app is installed and executed  
- logs and outputs are collected  

For emulator-based setups:

- a clean environment is used for each run  
- the instance is reset after execution  

---

## Running your first pipeline

Once the runner is active:

1. open Credimi  
2. select a pipeline  
3. choose your runner  
4. press **Run**  

Execution will start automatically.

---

## Troubleshooting

Common issues:

- device not properly configured (ADB, permissions)  
- OS updates requiring manual unlock  
- browser updates requiring environment refresh  

Once configured correctly, runners are stable over long periods.

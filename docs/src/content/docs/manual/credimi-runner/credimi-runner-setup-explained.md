---
title: Runner setup – interactive questions explained
description: Understand the questions asked during credimi-runner installation.
---

When you run:

```bash
curl -sL credimi.run | sh
```

the installer will ask a series of questions to configure your runner.

This page explains what each question means and how to answer it.

---

## Credimi connection

You will be asked to provide:

- **Credimi URL**
- **API Key**

The API Key is created in:

[Configure API Keys](./api-keys/)

The runner uses this key to:

- register itself in Credimi  
- authenticate all communication  
- receive pipeline execution requests  

---

## Runner identity

The setup will ask for:

- **Runner name**
- **Organization**

These are used to:

- generate a unique runner identifier  
- register the runner in Credimi  
- display it in the Runners list  

If a runner with the same identity already exists, you may be asked whether to:

- update the existing runner  
- or create a new one  

---

## Runner type

You will be asked to select a runner type.

Available options depend on your system:

- Android device  
- Android emulator  
- Redroid  
- iOS simulator (macOS only)  

This determines which environment will be used to execute pipelines.

---

## Android device configuration

If you select an Android device, you will be asked how to connect to it:

- **USB**
- **Wi-Fi**

### USB

- the runner will try to auto-detect a connected device  
- requires USB debugging enabled  

### Wi-Fi

You will need to provide:

- device IP address  
- port (default: `5555`)  

---

## Redroid configuration

If you select Redroid, additional parameters are required:

- **Redroid data directory**
- **Redroid data archive path**

Defaults are typically:

```text
/home/credimi/redroid-data
/home/credimi/redroid-data.tar
```

---

## avdctl / SSH configuration (Redroid / advanced setups)

In some configurations, the runner needs to control the device host via SSH.

You may be asked for:

- SSH target (e.g. Raspberry Pi running Redroid)  
- SSH password  
- whether `sudo` is required  
- sudo password (if applicable)  

This is used to:

- create and destroy device environments  
- manage lifecycle of emulator/Redroid instances  

> NOTE:
> This SSH configuration is stored locally on the runner machine and is not shared with Credimi.

---

## Exposure mode (after setup)

Once setup is complete, the runner is started manually using:

```bash
credimi-runner-service <mode>
```

Available modes:

### quick

- uses a temporary Cloudflare tunnel  
- automatically generates a public URL  
- registers it in Credimi  

Best for:

- quick setup  
- testing  
- local environments  

---

### direct

- uses a fixed public IP  

Requires:

- public IP address  
- reachable port  

Best for:

- stable environments without tunnels  

---

### named

- uses a preconfigured Cloudflare tunnel and domain  

Requires:

- Cloudflare tunnel token  
- domain name  

Best for:

- long-lived, stable deployments  

---

### down

Stops the runner and removes the local environment.

---

## What happens after setup

At the end of the setup:

- configuration is saved locally  
- the runner is registered in Credimi  
- it appears in the **Runners** list  

You can now start it and use it in pipelines.

---

## Summary

The setup process defines:

- how the runner connects to Credimi  
- which device it controls  
- how it is exposed to the platform  

Once configured, the runner operates automatically and requires no further manual interaction during pipeline execution.

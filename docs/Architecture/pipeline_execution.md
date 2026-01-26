# Pipeline Execution Architecture

This document explains how Credimi executes mobile compliance pipelines, from API request to emulator cleanup.

## System Overview

```mermaid
flowchart LR
  User[Customer] --> API[Credimi API]
  API --> Temporal[Temporal Server]
  Temporal --> Pool[AVD Pool Manager Workflow]
  Pool --> Worker[Pipeline Worker]
  Worker --> AVDCTL[avdctl]
  AVDCTL --> Emulator[Android Emulator]
  Worker --> Maestro[Maestro Runner]
  Emulator --> Maestro
  Maestro --> Storage[PocketBase + Object Storage]
  Worker --> Storage
```

## Pipeline Sequence

```mermaid
sequenceDiagram
  autonumber
  participant API as Credimi API
  participant Temporal as Temporal
  participant Pool as Pool Manager
  participant Worker as Pipeline Worker
  participant AVD as AVD/avdctl
  participant Maestro as Maestro
  participant PB as PocketBase

  API->>Temporal: Start pipeline workflow
  Worker->>Pool: Acquire slot
  Pool-->>Worker: Slot granted
  Worker->>AVD: Clone + Run emulator
  AVD-->>Worker: Emulator serial
  Worker->>Maestro: Install APK + Run flow
  Maestro-->>Worker: Flow output
  Worker->>Maestro: Stop recording
  Worker->>PB: Store results + artifacts
  Worker->>AVD: Stop emulator + delete clone
  Worker->>Pool: Release slot
  Worker-->>Temporal: Workflow completed
```

## State Machines

### AVD Lifecycle

```mermaid
stateDiagram-v2
  [*] --> Cloned
  Cloned --> Booting
  Booting --> Ready
  Ready --> Recording
  Recording --> Stopping
  Stopping --> Deleted
  Deleted --> [*]
```

### Pool Manager

- **Available**: Slots are free.
- **Queued**: Requests wait for a slot.
- **Active**: Leases are assigned to running workflows.
- **Expired**: Lease heartbeat expired → slot reclaimed.

### Recording

- **Starting** → **Recording** → **Stopping** → **Uploaded**
- On failure, a **DLQ entry** is recorded in `failed_cleanups` for reconciliation.

## Key Guarantees

- Cleanup runs in a saga with retries.
- Failed cleanup steps are recorded and reconciled.
- Search attributes capture status (`queued`, `booting`, `recording`, `cleanup`, `failed`, `completed`).
- Operators can query workflow state and pool status via CLI.

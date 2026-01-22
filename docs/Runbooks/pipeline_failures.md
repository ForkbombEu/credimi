# Pipeline Failure Runbooks

This runbook covers common pipeline failure symptoms and recovery steps.

## Pipeline stuck in queued

**Symptoms**
- Workflow search attribute `status = queued` for >10 minutes.
- Pool queue depth grows.

**Checks**
1. `credimi pool status --namespace <ns>`
2. Verify `available` slots and `queued` count.

**Actions**
- If capacity is too low, increase `AVD_POOL_MAX_CONCURRENT` and restart workers.
- If pool workflow is unhealthy, run `credimi pool reset --namespace <ns>`.

## Emulator fails to boot

**Symptoms**
- `boot_status` never reaches `ready`.
- Pipeline errors with emulator boot timeout.

**Checks**
1. Ensure `/dev/kvm` is available in the worker container.
2. Inspect emulator logs: `/tmp/emulator-<name>-<port>.log`.
3. Verify QEMU/AVD disk has enough space.

**Actions**
- Restart the worker with KVM enabled (`--privileged` + `/dev/kvm`).
- Recreate the golden image if boot regressions persist.

## Video corrupted or missing

**Symptoms**
- Pipeline results show missing/empty video.
- `upload_status.uploaded = false` with reason `ffprobe_failed` or `missing_or_empty`.

**Checks**
1. Ensure `/dev/shm` has free capacity.
2. Check ffmpeg/ffprobe availability in worker image.

**Actions**
- Increase `/dev/shm` size.
- Re-run the pipeline or trigger cleanup verification workflow.

## Cleanup failed

**Symptoms**
- `failed_cleanups` collection contains recent entries.
- Emulators remain in `avdctl ps`.

**Checks**
1. Query failed cleanup records in PocketBase.
2. Inspect reconciliation workflow logs.

**Actions**
- Run `credimi debug cleanup <workflow_id>` to force cleanup.
- If orphans remain, run `credimi cleanup orphans --force` on the worker host.


# Reference wallet emulator

## Runtime

- Serial: `emulator-5580`
- AVD: `credimi`
- Wallet app ID: `eu.europa.ec.euidi`
- Maestro: `/home/puria/.maestro/bin/maestro`
- Java: `/home/puria/.local/share/mise/installs/java/25.0.2`
- Viewer: `http://127.0.0.1:9999/`
- Wallet PIN used by repository flows: `123456`

Known launch shape:

```sh
QT_QPA_PLATFORM=xcb emulator @credimi -port 5580 -no-boot-anim -no-snapshot -no-snapshot-load -no-snapshot-save -skip-adb-auth -no-metrics -no-location-ui -no-audio
```

## State control

Clear Chrome before each independent pipeline run:

```sh
adb -s emulator-5580 shell pm clear com.android.chrome
```

Never clear wallet state when the test needs an issued PID. Use `clearState: false`.

Common starting states:

- `Welcome back`: run `unlock-wallet.yaml` or enter PIN.
- `Authenticate` bottom sheet: dismiss via its close surface or a verified safe top-area tap.
- Previous result screen: close it or relaunch the wallet without clearing state.
- Chrome on an issuer login page: clear Chrome, then relaunch the wallet.

Opening a deep link can trigger another `Welcome back` screen even after the wallet was unlocked. Handle it after `openLink`.

## PID

Reusable issuance flow:

`config_templates/fcaf/wallet_solution/relying_party/maestro-preconditions/obtain-pid-sdjwt.yaml`

Credential offer route:

`https://credimi.io/api/credential/deeplink?id=forkbomb-bv-andrea/misc-issuer-integration-demo/eudiw-pid-sd-jwt-vc-issuer-backend&redirect=true`

Use runtime issuer credentials. Never store them in the repository or this skill.

## UI inspection

Dump the hierarchy when selectors or state are unclear:

```sh
adb -s emulator-5580 shell uiautomator dump /sdcard/window.xml
adb -s emulator-5580 shell cat /sdcard/window.xml
```

Prefer stable resource IDs over text when available. For expandable credential cards, expand the request accordion before sharing and expand the result accordion again after sharing; they have independent state.

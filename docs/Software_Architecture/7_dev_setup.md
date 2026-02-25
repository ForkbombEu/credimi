# 🤖 Developer Setup

## **Prerequisites**

Before you begin, ensure you have the following tools installed:

-   [Git](https://git-scm.com/downloads)
-   [Make](https://www.gnu.org/software/make)
-   [Mise](https://mise.jdx.dev/getting-started.html)
-   [Temporal](https://docs.temporal.io/cli)
-   [Tmux](https://github.com/tmux/tmux/wiki/Installing)
-   [Pre-commit](https://pre-commit.com/)
-   [Golang](https://go.dev/doc/install)

### **Install `slangroom-exec`**

Download the appropriate `slangroom-exec` binary for your OS from the [releases page](https://github.com/dyne/slangroom-exec/releases).

Add `slangroom-exec` to PATH and make it executable:

```bash
wget https://github.com/dyne/slangroom-exec/releases/latest/download/slangroom-exec-Linux-x86_64 -O slangroom-exec
chmod +x slangroom-exec
sudo cp slangroom-exec /usr/local/bin/
```
### **Install `Pre-commit`**
For most Linux distrubution, just do: 
```bash
sudo apt install pre-commit
```


### **Install `slangroom-exec`**

## **Setup Workspace**

### **Clone the repository**

```bash
git clone https://github.com/ForkbombEu/credimi
```

### **Install dependencies**

```bash
cd credimi
mise trust
make credimi
```

## Edit your .env file 

Copy .env.example to .env

```bash
cp ./webapp/.env.example ./webapp/.env  
```

Then edit the .env file, particularly: 

1. set the absolute path in ROOT_DIR
1. Get a token from https://www.certification.openid.net and add it in TOKEN

Copy ./webapp/env.example to 

## **Start Development Server**

```bash
make dev
```

> [!TIP]
> Use `make help` to see all the commands available.

## Mobile Runner Semaphore Ops (Internal)

### Defaults and knobs

- Default acquire wait timeout: 45m.
- Override timeout: `MOBILE_RUNNER_SEMAPHORE_WAIT_TIMEOUT=30m` (or any valid `time.ParseDuration` value).
- Disable semaphore (no-op acquire/release): `MOBILE_RUNNER_SEMAPHORE_DISABLED=1`.
- Internal Temporal API auth key: `INTERNAL_ADMIN_API_KEY=<plaintext key>` (required for internal workflow HTTP calls).

### Internal admin API key rollout

1. Run DB migrations so `api_keys` supports `key_type`, `superuser`, `revoked`, `expires_at`.
2. Provision an internal admin key (hash stored in DB, plaintext returned once).
3. Set `INTERNAL_ADMIN_API_KEY` in backend and worker runtime secrets.
4. Deploy backend/workers with middleware + internal HTTP activity enabled.
5. Smoke-check one authenticated user route and one internal Temporal route.

### Emergency procedures

- Semaphore workflows live in the Temporal `default` namespace with IDs: `mobile-runner-semaphore/<runner_id>`.
- Query current state via Temporal UI (`GetState`) or `GET /api/mobile-runner/semaphore?runner_identifier=...`.
- To unstick a runner, terminate the semaphore workflow in Temporal; it will be recreated on the next acquire.

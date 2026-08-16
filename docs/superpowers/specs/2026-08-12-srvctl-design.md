# srvctl Design Spec

## 1. Purpose

`srvctl` is a single, self-contained Go binary that runs a **job** on remote servers. A job is a minimal YAML file that lists tasks (commands, scripts, or inline shell blocks) to execute, plus the servers and context for the run. It is intentionally not a full configuration management engine; it is the transport and ordering layer that makes shell-based configuration management practical across Linux, Unix, and Windows servers.

## 2. Goals & Non-Goals

Goals:
- One static binary, cross-compiled, with no runtime dependencies on the controller.
- Primary input is a YAML **job** file (`jobs.yaml` by convention).
- Support `bash`/`sh`/`zsh` on Linux/Unix and `powershell`/`pwsh` on Windows.
- Copy scripts or run commands on remote servers and report exit codes, stdout, stderr, and duration.
- Run tasks across multiple servers concurrently.
- Surface auth and host-key behavior clearly.

Non-goals:
- Idempotency or state model.
- Package/resource abstraction.
- Server-side agent or listener.
- Replacing full CM tools (Ansible, Puppet, etc.).

## 3. Recommended Approach

Approach B: **Job-driven execution**.

Two smaller alternatives considered:
- Approach A (thin SSH wrapper) is too limited for multi-server, repeatable work.
- Approach C (agent/listener) is out of scope because it requires software on every server and breaks the "just SSH" model.

## 4. Core Concepts

- **Job**: a YAML file with `servers`, `vars`, `env`, and a list of `tasks`.
- **Task**: a named step that runs a `command`, a `script` file, or an inline `shell` block.
- **Servers**: references to `Host` aliases in the user's `~/.ssh/config`.
- **Servers YAML** (`servers.yaml`): an optional file that lists `Host` aliases to use for a job.
- **Vars**: user-defined values available in task `command` and `shell` strings via `{{ .VarName }}`.
- **Env**: environment variables passed to the remote shell.

## 5. Job Format

```yaml
name: install nginx stack
servers: ./servers.yaml   # optional; can be inline list below

vars:
  version: "1.24"
  domain: example.com

env:
  DEBIAN_FRONTEND: noninteractive

tasks:
  - name: update packages
    command: apt-get update

  - name: install nginx
    script: ./scripts/install-nginx.sh
    env:
      VERSION: "{{ .version }}"

  - name: write config
    shell: |
      cat > /etc/nginx/nginx.conf <<'EOF'
      server {
          listen 80;
          server_name {{ .domain }};
      }
      EOF

  - name: restart nginx
    command: systemctl restart nginx
```

Job fields:
- `name` (required): human-readable label.
- `servers`: either a path to a `servers.yaml` file or an inline list of `Host` aliases.
- `defaults`: default values for tasks (e.g., `shell`).
- `vars`: variables for `{{ .VarName }}` templating.
- `env`: environment variables passed to every task unless overridden.
- `tasks`: list of tasks to run.

Task fields:
- `name` (required): human-readable label.
- `command`: a single command string.
- `script`: a local file path; uploaded and executed.
- `shell`: an inline multi-line script.
- `interpreter`: `bash` | `sh` | `zsh` | `powershell` | `pwsh` (overrides the job default).
- `env`: per-task environment variables.
- `timeout`: per-task timeout.
- `ignore_errors`: if true, a non-zero exit does not stop the run (unless `--fail-fast`).
- `servers`: a list of server aliases to limit this task to; if omitted, uses the job's servers.

Each task runs on all applicable servers before the next task begins.

## 6. Servers File

A `servers.yaml` file is just a list of `Host` aliases that already exist in the user's `~/.ssh/config`:

```yaml
servers:
  - web1
  - db1
  - win1
```

Or inline inside the job:

```yaml
servers: [web1, db1, win1]
```

No users, ports, keys, or hostnames are duplicated in `srvctl` files; all connection details come from `~/.ssh/config`.

## 7. SSH Config Resolution

`srvctl` resolves each server alias against the user's SSH config.

```ssh-config
Host web1
    HostName 10.0.0.5
    User admin
    Port 22
    IdentityFile ~/.ssh/id_ed25519

Host db1
    HostName 10.0.0.10
    User admin
    Port 22
    IdentityFile ~/.ssh/id_ed25519

Host win1
    HostName 10.0.0.6
    User Administrator
    IdentityFile ~/.ssh/win-key
```

Resolution rules:
- Default config file is `~/.ssh/config`.
- Override with `--ssh-config <path>` or the `SRVCTL_SSH_CONFIG` environment variable.
- `Host`, `HostName`, `User`, `Port`, `IdentityFile`, and `ProxyJump` are read from the matching block.
- Wildcard `Host` patterns are not expanded in v1; aliases must be explicit.
- Direct SSH only in v1; `ProxyJump` is a noted future enhancement.

## 8. Shell Model

Each task declares the interpreter to use via `interpreter`:

- `bash` | `sh` | `zsh` | `ksh`
- `powershell` (Windows PowerShell)
- `pwsh` (PowerShell Core)

A job-level `defaults.interpreter` can be set and overridden per task. Auto-detect by probing the server is possible but explicit is the default.

Execution template examples:

- Linux bash:
  - Upload to `/tmp/srvctl-<hash>-<name>.sh`
  - `chmod +x <path> && bash <path>`
- Windows PowerShell:
  - Upload to `C:\Windows\Temp\srvctl-<hash>-<name>.ps1`
  - `powershell -ExecutionPolicy Bypass -File <path>`
- pwsh:
  - `pwsh -File <path>`

A shebang line may be used as a fallback on Unix only.

## 9. Transport & Delivery

- SSH for command execution.
- SFTP for file upload. Fallback to `cat` over SSH if SFTP is unavailable.
- Scripts are written to a temporary path and cleaned up after the run unless `--keep` is set.
- Very small tasks may optionally run inline via `stdin` (`--inline`).

## 10. CLI

```
srvctl run jobs.yaml
srvctl run jobs.yaml -s servers.yaml
srvctl run jobs.yaml -S web1,db1,win1
srvctl validate jobs.yaml
```

Common flags:

- `-s, --servers <file>`: servers file path.
- `-S, --server-list <csv>`: inline server aliases.
- `--ssh-config <file>`: override `~/.ssh/config`.
- `-e, --env KEY=VAL`: environment variables.
- `--var KEY=VAL`: job variables.
- `--interpreter <shell>`: default interpreter.
- `--workers <n>`: concurrency (default 10).
- `--timeout <duration>`: per-task timeout.
- `--output <text|json>`: result format.
- `--log-dir <path>`: write per-server stdout/stderr to files.
- `--keep`: do not remove uploaded scripts.
- `--insecure-ignore-hostkey`: skip host key verification (testing only).
- `--fail-fast`: stop all tasks on first failure.

## 11. Concurrency & Results

- Worker pool of SSH connections; `--workers` controls it.
- Default output: colorized table with task, server, status, exit code, duration.
- JSON output for CI.
- Logs: capture per-server stdout/stderr; optional `--log-dir`.
- `srvctl` exit code: highest remote exit code, or `1` if any connection fails.
- `--fail-fast` aborts all pending/running tasks on first non-zero exit.

## 12. Error Handling

- Connection errors are reported per server and do not stop other servers unless `--fail-fast`.
- Non-zero remote exit codes are reported as failures.
- Timeouts per task.

## 13. Security Notes

- Secrets are passed as environment variables over the SSH channel, never embedded in the uploaded script.
- Temp files are removed by default.
- Host key verification is strict by default.

## 14. Dependencies

Go standard library plus:

- `golang.org/x/crypto/ssh`
- `golang.org/x/crypto/ssh/knownhosts`
- `gopkg.in/yaml.v3`
- SSH config parser (e.g., `github.com/kevinburke/ssh_config` or equivalent)

Compiled with `CGO_ENABLED=0` for a portable static binary.

## 15. Assumptions

- Binary name is `srvctl`.
- Servers have an SSH server running. Windows servers use OpenSSH for PowerShell/pwsh.
- Jobs and servers files use YAML.
- The user maintains `~/.ssh/config` with connection details.

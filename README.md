# ops

Run shell jobs on remote servers via SSH.

## Install

```bash
make install
```

## Usage

```bash
ops validate [jobs.yaml]             # defaults to jobs.yaml
ops run jobs.yaml -s servers.yaml
ops run jobs.yaml -S web1,db1
ops run jobs.yaml -q                 # summary only, no command output
ops run jobs.yaml --stream         # stream output live while running
ops run jobs.yaml --local          # run tasks on the current machine
ops run jobs.yaml --dry-run       # show what would run without executing
ops plan jobs.yaml                  # alias for run --dry-run
ops update                         # update to latest release
ops update --check                 # check for update without installing
ops update --force                # force reinstall even if already on latest
ops version                        # print version
ops new [directory]              # create sample jobs.yaml and servers.yaml
ops list|ls [-r] [directory]       # list job files (recursively with -r)
```

## Example job

```yaml
---
description: install and restart nginx
servers: ./servers.yaml
defaults:
  interpreter: bash
vars:
  version: "1.24"
tasks:
  - name: install
    script: ./scripts/install.sh
    env:
      VERSION: "{{ .version }}"
  - name: restart
    command: systemctl restart nginx
```

## Task types

Each task must use exactly one of `command`, `shell`, `script`, or `upload`. Commands, shell blocks, and `env` values are rendered with Go `text/template`; script file contents are not.

- `command`: a single command string. Run via the interpreter's `-c` flag.
- `shell`: an inline multi-line script. Uploaded to the remote host and executed.
- `script`: a local file path. The file is read as-is, uploaded to the remote host, and executed by the task's interpreter. The path is resolved relative to the directory where `ops run` is invoked.
- `upload`: a local file path. The file is uploaded to the remote `dest` path and not executed. The destination must be a full remote file path.

Use `workdir` in `defaults` or on a task to set the working directory for `command`, `shell`, and `script` tasks.

Uploaded scripts and shell blocks are written to `/tmp` on Linux/Unix and `C:\Windows\Temp` on Windows, and removed after execution unless you pass `--keep`.


## Output

By default, `ops run` shows a summary table followed by the stdout/stderr
from each task on each server. Use `-q` (`--quiet`) to show only the summary
table. Use `-o json` for machine-readable output that includes all fields. Use
`--stream` to print output live as each task runs. A progress bar is drawn to
stderr while tasks run (disabled with `--quiet`, `--stream`, or `--output json`).

## Example servers

```yaml
servers:
  - web1
  - db1
```

Ensure `web1` and `db1` exist as `Host` entries in `~/.ssh/config`.

## Integration test

Set up an SSH server accessible via `~/.ssh/config` and run:

```bash
OPS_TEST_HOST=localhost go test ./tests/...
```
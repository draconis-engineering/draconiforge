# draconiforge

> Streamline exhausting dev tasks. One binary, safe by default.

Binary is `draconiforge` (alias `forge` — avoids collision with system `df`).

## Install

**One-liner (website):**

```bash
# Linux / macOS — sh
curl -fsSL https://raw.githubusercontent.com/draconis-engineering/draconiforge/main/scripts/install.sh | sh
# or
wget -qO- https://raw.githubusercontent.com/draconis-engineering/draconiforge/main/scripts/install.sh | sh

# Windows — PowerShell
irm https://raw.githubusercontent.com/draconis-engineering/draconiforge/main/scripts/install.ps1 | iex
```

<details><summary>Manual</summary>

```bash
git clone https://github.com/draconis-engineering/draconiforge
cd draconiforge
make install   # -> ~/.local/bin/draconiforge + ~/.local/share/draconiforge/scripts
export PATH="$HOME/.local/bin:$PATH"
```
</details>

Requires Go 1.24+, `git`, `bash` (for `run`), `pwsh` for `.ps1`.

## Quick start

```bash
forge doctor              # check git config, remotes, .env, PATH
forge prework             # fetch + pull --ff-only (aborts if dirty/diverged)
# ... work ...
forge postwork -a -m "feat: thing"   # add all, commit, push
# or lazy:
forge sync -a -m "wip"    # commit + pull + push in one

forge new myapp -L go -l mit          # scaffold go/python/shell/julia/node
forge run --list          # list sh/ps1 scripts
forge run git_tracker     # run a script

# draconiforge also works everywhere forge does
```

## Commands

| Command | What it does |
|---|---|
| `prework [--dry-run] [--rebase]` | Fetch, abort if dirty/diverged, fast-forward pull |
| `postwork -m "msg" [-a] [--dry-run] [--no-push]` | Stage/commit/push, abort if dirty without -m |
| `sync -m "msg" [-a] [--rebase] [--dry-run]` | `prework + postwork` combined |
| `new <name> -L <lang> -l <license>` | Scaffold template. Langs: `go,python,shell,julia,node`. Licenses: `gplv3,mit,apache-2.0,bsd-3,none` |
| `doctor [--verbose] [--check-ssh]` | Read-only health checks with `→ fix:` hints |
| `run [script] [--dry-run] [--verbose]` | Run `*.sh`/`*.ps1` from `./scripts`, `~/.config/draconiforge/scripts`, or installed share |

See `docs/ROADMAP.md` for what's next.

## Dev

```bash
make build     # -> ./draconiforge
make vet       # go vet
make clean
```

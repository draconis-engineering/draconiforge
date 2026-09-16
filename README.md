# draconiforge

> Streamline exhausting dev tasks. One binary, safe by default.

Binary is `draconiforge` (alias `df` — add `~/.local/bin` first in PATH).

## Install

```bash
git clone https://github.com/draconis-engineering/draconiforge
cd draconiforge
make install   # -> ~/.local/bin/draconiforge + ~/.local/share/draconiforge/scripts
# fix df alias collision with /usr/bin/df:
export PATH="$HOME/.local/bin:$PATH"
```

Requires Go 1.24+, `git`, `bash` (for `run`), `pwsh` for `.ps1`.

## Quick start

```bash
draconiforge doctor              # check git config, remotes, .env, PATH
draconiforge prework             # fetch + pull --ff-only (aborts if dirty/diverged)
# ... work ...
draconiforge postwork -a -m "feat: thing"   # add all, commit, push
# or lazy:
draconiforge sync -a -m "wip"    # commit + pull + push in one

draconiforge new myapp -L go -l mit          # scaffold go/python/shell/julia/node
draconiforge run --list          # list sh/ps1 scripts
draconiforge run git_tracker     # run a script
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

See `ROADMAP.md` for what's next.

## Dev

```bash
make build     # -> ./draconiforge
make vet       # go vet
make clean
```

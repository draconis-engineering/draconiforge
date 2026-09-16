# Draconiforge Roadmap

> CLI to streamline exhausting dev tasks. Binary: `draconiforge` (alias `df`, requires `~/.local/bin` first in PATH).

## Vision
One binary, no config, safe by default. Every command aborts rather than destroys. `prework`/`postwork` bracket your day, `new` starts it.

---

## v0.2 — Done ✓

- `draconiforge` / `df` — Cobra, global install via `make install`
- `prework` — fetch, abort if dirty/diverged, fast-forward pull (`internal/cli/prework.go`, `internal/cli/git.go`)
- `postwork` — abort if dirty without `-m`, `git add -A`/`commit`/`push`, diverge check (`internal/cli/postwork.go`)
- `new [name] -L <lang> -l <license>` — scaffold via `embed.FS` (`internal/scaffold/`, `internal/license/`)
  - Languages: `go`, `python`, `shell`, `julia`, `node`
  - Licenses: `gplv3` (default), `mit`, `apache-2.0`, `bsd-3`, `none`

## v0.3 — Done ✓ — Script runner (hybrid)

Go front, sh/ps1 back — future commands stay as scripts, run through Go.

- **Dispatcher:** `internal/runner/runner.go` + `internal/cli/run.go`
- **Discovery:** `./scripts` (project), `~/.config/draconiforge/scripts` (user), `~/.local/share/draconiforge/scripts` (installed via `make install` copies `scripts/*.sh|*.ps1`).
- **Exec:** `.sh/.bash` → `bash`, `.ps1` → `pwsh -File` (fallback `powershell` on Windows).

## v0.4 — Done ✓ — Doctor

- `doctor` — read-only health checks (`internal/cli/doctor.go`): git installed, inside repo, `user.name`/`user.email`, `pull.ff`/`pull.rebase`, `init.defaultBranch`, remotes + upstream + `fetch --dry-run`, sync status (ahead/behind/diverged), `.env` tracked, large files (>10MB), `.gitignore`, working tree dirty, detached HEAD, binary + `~/.local/bin` in PATH. Flags `--verbose`, `--check-ssh`. Shows `✓/⚠/✖` with `→ fix:` hints.

## v0.5 — In progress

- **`sync` — Done ✓** `draconiforge sync -m "msg" -a [--dry-run] [--rebase] [--no-push] [--force]` (`internal/cli/sync.go`): commit staged (like postwork) → fetch → pull --ff-only → push. Aborts if dirty without -m, aborts if diverged. Ideal for `draconiforge sync -a -m "wip"`.

- **`release`** — automate version bump + tag + push. Reads `internal/cli/version.go` / `Makefile` ldflags, updates version, `CHANGELOG.md`, `git tag`, `git push --follow-tags`. Flags `--major/--minor/--patch`, `--dry-run`.

- **`standup`** — copy-paste daily summary. `git log --since=yesterday --author=$(git config user.name)` + `git status --short` across repos (reuse `runner` discovery or `git_tracker.sh` logic). Output Markdown for Slack.

- Also: `branch-clean` — `git fetch --prune` + list merged/gone branches, prompt delete.

## v0.6 — New templates v2

- Flags for `new`: `--author`, `--module-path` (Go), `--python-version` — add config file `~/.config/draconiforge/config.yaml` for defaults.
- Templates: add `rust`, `zig`; `--with-ci` GitHub Actions.
- `new --from <template-repo>` — clone custom template.

## v0.7 — Safety upgrades (future, opt-in)

- `prework --stash` / autobranching — keep v1 abort, add flag: stash dirty, pull, pop; or `git switch -c wip/<date>`.
- `postwork --amend`, `--force-with-lease` confirmation.
- Undo log for `tidy`/`branch-clean`.

## Backlog

- `df config init` — interactive setup for defaults.
- `df update` — self-update via `go install` or GitHub releases.
- Docs: `README.md` is empty — fill with install + 3 commands demo.
- Shell completion: `draconiforge completion zsh|bash|fish` already from Cobra, add `make install-completion`.

---

## Contributing

1. `make build && ./draconiforge --help`
2. Add command in `internal/cli/*.go`, register in `internal/cli/root.go`; add scripts in `scripts/` or `~/.config/draconiforge/scripts/`
3. `go vet ./...`, `make install`, test with `draconiforge <cmd> --dry-run` / `draconiforge run <script> --dry-run`
4. Keep abort-first behavior for any git-mutating command.

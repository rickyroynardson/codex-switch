# codex-switch

Keep multiple Codex auth profiles on one machine and switch between them by tag —
while your session history, prompts, and shell state stay shared across all of them.

Each account gets its own login (`auth.json`) and config; your `sessions/` and
`history.jsonl` live in one shared store that every account symlinks to. Switch
from `work` to `personal` and the conversation history follows you.

## Install

```sh
go install github.com/rickyroynardson/codex-switch@latest
codex-switch init
export PATH="$HOME/.codex-switch/bin:$PATH"   # add to your shell profile
```

`init` installs a `codex` wrapper on your PATH and imports your existing
`~/.codex` session history into the shared store. Put `~/.codex-switch/bin`
**before** the real `codex` on your PATH so the wrapper takes over.

## Usage

```sh
codex-switch login work        # log in and tag this account "work"
codex-switch login personal    # log in another account

codex-switch switch personal   # make "personal" the active account
codex-switch current           # print the active tag

codex                          # runs Codex as the active account
codex-switch status            # show all accounts (cached, instant)
codex-switch status --refresh  # probe Codex for live auth + quota
codex-switch remove work       # delete an account (not the active one)
```

Once installed, you use plain `codex` as normal — it transparently runs as
whichever account is active. `codex-switch` is only for managing profiles.

### status

`status` is cache-first: by default it prints the last known auth state and
quota for each account instantly, with a `CHECKED` column showing how stale each
row is. Pass `--refresh` to probe Codex live and update the cache. Use `--tag
<tag>` for a single account's detail.

## Commands

| Command | What it does |
|---|---|
| `init` | Install the `codex` wrapper and import existing session history |
| `login <tag>` | Log in to a Codex account and tag it |
| `switch <tag>` | Set the active account |
| `current` | Print the active tag |
| `status [--tag <tag>] [--refresh]` | Show account auth and quota (cached; `--refresh` for live) |
| `remove <tag>` | Delete an account (refuses the active one) |

## How it works

`codex-switch init` drops a small `codex` shim in `~/.codex-switch/bin`. When you
run `codex`, the shim calls `codex-switch proxy`, which runs the real Codex with
`CODEX_HOME` pointed at the active account's directory:

```
~/.codex-switch/
├── accounts/
│   ├── work/          <- a full CODEX_HOME (auth.json, config.toml, per-account)
│   │   ├── auth.json          real, per-account
│   │   ├── config.toml        real, per-account
│   │   ├── sessions        -> ../../shared/sessions
│   │   └── history.jsonl   -> ../../shared/history.jsonl
│   └── personal/      <- same layout, its own auth
├── shared/            <- sessions, archived_sessions, history, session_index
├── state/accounts.json
└── bin/codex          <- the wrapper shim
```

Auth and config are real per-account files, so each account keeps its own login
and Codex writes refreshed tokens straight back to it. The session entries are
symlinks into `shared/`, so history is common to every account.

Override the root directory with `CODEX_SWITCH_HOME` if you don't want
`~/.codex-switch`.

## License

MIT

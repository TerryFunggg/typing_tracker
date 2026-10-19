# typing-tracker

A simple, private, local coding-time tracker. One Go binary, like WakaTime. Supports Vim and Emacs.

Nothing is uploaded. File contents and keystrokes are never sent.

Reading status on terminal.

## OS-support

- Linux
- macOS: not supported now
- Windows: not supported now

## What is stored

Each heartbeat records:

- time
- absolute file path
- git project and branch, when the file is inside a repository
- language (`go`, `markdown`, the Emacs mode, the Vim filetype)
- editor (`vim` or `emacs`)
- whether the event was a save

The database is `~/.local/share/typing-tracker/tracker.db`. Delete that file to erase the history.

`~/.config/typing-tracker/config.toml` holds the port, the session gap, and directories to ignore. `~/.config/typing-tracker/token` is a bearer token created by `tt serve`. Both files are mode `0600`. The server listens on `127.0.0.1` only.

## Run

```bash
go build -o tt ./cmd/tt
install -m 755 tt ~/.local/bin/tt
tt serve
```

Leave `tt serve` running while you edit. If it is not running, Vim and Emacs keep working and show one message the first time a heartbeat fails.

In another terminal:

```bash
tt today
tt week
tt status
```

`tt today` is today's total, then time by project, language, and editor. `tt week` is the last 7 local days. Coding time is the sum of gaps between heartbeats that are 2 minutes or less apart. A longer gap ends the session.

Typing sends at most one heartbeat every 30 seconds per file. Opening a file sends one. Saving always sends one.

## Config

`~/.config/typing-tracker/config.toml` is read when the server starts. After an edit, restart with `systemctl --user restart typing-tracker.service`.

| Setting | Default | What it does |
|---|---|---|
| `port` | `57147` | TCP port for heartbeats. Must be from 1 to 65535. The editor plugins read this same value. |
| `bind` | `"127.0.0.1"` | Address the server listens on. Only a loopback address is accepted, such as `127.0.0.1` or `::1`. |
| `gap_seconds` | `120` | A gap longer than this many seconds ends a coding session. Must be at least 1. |
| `ignore` | `[]` | Absolute directory prefixes to drop. Heartbeats for files under these paths are not stored. |

`ignore` is one line, as a JSON list of absolute paths:

```toml
ignore = ["/home/terryfung/notes", "/home/terryfung/private"]
```

A path matches that directory and everything inside it.

## Editors

Under ~editor~ folder support Vim and Emacs.

Vim, in `~/.vimrc`:

```vim
source typing_tracker.vim
```

Emacs, in `~/.emacs.d/init.el`:

```elisp
(load "typing-tracker.el" nil 'nomessage)
```

Remove those lines to stop tracking. Restart Vim or Emacs after you change them.

The plugins read the port and the token from `~/.config/typing-tracker/`. The service creates those files the first time it starts.

## Auto Start when login

The user service is installed at `~/.config/systemd/user/typing-tracker.service`. It runs `tt serve` as you, because the database and token live in your home directory.

```ini
[Unit]
Description=Local coding time tracker

[Service]
ExecStart=%h/.local/bin/tt serve
Restart=on-failure

[Install]
WantedBy=default.target
```

```bash
systemctl --user enable --now typing-tracker.service
```

`enable --now` starts it for this login. After changing the config file:

```bash
systemctl --user restart typing-tracker.service
```

## Limits

Time is counted only while `tt serve` is running and an editor sends heartbeats. A stretch with the server stopped is absent from the report. A single heartbeat, with no second one inside 2 minutes, adds no duration. Project and branch are whatever `git` reports for that file when the heartbeat arrives.

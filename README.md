# investments-healthcheck

A small Mac CLI that scrapes your investment funds from [analizy.pl](https://www.analizy.pl) once a day, computes profit/loss and trend signals from accumulated price history, and posts a daily verdict (very good / good / stable / warning / alert) to a private Telegram channel.

Single static Go binary. No runtime dependencies. Schedules itself via `launchd` and runs in the background.

## Install

### Homebrew (recommended)

```bash
brew install lizarusi/tap/healthcheck
healthcheck init        # interactive setup (Telegram + funds + launchd schedule)
```

`brew upgrade healthcheck` later picks up new releases automatically.

### From source

Requires Go 1.26+.

```bash
git clone https://github.com/lizarusi/investments-healthcheck
cd investments-healthcheck
make install            # builds and copies binary to /usr/local/bin/
healthcheck init
```

`make install` may require `sudo` depending on `/usr/local/bin` permissions.

## What `init` asks for

1. **Telegram bot token.** Talk to `@BotFather` in Telegram → `/newbot` → save the token.
2. **Telegram channel ID.** Create a private channel, add your bot as admin, send a message there, then visit `https://api.telegram.org/bot<TOKEN>/getUpdates` to find the `chat.id` (a negative number starting with `-100…`).
3. **Funds.** For each fund: paste the analizy.pl URL (e.g. `https://www.analizy.pl/fundusze-zagraniczne/FIL133_A_USD/...`), purchase date (`YYYY-MM-DD`), units purchased, purchase NAV price.

After init, the wizard installs a `launchd` agent that runs daily at 18:00 (configurable in `config.yaml`).

## Files

| Path | Purpose |
|---|---|
| `~/.config/investments-healthcheck/config.yaml` | Editable config: Telegram secrets, fund list, alert thresholds, schedule time |
| `~/Library/Application Support/investments-healthcheck/data.db` | SQLite price history |
| `~/Library/Logs/investments-healthcheck/run.log` | launchd stdout/stderr |
| `~/Library/LaunchAgents/com.user.investments-healthcheck.plist` | The schedule itself |

## Commands

| Command | Purpose |
|---|---|
| `healthcheck init` | First-time setup; idempotent (re-running edits config). |
| `healthcheck run` | Manually run the daily job: scrape, store, send Telegram. |
| `healthcheck run --dry-run` | Same as `run` but prints the message instead of sending it. Still writes today's price to the DB. |
| `healthcheck show` | Print today's report to stdout. Does NOT write to the DB and does NOT send Telegram. |
| `healthcheck uninstall` | Remove the launchd schedule. Config and DB are preserved. |

## Verdict rules

Worst level wins.

| Level | Triggered by |
|---|---|
| 🔴 ALERT | Any fund 1d drop ≥ 3% **or** any fund 5d drop ≥ 7% **or** portfolio 5d drop ≥ 5% |
| 🟡 WARNING | Any fund 5d drop ≥ 3% (and not alert) |
| ⚪ STABLE | All moves within ±1% over 5 days |
| 🟢 GOOD | Portfolio 5d gain ≥ 1% |
| 🟢🟢 VERY GOOD | Portfolio 5d gain ≥ 5% |

Thresholds live in `config.yaml` under `thresholds:` — edit and save, no restart needed.

## Cold start

The tool does not (yet) backfill historical prices. Each daily run appends one NAV per fund. Until ~5 trading days of history accumulate, the analyzer skips the 5d-trend rules and reports profit/loss against your purchase price only.

## Environment overrides

Useful for testing or unusual setups:

| Variable | Overrides |
|---|---|
| `HEALTHCHECK_CONFIG` | path to `config.yaml` |
| `HEALTHCHECK_DB` | path to `data.db` |
| `HEALTHCHECK_LOG` | path to `run.log` |

## Updating

If you installed via Homebrew:

```bash
brew upgrade healthcheck
```

The launchd schedule keeps pointing at the binary path, so a brew upgrade is enough — no need to re-run `init`.

## Uninstall

Always run `healthcheck uninstall` *before* removing the binary — it cleans up the launchd schedule.

If you installed via Homebrew:

```bash
healthcheck uninstall                    # remove launchd schedule
brew uninstall --cask healthcheck        # remove binary
```

If you installed from source:

```bash
make uninstall                           # both steps in one
```

Config and DB are preserved by design. Delete them yourself for a clean wipe:

```bash
rm ~/.config/investments-healthcheck/config.yaml
rm ~/Library/Application\ Support/investments-healthcheck/data.db
```

## Development

```bash
make build              # bin/healthcheck
make test               # go test ./...
make show               # build + healthcheck show
make dryrun             # build + healthcheck run --dry-run
```

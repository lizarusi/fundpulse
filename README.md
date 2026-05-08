# fundpulse

A small Mac CLI that scrapes your investment funds from [analizy.pl](https://www.analizy.pl) once a day, computes profit/loss and trend signals from accumulated price history, and posts a daily verdict (very good / good / stable / warning / alert) to a private Telegram channel.

Single static Go binary. No runtime dependencies. Schedules itself via `launchd` and runs in the background.

## Install

### Homebrew (recommended)

```bash
brew install lizarusi/tap/fundpulse
fundpulse init        # interactive setup (Telegram + funds + launchd schedule)
```

`brew upgrade fundpulse` later picks up new releases automatically.

### From source

Requires Go 1.26+.

```bash
git clone https://github.com/lizarusi/fundpulse
cd fundpulse
make install            # builds and copies binary to /usr/local/bin/
fundpulse init
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
| `~/.config/fundpulse/config.yaml` | Editable config: Telegram secrets, fund list, alert thresholds, schedule time |
| `~/Library/Application Support/fundpulse/data.db` | SQLite price history |
| `~/Library/Logs/fundpulse/run.log` | launchd stdout/stderr |
| `~/Library/LaunchAgents/com.lizarusi.fundpulse.plist` | The schedule itself |

## Commands

| Command | Purpose |
|---|---|
| `fundpulse init` | First-time setup; idempotent (re-running edits config and reinstalls launchd). |
| `fundpulse config` | Edit existing config interactively — Telegram, schedule, base currency, thresholds, and funds. Press Enter at any prompt to keep the current value. Reinstalls launchd only if the schedule changed; fetches a snapshot for any newly added fund. |
| `fundpulse config show` | Print the current config (bot token masked) and the funds table. Read-only. |
| `fundpulse backfill` | Re-fetch all historical prices for configured funds from analizy.pl's chart endpoint. Idempotent (upserts by date). |
| `fundpulse run` | Manually run the daily job: scrape, store, send Telegram. |
| `fundpulse run --dry-run` | Same as `run` but prints the message instead of sending it. Still writes today's price to the DB. |
| `fundpulse show` | Print today's report to stdout. Does NOT write to the DB and does NOT send Telegram. |
| `fundpulse uninstall` | Remove the launchd schedule. Config and DB are preserved. |

For ad-hoc fund management, `fundpulse config` is the canonical tool. After the top-level prompts, you'll see each existing fund and choose **(k)eep / (e)dit / (d)elete**, then a final "Add a fund?" loop.

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

`fundpulse init` automatically backfills historical NAVs since each fund's purchase date by calling `analizy.pl`'s chart endpoint. If that fails (rate-limited, schema change), the daily run keeps appending one price per day; after ~5 trading days the 5d-trend rules become available. You can re-run `fundpulse backfill` at any time.

## Environment overrides

Useful for testing or unusual setups:

| Variable | Overrides |
|---|---|
| `FUNDPULSE_CONFIG` | path to `config.yaml` |
| `FUNDPULSE_DB` | path to `data.db` |
| `FUNDPULSE_LOG` | path to `run.log` |

## Updating

If you installed via Homebrew:

```bash
brew upgrade fundpulse
```

The launchd schedule keeps pointing at the binary path, so a brew upgrade is enough — no need to re-run `init`.

## Uninstall

Always run `fundpulse uninstall` *before* removing the binary — it cleans up the launchd schedule.

If you installed via Homebrew:

```bash
fundpulse uninstall                    # remove launchd schedule
brew uninstall --cask fundpulse        # remove binary
```

If you installed from source:

```bash
make uninstall                           # both steps in one
```

Config and DB are preserved by design. Delete them yourself for a clean wipe:

```bash
rm ~/.config/fundpulse/config.yaml
rm ~/Library/Application\ Support/fundpulse/data.db
```

## Development

```bash
make build              # bin/fundpulse
make test               # go test ./...
make show               # build + fundpulse show
make dryrun             # build + fundpulse run --dry-run
```

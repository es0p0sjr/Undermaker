# Updating the Undercards snapshot

Use this workflow whenever the game changes. It does not require Codex. The browser captures the loaded catalog and images; the local updater validates, compares, builds, tests and installs them.

You need the **full project folder**, Go 1.23+, Node 18+ with npm, and Chrome or Edge. Keep `Sources/CardEditor` with the project because the build uses its local rendering assets. No `npm install` is needed. Someone with only `uc-print.exe` should obtain a newly built executable from a maintainer.

## 1. Prepare the capture

Open PowerShell in the project folder and run:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\update-snapshot.ps1 -Prepare
```

This regenerates `JSON/update/capture-snapshot.js` and copies it to the clipboard. If clipboard access is unavailable, copy the file's contents manually. Preparation makes no network requests.

## 2. Save a fresh capture in the browser

1. Open the Undercards collection in Chrome or Edge. Reload the page and wait for cards to appear.
2. Open **F12 → Sources → Snippets**, create a snippet, paste the helper and press **Ctrl+Enter**.
3. Click **Save complete capture**, choose the project's **JSON/update** folder, and allow the browser to write there.
4. Keep the tab open until **SUCCESS**. The panel shows the capture folder and the exact command to run next.

Each run creates a new `capture-<timestamp>` folder containing `AllCards.json` (card records, source and capture time), `en.json`, `assets/cards/` and `assets/powers/`.

The helper reads the game's `allCards` catalog; it does not request `/AllCards`, export your account, or change the game. A fresh folder prevents old images with unchanged filenames from being reused. If downloading stops, use **Resume** in the same panel. If the tab was closed, start a fresh capture. A 403 still requires a working browser session or a later retry; do not mix files from different captures.

## 3. Validate and install

Close UC Print if convenient. Paste the command shown by the capture panel, or replace `capture-TIMESTAMP` below with the actual folder name:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\update-snapshot.ps1 -Capture ".\JSON\update\capture-TIMESTAMP"
```

The updater:

1. Validates the installed snapshot, checks capture provenance/time, and requires the card IDs to match both the English name and rule entries.
2. Imports into a separate candidate folder, with networking disabled. Checks cover stats, resolved rules, image decoding and checksums.
3. Prints added/changed/removed counts and writes `comparison.json` with field changes and changed image paths.
4. Builds a separate executable and starts it from an empty working folder. Checks verify embedded cards, all standard card/status images, frontend availability, deck import and two-page PDF export.
5. Moves the previous snapshot and executable into a dated `snapshot-backups/` folder, then installs the tested pair. Failed replacements attempt to restore the previous pair automatically.

Wait for **UPDATE COMPLETE**, then launch `start.bat`. An already-running app still has its old data; close it and relaunch. If Windows prevents moving an open executable, the updater stops and rolls back; close the old app and retry.

All import/build/test activity is local. Only the browser capture downloads game resources. Do not run other update/import/build commands while the updater is running.

### Review without installing

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\update-snapshot.ps1 -Capture ".\JSON\update\capture-TIMESTAMP" -CheckOnly
```

This validates and compares but does not build or replace the installed snapshot/executable. Output identifies the `.build/snapshot-updates/<run>/` folder containing `comparison.json` and `import.log`.

Normal installation stops if existing card IDs disappear. Review the report and recapture if incomplete. If the game intentionally removed those cards, explicitly opt in:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\update-snapshot.ps1 -Capture ".\JSON\update\capture-TIMESTAMP" -AllowRemovals
```

This only permits removals; it does not bypass missing text, invalid images, stale timestamps or other validation. Resolve a catalog/English mismatch by reloading the collection and capturing again. Upstream format changes may require importer maintenance.

## Backups, logs and recovery

Successful updates retain:

```text
snapshot-backups/<run>/
  data/                previous complete snapshot (if one existed)
  uc-print.exe         previous executable (if one existed)
  comparison.json      changes installed by this run
  transaction.json     replacement paths and status

.build/snapshot-updates/<run>/
  import.log
  build.log
  smoke.log
  comparison.json
```

Backups are never automatically pruned. Keep at least the last known working pair. Original captures remain in `JSON/update/`. Failed/check-only candidates stay under `.build/snapshot-updates/`; remove them only when no updater is running and they are no longer needed for recovery.

**Validation, import, build or smoke-test failure:** read the error and log, fix the input/environment, and rerun. Installation has not started, so the old snapshot and executable remain in place.

**Replacement failure:** the journal says `rolled-back` if automatic recovery succeeded. A `recovery-required` status, or `installing` left after a crash/power loss, blocks the next update. Inspect the journal's paths before moving files: a crash can happen between a rename and the journal write, so `completed` may lag behind the actual files.

**Restore a successful backup:** close UC Print and all update/build commands. In File Explorer, rename the current project `data` folder to an unused name such as `data-before-restore`, and rename `dist/uc-print.exe` to `uc-print-before-restore.exe`. Copy the selected backup's `data` folder into the project root and its `uc-print.exe` into `dist`. Validate with `go run ./tools/ucsync validate`, then run `start.bat`. Keep displaced files until the restored pair works.

**Recover an interrupted replacement:** inspect the original/candidate/backup paths in `transaction.json`. The old snapshot may still be at `data` or already in the backup; likewise the old executable may still be in `dist` or already in the backup. Preserve every available copy, restore the old pair to `data` and `dist/uc-print.exe`, and validate it. Only after recovery succeeds, set that journal's `status` to `rolled-back`. If the old pair cannot be identified, seek help using the journal and logs.

**Stale lock:** `.data.ucsync.lock` prevents overlapping updates/imports. After a crash, check that no updater or `ucsync` process is running and handle unfinished transactions first. Then delete only that lock file and retry. Do not delete an active update's lock.

## Scope and maintenance

This workflow updates official standard artwork, names, stats, rules and status icons. Separate `ReworkedCards` files are not changed. If the installed snapshot has alternate/full-art variants, the updater stops before importing so they are not silently discarded. A maintainer must merge/reapply those variants with the local catalog using [artwork variant instructions](docs/artwork-variants.md), then validate/build the combined snapshot.

Physical print calibration and visual browser review remain manual. The smoke test checks local data, assets and PDF transport/layout; it does not render every card in a browser.

Run `go test ./...` and `npm test` (or `test.bat`) when maintaining the workflow. Update tests cover incomplete/stale captures, comparison results, failed replacement steps, rollback failure reporting, backup preservation and successful installation.

# Artwork downloads returning 403

Your saved AllCards and English JSON files solve the database/rules download. They do not contain the image bytes. Re-importing the same JSON cannot fix an image-server refusal.

The browser helper is an explicit part of `tools/ucsync`. It downloads only card/status images named by your dataset, one at a time, using your normal browser session. UC Print itself stays offline. It does not extract cookies, upload files or install an incomplete snapshot.

## Do this now

### Capture the loaded game when updating an older snapshot

**Use [the standard update workflow](../../UPDATING.md)** for routine updates. `update-snapshot.ps1 -Prepare` prepares and copies the helper; `update-snapshot.ps1 -Capture <folder>` validates, compares, builds, tests and installs with backups. The lower-level commands below are for manual maintenance and do not provide the complete coordinated workflow.

The September `undercards-print-data.json` exports recorded `capture.source: "loaded game"`. The original helper read the game's `allCards` variable. This approach works independently of the `/AllCards` endpoint as long as the game has loaded its catalog. The old `JSON/download-assets.js` only knows the September image list and cannot acquire new cards.

1. Generate the combined helper: `go run ./tools/ucsync browser-capture`. It writes `JSON/update/capture-snapshot.js` without any network requests.
2. In Chrome or Edge, open the game's collection and wait for its cards to appear. In **F12 > Sources > Snippets**, create a snippet containing `JSON/update/capture-snapshot.js` and run it with **Ctrl+Enter**.
3. Click **Save complete capture**, select this project's **JSON/update** folder, and leave the tab open until **SUCCESS**. The helper captures the loaded catalog, fetches English text and downloads every referenced card/status image. It creates a fresh `capture-<timestamp>` subfolder so older artwork is not silently reused. Stop/resume within the panel retains completed images. It makes no `/AllCards` request and does not fall back to an older browser cache.
4. Run the command printed in the capture panel. For a manual import/build instead, run the following commands after replacing `capture-TIMESTAMP` with the folder shown in the panel:

```powershell
$captureFolder = './JSON/update/capture-TIMESTAMP'
go run ./tools/ucsync import "$captureFolder/AllCards.json" --translations "$captureFolder/en.json" --assets "$captureFolder/assets"
if ($LASTEXITCODE -ne 0) { throw 'Import failed; build stopped' }
powershell -NoProfile -ExecutionPolicy Bypass -File ./build.ps1
if ($LASTEXITCODE -ne 0) { throw 'Build failed' }
```

The capture JSON includes its source and timestamp. It preserves the loaded card records; full rule/stat/image validation occurs during import. Check the captured catalog against the current English dictionary and previous snapshot before installation to detect a stale or partial catalog. The installed snapshot remains unchanged until a complete import passes. Close the running UC Print executable before rebuilding, then restart `start.bat`.

The earlier `browser-data.js` helper requests `/AllCards` directly and was also refused in the browser. Use the loaded-game helper above for this case. `en.json` alone supplies text, not updated costs/stats or complete new card records.

### Download artwork for the saved JSON

**UC Print does not need to be running.** If `start.bat` says setup is incomplete or `manifest.json` is missing, proceed with the steps below. The helper runs on the Undercards website. The final offline import creates the manifest automatically; do not create one by hand. `ucsync validate` only checks an existing snapshot.

1. In your usual **Chrome or Edge**, open <https://undercards.net/images/cards/ACT_Button.png>. If the browser also shows 403, stop: this helper cannot guarantee access either. Wait for access to return or obtain a matching `cards/` and `powers/` asset folder from a browser/machine that can access the images. Do not repeatedly run the importer.
2. Open <https://undercards.net> in that browser and leave the tab open.
3. Open `JSON/download-assets.js` in your editor and copy its contents. In the Undercards tab press **F12**, select **Sources**, then **Snippets** (under the left sidebar's `>>` or **More tabs** menu), then **New snippet**. Paste the helper into the snippet, and press **Ctrl+Enter** to run it. This is a reviewable local script, not a command to paste into UC Print's webpage. [Chrome's Snippets instructions](https://developer.chrome.com/docs/devtools/javascript/snippets/).
4. A **UC Print** panel appears on the webpage. Click **Choose assets folder** and select this project's **`JSON/assets`** folder. Allow the requested folder write access. Keep the tab open until **SUCCESS**. Downloading all the images may take several minutes. Stop/resume is supported: choose the same folder and it reuses valid files. On 403 or an invalid image it stops and retains completed files.
5. Back in PowerShell, from the project folder, run:

```powershell
go run ./tools/ucsync import .\JSON\AllCards.Json --translations .\JSON\en.json --assets .\JSON\assets
if ($LASTEXITCODE -ne 0) { throw 'Import failed; build stopped' }
powershell -NoProfile -ExecutionPolicy Bypass -File .\build.ps1
if ($LASTEXITCODE -ne 0) { throw 'Build failed' }
.\dist\uc-print.exe --open
```

`--assets` makes that import fully offline. It checks every image, rules record and checksum before installing anything. The browser's SUCCESS message is not a replacement for this validation.

## Prepare again after replacing the JSON

The helper already generated in `JSON/download-assets.js` matches your current 931-card input. Regenerate it whenever the dataset changes:

```powershell
$env:GOCACHE = Join-Path (Get-Location) '.go-cache'
go run ./tools/ucsync browser-assets .\JSON\AllCards.Json --translations .\JSON\en.json --output .\JSON\download-assets.js
```

This command reads local JSON, resolves rules and writes a helper with the exact required filenames. It makes no network requests and does not modify `data/` or the input JSON. Images are stored as `JSON/assets/cards/<filename>` and `JSON/assets/powers/<filename>`. Select a separate empty assets folder for a fresh image revision rather than reusing old artwork with the same filenames.

The folder picker requires a user click on a secure webpage in a supporting browser; see [the folder picker API](https://developer.mozilla.org/en-US/docs/Web/API/Window/showDirectoryPicker). The helper is not included in the app's frontend or release executable.

Automated tests cover helper generation and simulated downloads/resume/errors. Download success against Undercards from your browser still needs to be checked; a browser helper cannot promise to remove a server-side 403.

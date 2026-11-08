# Rework Studio

A separate, offline companion for the physical card reworks. Double-click
`start-reworks.bat` in the project root. Keep its console open while using it.
It chooses an available local port, so UC Print can run alongside it.

The 15 starting cards were transcribed from `ReworkedCards/*.png`. They are
marked **Needs review**: compare the editable text and stats with each reference,
then check the review box and save. Inherited official status icons that are
absent from the references were removed. Original artwork, rarity and expansion
are reused. Rules such as Armor and Program remain in the transcribed text.
The shared renderer displays tribe badges from the JSON using the original
CardEditor icons. Rules text remains plain; keyword colors are not yet reproduced.

Select a card to edit its name, cost, attack, health, rules, rarity, expansion,
and status icons/counters. **Save rework** writes to `ReworkedCards/reworks.json`.
The original PNGs and official snapshot are never written by the studio.
**Download reworks JSON** makes a backup of saved edits. To restore a backup,
close the studio and replace `ReworkedCards/reworks.json` with that file.

**Download PNG** exports the current editor at 744 × 1039 with 300 DPI metadata.
Set quantities in the sidebar, or choose **One of each**, then **Export A4 PDF**.
PDF export uses saved reworks and supports up to 250 cards with optional cut
marks. Cards measure 63 × 88 mm, nine per A4 page. Print at Actual Size / 100%.
Download PDFs before closing the studio; only the last five remain in memory.

This first version edits the existing 15-card collection. Adding new reference
files does not automatically transcribe or register them. Deck-code import,
new-card creation, and colored rules editing are not included.

The companion reads `data/`, `web/renderer.js`, and `web/renderer-assets.json`
from this project. Its executable must remain with the project; it is not a
self-contained distribution. The old application does not need to run.

To rebuild only the companion, from the project root:

```powershell
$env:GOCACHE = Join-Path (Get-Location) '.go-cache'
$env:GOPROXY = 'off'
$env:GOTOOLCHAIN = 'local'
go build -o dist/rework-studio.exe ./ReworkStudio
```

Development launch: `go run ./ReworkStudio --root . --open`.
Tests: `go test ./ReworkStudio`.

Browser verification: `python ReworkStudio/browser_check.py`. This renders all
15 cards, checks quantity controls, and exports a PNG and two-page PDF to
`output/`. Automated Chrome startup was blocked by Windows process permissions
in the development sandbox; run this from a normal terminal, or verify the
previews and one exported sheet manually.

# Alternate and full artwork

Open a card preview and use **Artwork** to choose an installed skin. Choices are
remembered in this browser and apply to all queued copies of that card, sheet
previews, PNG downloads and PDFs. Card IDs, deck codes, rules and stats stay the same.

The current real snapshot contains standard images only. Official skin catalog
acquisition is still pending: no verified catalog or skin files were available
when this feature was added. The importer does not guess official endpoints,
filenames, card mappings or skin types.

To install artwork you have saved locally, prepare a JSON array like this
(illustrative filenames; use actual card IDs and your actual files):

```json
[
  {"cardId": 642, "id": "skin-1", "name": "Alternate artwork", "image": "skin-1.png", "layout": "alternate"},
  {"cardId": 642, "id": "skin-2", "name": "Full artwork", "image": "skin-2.png", "layout": "full"}
]
```

Place those PNG/JPG files in `Assets/cards/` alongside the existing artwork. Use
unique filenames to avoid overwriting standard images. Then import and rebuild:

```powershell
go run ./tools/ucsync import ./JSON/AllCards.Json --translations ./JSON/en.json --assets ./Assets --art-variants ./JSON/art-variants.json
if ($LASTEXITCODE -ne 0) { throw 'Import failed' }
powershell -NoProfile -ExecutionPolicy Bypass -File ./build.ps1
```

Include `--art-variants` whenever rebuilding the snapshot to retain the skins.
The catalog is preserved verbatim under `data/raw/art-variants.json`. Missing,
corrupt or unsafe image references, duplicate variant IDs and unknown card IDs
fail validation and leave the previous snapshot intact. All images are embedded
in the executable; the app never downloads artwork.

`alternate` fits the image in the usual artwork window. `full` fills the card
behind translucent rules and stat panels, cropping the image centrally to fit.
This is a print layout, not an exact reproduction of the game's skin frame.
Use artwork without baked-in rules or stats to avoid duplicate text. Animated
skins require a still PNG/JPG. Original image resolution limits print sharpness.

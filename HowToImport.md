# Importing complete reworked cards

1. Keep `reworks.json` and the finished card PNGs in `ReworkedCards`.
2. Start the updated UC Print with `start.bat` or `dist/uc-print.exe`.
3. Click **Import reworked PNGs**. The library switches to **Reworked PNGs**.
4. Add individual cards with quantities, or click **One of each rework**.
5. Choose **Preview sheets** or **Export PDF**. Print on A4 at **100% / Actual Size**.

Cards measure 63 × 88 mm, with nine per A4 sheet and optional cut marks. You can mix official and reworked cards by changing the **Collection** filter. Importing preserves the current print queue; **One of each rework** adds one copy per click. Loading a deck code still replaces the queue with official cards.

The importer reads each entry's `card.id`, `card.name`, other library metadata, and `reference`:

```json
[
  {
    "card": {
      "id": 222,
      "name": "Blue Snail",
      "type": "MONSTER",
      "rarity": "COMMON",
      "description": "Optional text used for library searches"
    },
    "reference": "Blue Snail.png"
  }
]
```

Each ID must be a unique positive integer within this file. The `reference` must be a PNG filename in the same folder. The existing JSON already has these fields; no conversion is needed. `card.image` is artwork metadata and is not used for printing these complete fronts.

The entire PNG is fitted inside the card area without cropping or stretching. Its frame, text, colors, and badges are printed as supplied; JSON rules and stats are not drawn over it. Small source images retain their original detail when enlarged, and the preview reports their resolution. The review checkbox in Rework Studio does not block printing the supplied PNG.

To add more cards, add their PNGs and entries to this JSON, then click **Import reworked PNGs** again. Invalid JSON, duplicate IDs, missing files, or damaged PNGs produce an error and leave the previous imported collection available. Successful imports and quantities are restored when reopening the same browser address while the folder remains available.

UC Print finds `ReworkedCards` in its working folder, beside its executable, or one directory above the executable (the project's `dist` layout). Keep this folder with the application when moving it to another computer; custom PNGs are not embedded during the build.

Rework Studio remains a separate text/stats editor. Its edits update JSON but do not redraw the complete PNGs used by this import feature. Update the PNG itself when you want its printed appearance to change.

To regenerate the complete PNG collection from the reviewed JSON, run `python scripts/regenerate_reworks.py`. This uses the shared renderer and an installed Chrome/Edge browser, writes 744 × 1039 PNGs with 300 DPI metadata plus an A4 PDF under a timestamped `output/reworked-cards-300dpi-*` folder, and validates the PNG/PDF pixels. It backs up existing images and JSON under `ReworkedCards/originals-*` before replacing the referenced PNGs. Click **Import reworked PNGs** again in an already-open UC Print window to load the replacements. Tribe badges are included; rules text still uses plain colors. The JSON is not modified.

For the project's installed headless-shell setup, the script also accepts `--browser PATH --single-process`.

Developer browser check: `python scripts/browser_test.py dist/uc-print.exe tests/reworked-cards.js` (adjust the executable path for other builds).

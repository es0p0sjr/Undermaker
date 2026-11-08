# UC Print

UC Print turns a **validated local snapshot** of Undercards into printable card fronts. The browser and Go backend work offline. Only an explicit `tools/ucsync` command can download game data.

**Start with [TODO.md](TODO.md).** It includes a ready-to-run fictional demo, real-data import commands, build instructions, and a print checklist.

The application provides a searchable card library, Base64/JSON deck-code import, quantities, a separate token pool, individual 744 x 1039 PNGs with 300 DPI metadata, and A4 PDFs with nine 63 x 88 mm cards and optional cut marks. Use Actual Size / 100% when printing.

Build prerequisites: Go 1.23+, Node 18+ and npm. No third-party Go or npm packages and no `npm install` are needed. The resulting `dist/uc-print.exe` embeds the frontend, normalized card database, artwork, status icons and manifest; the recipient needs only the executable and a browser. Python is optional for extra testing and the compatibility launcher.

Double-clicking the executable opens the app in your browser. Keep its console window open while using the app. If the default port (8765) is occupied, the new copy uses an available local port and opens its own address. An explicit `--port` stays fixed; use `--port 0 --open` to request an available port yourself. Launch errors remain visible until you press Enter when starting without arguments.

```text
Explicit ucsync download/import
  -> data/.staging (exact raw JSON + normalized JSON + all required assets)
  -> validation
  -> install complete snapshot, preserving the old one on failure
  -> build.ps1 (validate -> frontend build -> Go embed/build)
  -> uc-print.exe (offline)
```

The real snapshot captured on November 3, 2026 is installed in this workspace: 951 cards, 951 artwork files and 19 status icons, with English rules resolved during import. It adds 20 cards and updates 87 existing cards compared with the September snapshot, with no removed cards. Run `start.bat` or `dist/uc-print.exe --open` to use it. The included demo contains **four fictional cards and geometric test artwork**, clearly labelled in the app and on the cards. Artifacts and SOUL reference cards remain pending verified complete source data; their deck metadata is retained with a visible warning. Monsters, spells and token cards are supported now.

See [data-source notes](docs/data-source.md) for the adapter contract and source investigation, and [validation notes](docs/validation.md) for checks and test coverage.

To update the snapshot, follow [UPDATING.md](UPDATING.md). Run `update-snapshot.ps1 -Prepare`, capture the loaded game with the browser snippet, then run `update-snapshot.ps1 -Capture <folder>`. The updater validates and compares the capture, builds and tests a candidate executable, and installs it with a dated backup. No Codex session is required. `-CheckOnly` reviews a capture without installing it. Downloading `en.json` alone cannot update card stats or add complete new cards.

Alternate and full-art images can be imported locally and selected in each card's preview. See [artwork variants](docs/artwork-variants.md). The current snapshot still contains standard artwork only; official skin files must be acquired separately.

Complete reworked card PNGs can be printed directly: choose **Import reworked PNGs** to load `ReworkedCards/reworks.json`, then add cards or choose **One of each rework**. The `reference` field identifies each finished PNG in that folder. Previews and exports use the complete image, preserving its text and frame. Official and reworked versions can share a print sheet. Keep `ReworkedCards` in the project folder or beside the executable. See [custom-card import](HowToImport.md).

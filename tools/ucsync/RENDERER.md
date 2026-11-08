# Download the existing renderer resources

**Installed in the app:** the 52 source files are present and their display resources now drive card previews, PNGs and PDFs. Launch `start.bat`, check the **CARDEDITOR STYLE** badge, and open `http://127.0.0.1:8765/?card=771` to check ACT Button. No further download is needed.

`npm run build` verifies and packages these local files automatically. For development, `go run ./tools/ucsync renderer-bundle` regenerates `web/renderer-assets.json` without network access. The executable embeds the finished bundle.

To regenerate the comparison after changing inputs:

```powershell
go run ./tools/ucsync renderer-preview
```

The comparison intentionally reveals what the unadapted source supplies. In the live app, a small adapter positions local status icons over the artwork. Decorative corners are still absent from the source. The card template is extracted and specialized without executing the editor's JavaScript or loading its external dependencies.

Your card database, English rules and 947 card/power images are already installed. The following acquisition instructions are only needed when setting up a fresh checkout or obtaining a newer source version.

Run this in PowerShell from the project folder:

```powershell
go run ./tools/ucsync renderer-download
```

On success, the source files are saved to `Sources/CardEditor/`. You do not need to start UC Print or run the previous artwork helper.

## Browser fallback

The coding workspace could not reach GitHub because of network restrictions. If the command also fails in your own terminal:

1. [Download CardEditor-master.zip](https://github.com/UCProjects/CardEditor/archive/refs/heads/master.zip). Alternatively open [the repository](https://github.com/UCProjects/CardEditor) and choose **Code > Download ZIP**.
2. Save it in this project's `Sources` folder as **CardEditor.zip**. Keep it zipped.
3. Run:

```powershell
go run ./tools/ucsync renderer-import ./Sources/CardEditor.zip
```

The ZIP importer works offline. It preserves the archive, extracts selected source files, checks required files and rarity images, and records SHA-256 checksums. It refuses to overwrite a previous bundle; use `--output Sources/CardEditor-new` for a later revision.

Expected files include:

```text
Sources/CardEditor/
  source.zip
  source-manifest.json
  public/
    resources/
      DTM-Mono.otf
      undercards.css
      style.css
      modules/card.js
    rarity/
      TOKEN.png
      COMMON.png
      ...
```

The saved editor JavaScript is not executed and no third-party packages are installed. Current-game Loop placement and decorative corners may still need the game's own CSS/assets. See `docs/rendering-sources.md` for implementation and source notes.

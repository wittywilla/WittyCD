# Changelog

What changed in WittyCD, newest first. What changed before it had a repository of its own is in the
[WittyMediaServer changelog](https://github.com/wittywilla/WittyMediaServer/blob/master/CHANGELOG.md).

## 0.3.4

- **Updated dependencies:** esbuild from 0.27.7 to 0.28.2.

## 0.3.3

- **No computer keys on a television.** In TV mode the CD player no longer lists the keyboard's shortcut keys. They still work, and a computer still lists them.

## Developer notes for 0.3.3

- `html.tv-mode .cdp-keyhelp` is hidden, as the server's TV mode does for its own key legends (docs/TV-MODE.md).
- `readRawWindow` no longer takes an extent it never used; an unused `os.Lstat` result in `resolveBeside` is dropped (staticcheck SA4009, SA4006).
- `scripts/test-web.mjs` imports the built bundle by file URL, so `npm test` also runs on Windows.

## 0.3.2

- **Fixed: CD rips with a cue sheet** play when the sheet names a .wav that the folder has as .flac or another sound file. A sheet naming a file that isn't there is skipped quietly.

## 0.3.1

- The first published release on module API 5, with the WittyModule 0.5.4 workflows. The tag v0.3.0 was made with workflows that couldn't build it and was never published.

## 0.3.0

- **WittyCD 0.3.0: module API 5.** WittyCD owns the CD player: its page in the web app (`/cd/:id`, the web part's `register`), and the engine that plays a disc through the app's audio element (the app's audio-engine slot), moved from the server file for file. Finding which album a disc is has its own routes, `GET /api/modules/wittycd/cds/{id}/matches` and `POST /api/modules/wittycd/cds/{id}/identify` (administrators), which ask the server's host API: the server still asks MusicBrainz and fetches the cover, so the process that reads an image never has a network. On every app, the TV apps among them, the page `wittycd-identify` does the same, from its source `wittycd.matches` and its command `wittycd.identify`. The one program is both the helper for one image and the module's service. Built on WittyModule 0.5.4 and module API 5.
- **WittyCD 0.2.1:** built for module API 4: it carries `sdui.json` (its theme scope, for the theme editor), kept equal to `UI()` by a test. Its pages stay the server's own disc pages.
- **WittyCD 0.2.1:** for servers up to 0.8.x. Relicensed to AGPL-3.0-or-later, with the licence header on every source file.
- A helper, `wittycd-module`, is now how a server reaches WittyCD, with no copy compiled in.
- A cue sheet can split a sound file in any format: a host with FFmpeg sets `ForeignAudio` to make FLAC of what this package doesn't read itself.
- WittyCD is now a project of its own, developed and released from this repository.

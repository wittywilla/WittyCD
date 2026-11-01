# WittyCD

Plays images of audio CDs in WittyMediaServer the way a CD player does: one continuous program, with no gap between tracks. You need it if you keep CD images in a music or disc library.

## What it does

- **Reads images:** `.cue` with `.bin`, `.wav`, `.aiff` or a single `.flac`; `.toc`; `.ccd` with `.img`; `.nrg`; `.chd`; and a bare `.iso`, `.img` or `.bin` that is really audio.
- **Knows the layout:** tracks, indexes, pregaps, sessions, data tracks, pre-emphasis flags, ISRCs, the barcode, CD-TEXT and CD+G.
- **Plays without gaps.** A disc is one stream, and track changes are places in it. Chosen tracks and repeats play the same way.
- **Streams FLAC or WAV.** WAV takes no encoding. FLAC uses its own small encoder.
- **Corrects pre-emphasis** on flagged tracks, and gives the same samples when you seek as when you play through.
- **Gets album details.** MusicBrainz and the Cover Art Archive are asked only by the disc's table of contents. Finding names online, and how a library shows a disc, stay with the server.
- **Reports problems.** An image whose audio is scrambled shows up as noise and is reported, and so is a DTS surround disc. Nothing is decrypted or bypassed.

It doesn't read these:

- Compressed audio other than a single FLAC (`.mp3`, `.ape` and so on). A sheet beside per-track FLAC files isn't treated as an image either, since those are tagged songs already.
- Raw `.sub` files for CD+G. Use a `.cdg` file beside the image.
- CD-TEXT in double-byte character sets, which is reported instead.

CCD and NRG are tested against images built to their published descriptions, not against dumps from the original programs. CHD and TOC are checked against the real tools.

How to use it is in **Help** inside WittyMediaServer: see Music, audiobooks and podcasts.

## How to get it

An admin adds it from the module store inside WittyMediaServer: **Settings → Modules**, then **Browse the store**, then **Add** beside WittyCD. That needs WittyMediaServer 0.6.0-alpha or later, and it works with nothing older. The range this version fits is in `module.json`: 0.6.0-alpha through 0.6.x.

## What it needs

- [WittyModule](https://github.com/wittywilla/WittyModule), which is how a server talks to it, built in. Nothing else.
- No music comes with it.

It's written in Go with no cgo, from the publicly described formats.

## Platforms

A program is built for each system below, and the server uses the right one.

| System | Where it stands |
|---|---|
| Linux (x64, arm64, 32-bit ARM, x86) | Its tests pass on x64. The other CPUs are only built |
| Windows (x64, arm64) | Built. This version hasn't been run on Windows since it became its own program |
| macOS (Intel, Apple silicon) | Built by the release tool. It has not been run on a real Mac |
| FreeBSD (amd64, arm64) | Cross-compiled only. It has not been run |

On macOS and FreeBSD the server has no sandbox for a module's helper: it gets a small environment and a soft memory limit, and nothing more.

## Building and testing

You need Go 1.26 or newer. In a worktree, first run the copy of `dev-modules.mjs` kept above the repositories (`node D:/projects/dev-modules.mjs` on the development machine). It writes a `go.work` that git ignores, so the checkout builds against the repositories beside it. Never run `go work use`.

```bash
go vet ./...
```

```bash
go test ./... -count=1
```

```bash
go run github.com/wittywilla/wittymodule/cmd/wittymodule check .
```

The last one checks the manifest. On the same tool, `pack -targets linux/amd64 -out release .` builds a release for one system. CI also builds for every system.

To look at an image by hand:

```bash
go run ./cmd/wittycd info "Album.cue"
```

```bash
go run ./cmd/wittycd check "Album.cue"
```

`info` shows the structure. `check` reads the whole image and says what's wrong.

To use it as a library, `wittycd.Open` gives a `Disc`. `NewStream`, `TrackStream` and `RangeStream` give a stream readable at any offset by any number of readers, with `WriteFLAC` and `WAV`. `TOC` gives `MusicBrainzID`, `LookupString` and `CDDBID`. The `lookup` package asks MusicBrainz and the Cover Art Archive, and has an `Attribution` to show wherever the data is used.

## Licence

Apache-2.0: see [LICENSE](LICENSE) and [NOTICE](NOTICE). If you redistribute it or build on it, keep the NOTICE attribution. It contains no code from GPL libraries. Disc IDs follow the method published by [MusicBrainz](https://musicbrainz.org/doc/Disc_ID_Calculation), which isn't connected with this project. Each release archive carries a `THIRD-PARTY-NOTICES.txt` for what its programs link.

## Legal

WittyCD is an independent open-source project. It isn't associated with, sponsored by or endorsed by
the owners of the Compact Disc format, any record label or artist, or MusicBrainz. It isn't a licensed
Compact Disc product and uses no disc logo. It reads CD images you already have: no music comes with it,
and you're responsible for having the rights to what you open. Album details come from the MusicBrainz
and Cover Art Archive services, which are asked only by a disc's table of contents. Any third-party names
here only describe what it works with, and belong to their owners. It's provided “as is”, with no
warranty; see [LICENSE](LICENSE). Local law applies to how you use it. This isn't legal advice.

<!-- module-kit -->
## Releasing

A version tag builds a program for every platform, signs the checksum list, and publishes a release that servers can install.

Versions move in small steps as work lands. `go run github.com/wittywilla/wittymodule/cmd/wittymodule@v0.3.0 bump .` raises it
everywhere it is written (add `-part minor` for a bigger step). A bump isn't a release: nothing is
tagged or published until you tag it. CI fails when code changed since the last tag and the version
didn't, because tags never move.

To release:

1. Check that `version` in `module.json` is the one to publish, with the bump committed.
2. Tag that commit and push the tag: `git tag vX.Y.Z`, then `git push origin vX.Y.Z`. A tag with a
   suffix, such as `v1.2.0-beta`, is published as a prerelease.
3. Watch the *Actions* tab. The signing step uses the `WMS_SIGNING_KEY` secret of the `release` environment, and nothing is published without it.

To take a version back, run the *withdraw* workflow from the *Actions* tab. It adds the version to a signed list on the release tagged `withdrawals`, which servers read.

Licensed under Apache-2.0: see [LICENSE](LICENSE) and [NOTICE](NOTICE).
<!-- /module-kit -->

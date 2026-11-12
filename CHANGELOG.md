# Changelog

What changed in WittyCD, newest first. What changed before it had a repository of its own is in the
[WittyMediaServer changelog](https://github.com/wittywilla/WittyMediaServer/blob/master/CHANGELOG.md).

## Unreleased

- **WittyCD 0.2.1:** for servers up to 0.8.x. Relicensed to AGPL-3.0-or-later, with the licence header on every source file.
- A helper, `wittycd-module`, is now how a server reaches WittyCD, with no copy compiled in.
- A cue sheet can split a sound file in any format: a host with FFmpeg sets `ForeignAudio` to make FLAC of what this package doesn't read itself.
- WittyCD is now a project of its own, developed and released from this repository.
